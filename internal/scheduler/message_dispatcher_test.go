package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/akshaysangma/go-notify/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockMessageService is a mock implementation of the MessageDispatchScheduler interface.
type MockMessageService struct {
	mock.Mock
}

func (m *MockMessageService) FetchAndSendPending(ctx context.Context, limit int) error {
	args := m.Called(ctx, limit)
	// Simulate work
	if delay, ok := ctx.Value("delay").(time.Duration); ok {
		time.Sleep(delay)
	}
	return args.Error(0)
}

func TestScheduler_StartStop(t *testing.T) {
	mockService := new(MockMessageService)
	logger := zap.NewNop()
	// Use a long interval to prevent the ticker from firing during this test.
	cfg := config.SchedulerConfig{RunsEvery: 1 * time.Hour}
	scheduler := NewMessageDispatchSchedulerImpl(mockService, logger, cfg)

	// Test initial state
	assert.False(t, scheduler.IsRunning(), "Scheduler should not be running initially")

	// Test Start
	err := scheduler.Start()
	assert.NoError(t, err)
	assert.True(t, scheduler.IsRunning(), "Scheduler should be running after Start()")

	// Test starting an already running scheduler
	err = scheduler.Start()
	assert.Error(t, err)
	assert.Equal(t, ErrAlreadyRunning, err)

	// Test Stop
	err = scheduler.Stop()
	assert.NoError(t, err)
	assert.False(t, scheduler.IsRunning(), "Scheduler should be stopped after Stop()")

	// Test stopping an already stopped scheduler
	err = scheduler.Stop()
	assert.Error(t, err)
	assert.Equal(t, ErrNotRunning, err)
}

func TestScheduler_LoopExecution(t *testing.T) {
	mockService := new(MockMessageService)
	logger := zap.NewNop()
	// Use a very short interval for quick testing.
	cfg := config.SchedulerConfig{
		RunsEvery:   50 * time.Millisecond,
		MessageRate: 10,
		GracePeriod: 10 * time.Millisecond,
	}
	scheduler := NewMessageDispatchSchedulerImpl(mockService, logger, cfg)

	// Expect FetchAndSendPending to be called.
	// We use a channel to wait for the call to happen.
	callSignal := make(chan struct{})
	mockService.On("FetchAndSendPending", mock.Anything, cfg.MessageRate).Return(nil).Run(func(args mock.Arguments) {
		// Signal that the method was called.
		// Use a non-blocking send in case the test times out first.
		select {
		case callSignal <- struct{}{}:
		default:
		}
	})

	err := scheduler.Start()
	assert.NoError(t, err)

	// Wait for the mock to be called, with a timeout.
	select {
	case <-callSignal:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for FetchAndSendPending to be called")
	}

	err = scheduler.Stop()
	assert.NoError(t, err)
}

func TestScheduler_SkipOverlapExecution(t *testing.T) {
	mockService := new(MockMessageService)
	logger := zap.NewNop()
	cfg := config.SchedulerConfig{
		RunsEvery:   50 * time.Millisecond,
		MessageRate: 10,
		GracePeriod: 10 * time.Millisecond,
	}
	scheduler := NewMessageDispatchSchedulerImpl(mockService, logger, cfg)

	// The first call will be slow, causing the second tick to be skipped.
	// The third tick should proceed as normal.
	callCount := 0
	mockService.On("FetchAndSendPending", mock.Anything, cfg.MessageRate).Return(nil).Run(func(args mock.Arguments) {
		callCount++
		if callCount == 1 {
			// Make the first call take longer than the tick interval.
			ctx := args.Get(0).(context.Context)
			ctx = context.WithValue(ctx, "delay", 60*time.Millisecond)
			args[0] = ctx
		}
	}).Twice() // Expect it to be called twice (1st and 3rd tick), not three times.

	scheduler.Start()

	// Let the scheduler run for a duration that covers multiple ticks.
	time.Sleep(120 * time.Millisecond)

	scheduler.Stop()

	// Assert that the mock was called exactly twice.
	mockService.AssertNumberOfCalls(t, "FetchAndSendPending", 2)
}
