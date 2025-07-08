package messages

import (
	"context"
	"fmt"
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
}

// NewMessageService creates a new MessageService instance.
func NewMessageService(
	repo MessageRepository,
	webhook WebhookSender,
	logger *zap.Logger,
	cacheService CacheService,
) *MessageService {
	return &MessageService{
		repo:         repo,
		webhook:      webhook,
		logger:       logger,
		cacheService: cacheService,
	}
}

// GetPendingMessages retrieves a batch of Pending messages.
func (s *MessageService) GetPendingMessages(ctx context.Context, limit int32) ([]Message, error) {
	s.logger.Debug("Attempting to retrieve pending messages", zap.Int32("limit", limit))
	msgs, err := s.repo.GetPendingMessages(ctx, limit)
	if err != nil {
		s.logger.Error("Failed to retrieve pending messages", zap.Error(err), zap.Int32("limit", limit))
		return nil, fmt.Errorf("failed to get pending messages: %w", err)
	}
	s.logger.Debug("Successfully retrieved pending messages", zap.Int("count", len(msgs)))
	return msgs, nil
}

func (s *MessageService) SendMessage(ctx context.Context, msg Message) error {
	logFields := []zap.Field{
		zap.String("message_id", msg.ID),
		zap.String("recipient", msg.Recipient),
	}
	s.logger.Info("Attempting to send message", logFields...)

	externalMessageID, err := s.webhook.Send(ctx, msg.Recipient, msg.Content)
	if err != nil {
		s.logger.Error("Failed to send message via webhook", append(logFields, zap.Error(err))...)
		msg.MarkAsFailed(fmt.Sprintf("failed to send message via webhook: %v", err))
		err := s.repo.UpdateMessageStatus(ctx, msg)
		if err != nil {
			s.logger.Error("Failed update message status to failed", append(logFields, zap.Error(err))...)
		}
		return fmt.Errorf("failed to send message %s: %w", msg.ID, err)
	}

	s.logger.Info("Message successfully sent via webhook, marking as sent in DB",
		append(logFields, zap.String("external_id", externalMessageID))...)

	msg.MarkAsSent(externalMessageID)
	err = s.repo.UpdateMessageStatus(ctx, msg)
	if err != nil {
		s.logger.Error("Failed to mark message as sent in DB after successful send", append(logFields, zap.Error(err))...)
		msg.MarkAsFailed(fmt.Sprintf("failed to update message status in db: %v", err))
		err := s.repo.UpdateMessageStatus(ctx, msg)
		if err != nil {
			s.logger.Error("Failed update message status to failed", append(logFields, zap.Error(err))...)
		}
		return fmt.Errorf("failed to mark message %s as sent in DB: %w", msg.ID, err)
	}
	s.logger.Info("Message successfully processed and marked as sent",
		append(logFields, zap.String("external_id", externalMessageID))...)

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
