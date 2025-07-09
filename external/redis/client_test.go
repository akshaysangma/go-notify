package redis

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// MockRedisClientInterface is a mock implementation of the RedisClientInterface.
type MockRedisClientInterface struct {
	mock.Mock
}

func (m *MockRedisClientInterface) Ping(ctx context.Context) *redis.StatusCmd {
	args := m.Called(ctx)
	return args.Get(0).(*redis.StatusCmd)
}

func (m *MockRedisClientInterface) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(*redis.StatusCmd)
}

func TestRedisService_CacheSentMessage(t *testing.T) {
	observerCore, recordedLogs := observer.New(zap.DebugLevel)
	mockLogger := zap.New(observerCore)

	mockClient := new(MockRedisClientInterface)
	service := &RedisService{
		client: mockClient,
		logger: mockLogger,
	}

	ctx := context.Background()
	messageID := "test-msg-123"
	externalMessageID := "ext-abc-456"
	sentAt := time.Date(2025, 7, 9, 10, 0, 0, 0, time.UTC)
	expectedKey := "sent_messages:test-msg-123"
	expectedValue := "ext_id:%s;sent_at:%s"
	expectedExpiration := 24 * time.Hour

	t.Run("Success - message cached", func(t *testing.T) {
		statusCmd := redis.NewStatusCmd(ctx)
		// Simulate successful Set operation
		statusCmd.SetVal("OK")
		mockClient.On("Set", ctx, expectedKey, fmt.Sprintf(expectedValue, externalMessageID, sentAt.Format(time.RFC3339)), expectedExpiration).Return(statusCmd).Once()

		err := service.CacheSentMessage(ctx, messageID, externalMessageID, sentAt)
		assert.NoError(t, err)
		assert.Equal(t, 1, recordedLogs.Len())
		assert.Equal(t, "Successfully cached sent message", recordedLogs.All()[0].Message)
		mockClient.AssertExpectations(t)
		recordedLogs.TakeAll()
	})

	t.Run("Error - failed to cache message", func(t *testing.T) {
		setErr := errors.New("redis set error")
		statusCmd := redis.NewStatusCmd(ctx)
		// Simulate failed Set operation
		statusCmd.SetErr(setErr)
		mockClient.On("Set", ctx, expectedKey, fmt.Sprintf(expectedValue, externalMessageID, sentAt.Format(time.RFC3339)), expectedExpiration).Return(statusCmd).Once()

		err := service.CacheSentMessage(ctx, messageID, externalMessageID, sentAt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to cache message test-msg-123")
		assert.Contains(t, err.Error(), setErr.Error())
		assert.Equal(t, 1, recordedLogs.Len())
		assert.Equal(t, zapcore.ErrorLevel, recordedLogs.All()[0].Level)
		assert.Equal(t, "Failed to cache sent message in Redis", recordedLogs.All()[0].Message)
		mockClient.AssertExpectations(t)
		recordedLogs.TakeAll()
	})

	t.Run("Warn - client not initialized", func(t *testing.T) {
		uninitializedService := &RedisService{
			// Simulate uninitialized client
			client: nil,
			logger: mockLogger,
		}

		err := uninitializedService.CacheSentMessage(ctx, messageID, externalMessageID, sentAt)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not initialized")
		assert.Equal(t, 1, recordedLogs.Len())
		assert.Equal(t, zapcore.WarnLevel, recordedLogs.All()[0].Level)
		assert.Equal(t, "Redis client is not initialized, cannot cache message", recordedLogs.All()[0].Message)
		// Ensure Set was not called
		mockClient.AssertNotCalled(t, "Set")
		recordedLogs.TakeAll()
	})
}
