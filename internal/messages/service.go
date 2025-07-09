package messages

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// WebhookSender defines the contract for sending messages to an external webhook service.
type WebhookSender interface {
	Send(ctx context.Context, to, content string) (externalMessageID string, err error)
}

// CacheService defines the contract for caching sent messages.
type CacheService interface {
	CacheSentMessage(ctx context.Context, messageID, externalMessageID string, sentAt time.Time) error
}

// MessageService implements the core business logic for message handling.
type MessageService struct {
	repo         MessageRepository
	webhook      WebhookSender
	logger       *zap.Logger
	cacheService CacheService
	workerCount  int
	jobTimeout   time.Duration
}

func NewMessageService(
	repo MessageRepository,
	webhook WebhookSender,
	logger *zap.Logger,
	cacheService CacheService,
	workerCount int,
	jobTimeout time.Duration,
) *MessageService {
	return &MessageService{
		repo:         repo,
		webhook:      webhook,
		logger:       logger,
		cacheService: cacheService,
		workerCount:  workerCount,
		jobTimeout:   jobTimeout,
	}
}

// FetchAndSendPending is called by the scheduler. It fetches pending messages
// and uses a worker pool to process and send them concurrently.
func (s *MessageService) FetchAndSendPending(ctx context.Context, limit int) error {
	s.logger.Info("Fetching pending messages to process.", zap.Int("limit", limit))
	pendingMsgs, err := s.repo.GetPendingMessages(ctx, int32(limit))
	if err != nil {
		return fmt.Errorf("failed to get pending messages: %w", err)
	}

	if len(pendingMsgs) == 0 {
		s.logger.Info("No pending messages to process.")
		return nil
	}

	jobs := make(chan Message, len(pendingMsgs))
	var wg sync.WaitGroup

	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go s.worker(ctx, &wg, i+1, jobs)
	}

	for _, msg := range pendingMsgs {
		jobs <- msg
	}
	close(jobs)

	wg.Wait()
	s.logger.Info("Finished processing message batch.", zap.Int("processed_count", len(pendingMsgs)))
	return nil
}

// worker represents a single routine that processes messages from the jobs channel.
func (s *MessageService) worker(ctx context.Context, wg *sync.WaitGroup, id int, jobs <-chan Message) {
	defer wg.Done()
	s.logger.Info("Worker started", zap.Int("worker_id", id))
	for msg := range jobs {

		// Stop processing if the parent context is cancelled.
		if ctx.Err() != nil {
			s.logger.Warn("Context cancelled, worker stopping early.", zap.Int("worker_id", id), zap.Error(ctx.Err()))
			return
		}
		// Create a new context with the per-job timeout.
		jobCtx, cancel := context.WithTimeout(context.Background(), s.jobTimeout)
		defer cancel()
		if err := s.sendMessage(jobCtx, msg); err != nil {
			s.logger.Error("Worker failed to send message",
				zap.Int("worker_id", id),
				zap.String("message_id", msg.ID),
				zap.Error(err),
			)
		}
	}
	s.logger.Info("Worker finished", zap.Int("worker_id", id))
}

func (s *MessageService) sendMessage(ctx context.Context, msg Message) error {
	logFields := []zap.Field{
		zap.String("message_id", msg.ID),
		zap.String("recipient", msg.Recipient),
	}
	s.logger.Info("Attempting to send message", logFields...)

	// Mark the message as 'sending' to prevent other workers from picking it up.
	msg.MarkAsSending()
	if err := s.repo.UpdateMessageStatus(ctx, msg); err != nil {
		s.logger.Error("Failed to mark message as 'sending'", append(logFields, zap.Error(err))...)
		return fmt.Errorf("failed to update status to sending for message %s: %w", msg.ID, err)
	}

	externalMessageID, webhookErr := s.webhook.Send(ctx, msg.Recipient, msg.Content)
	if webhookErr != nil {
		s.logger.Error("Failed to send message via webhook", append(logFields, zap.Error(webhookErr))...)
		msg.MarkAsFailed(fmt.Sprintf("webhook send failed: %v", webhookErr))
		// Avoid shadowing the original webhookErr.
		if updateErr := s.repo.UpdateMessageStatus(ctx, msg); updateErr != nil {
			s.logger.Error("Failed to update message status to 'failed'", append(logFields, zap.Error(updateErr))...)
		}
		return fmt.Errorf("failed to send message %s: %w", msg.ID, webhookErr)
	}

	s.logger.Info("Message successfully sent via webhook, marking as 'sent' in DB",
		append(logFields, zap.String("external_id", externalMessageID))...)

	msg.MarkAsSent(externalMessageID)
	// If the webhook send succeeded but this DB update fails, the message remains
	// in the 'sending' state and will be retried, which is the desired behavior.
	if err := s.repo.UpdateMessageStatus(ctx, msg); err != nil {
		s.logger.Error("Failed to mark message as 'sent' in DB after successful send", append(logFields, zap.Error(err))...)
		return fmt.Errorf("failed to mark message %s as sent in DB: %w", msg.ID, err)
	}

	s.logger.Info("Message successfully processed and marked as sent",
		append(logFields, zap.String("external_id", externalMessageID))...)

	// Caching is a best-effort operation; run it in the background with a timeout.
	cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if cacheErr := s.cacheService.CacheSentMessage(cacheCtx, msg.ID, externalMessageID, time.Now()); cacheErr != nil {
		s.logger.Warn("Failed to cache sent message",
			zap.String("message_id", msg.ID),
			zap.String("external_id", externalMessageID),
			zap.Error(cacheErr),
		)
	}

	return nil
}

// GetAllSentMessages take limit and offset to return paginated sent message from database.
func (s *MessageService) GetAllSentMessages(ctx context.Context, limit, offset int32) ([]Message, error) {
	s.logger.Debug("Attempting to retrieve sent messages", zap.Int32("limit", limit), zap.Int32("offset", offset))
	msgs, err := s.repo.GetSentMessages(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to retrieve sent messages", zap.Error(err), zap.Int32("limit", limit), zap.Int32("offset", offset))
		return nil, fmt.Errorf("failed to get sent messages: %w", err)
	}
	s.logger.Debug("Successfully retrieved sent messages", zap.Int("count", len(msgs)))
	return msgs, nil
}

// CreateMessages insert a message for multiple recipients in the database
func (s *MessageService) CreateMessages(ctx context.Context, content string, recipients []string, charLimit int) error {
	var msgsToCreate []*Message
	for _, recipient := range recipients {
		msg, err := NewMessage(content, recipient, charLimit)
		if err != nil {
			return fmt.Errorf("invalid message for recipients %v: %w", recipients, err)
		}
		msgsToCreate = append(msgsToCreate, msg)
	}

	if len(msgsToCreate) == 0 {
		return nil
	}

	err := s.repo.CreateMessages(ctx, msgsToCreate)
	if err != nil {
		s.logger.Error("Failed to bulk insert messages", zap.Error(err))
		return fmt.Errorf("could not save messages: %w", err)
	}

	s.logger.Info("Successfully created messages for multiple recipients", zap.Int("count", len(msgsToCreate)))
	return nil
}
