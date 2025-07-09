package scheduler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/akshaysangma/go-notify/internal/config"
	"go.uber.org/zap"
)

var (
	// ErrAlreadyRunning is returned when trying to start an already running scheduler.
	ErrAlreadyRunning = errors.New("scheduler is already running")
	// ErrNotRunning is returned when trying to stop a scheduler that is not running.
	ErrNotRunning = errors.New("scheduler is not running")
)

// MessageService defines the interface for the message service that the scheduler will use.
type MessageDispatchScheduler interface {
	FetchAndSendPending(ctx context.Context, limit int) error
}

type MessageDispatchSchedulerImpl struct {
	messageService MessageDispatchScheduler
	logger         *zap.Logger
	config         config.SchedulerConfig
	workerPoolSize int           // max allowed is 2 * runtime.NumCPU() for I/O ops
	isProcessing   atomic.Bool   // state representing in flight status
	isRunning      atomic.Bool   // state representing schedule running status
	stopChan       chan struct{} // chan to signal graceful shutdown of scheduler
	wg             sync.WaitGroup
}

func NewMessageDispatchSchedulerImpl(service MessageDispatchScheduler,
	logger *zap.Logger,
	config config.SchedulerConfig) *MessageDispatchSchedulerImpl {

	return &MessageDispatchSchedulerImpl{
		messageService: service,
		logger:         logger,
		config:         config,
		stopChan:       make(chan struct{}),
	}
}

// Start begins the scheduler's main loop in a new goroutine.
// It is safe to call Start multiple times; it will only start if not already running.
func (s *MessageDispatchSchedulerImpl) Start() error {
	if !s.isRunning.CompareAndSwap(false, true) {
		s.logger.Warn("Scheduler is already running.")
		return ErrAlreadyRunning
	}

	s.stopChan = make(chan struct{})
	s.wg.Add(1)
	go s.loop()

	s.logger.Info("Scheduler started successfully.",
		zap.Duration("runs_every", s.config.RunsEvery),
		zap.Int("allowed_message_rate", s.config.MessageRate),
		zap.Int("worker_count", s.workerPoolSize),
	)

	return nil
}

// IsRunning returns the current running state of the scheduler.
func (s *MessageDispatchSchedulerImpl) IsRunning() bool {
	return s.isRunning.Load()
}

// Stop gracefully shuts down the scheduler.
func (s *MessageDispatchSchedulerImpl) Stop() error {
	if !s.isRunning.CompareAndSwap(true, false) {
		s.logger.Warn("Scheduler is not running.")
		return ErrNotRunning
	}

	close(s.stopChan)
	s.wg.Wait()
	s.logger.Info("Scheduler stopped gracefully.")
	return nil
}

// loop is the main loop for the scheduler.
func (s *MessageDispatchSchedulerImpl) loop() {
	defer s.wg.Done()
	// uncomment below if delay for first set of message processing is undesirable
	// s.execute()
	ticker := time.NewTicker(s.config.RunsEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.execute()
		case <-s.stopChan:
			s.logger.Info("Stop signal received, shutting down scheduler loop.")
			return
		}
	}
}

// execute handles a single ticker event.
func (s *MessageDispatchSchedulerImpl) execute() {
	if !s.isProcessing.CompareAndSwap(false, true) {
		s.logger.Warn("Skipping tick, previous processing run is still active.")
		return
	}
	defer s.isProcessing.Store(false)

	s.logger.Info("Ticker triggered, starting message processing batch.")

	// Calculate the deadline for this batch.
	processingTimeout := s.config.RunsEvery - s.config.GracePeriod

	batchCtx, cancel := context.WithTimeout(context.Background(), processingTimeout)
	defer cancel()

	err := s.messageService.FetchAndSendPending(batchCtx, s.config.MessageRate)
	if err != nil {
		// Check if the error was due to our intentional cancellation.
		if errors.Is(err, context.DeadlineExceeded) {
			s.logger.Warn("Message processing timed out and was gracefully cancelled. Messages will be retried on the next tick.")
		} else {
			s.logger.Error("An unexpected error occurred during message processing.", zap.Error(err))
		}
	} else {
		s.logger.Info("Message processing batch completed successfully.")
	}
}
