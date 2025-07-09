package messages

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockMessageRepository is a mock of MessageRepository
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) GetPendingMessages(ctx context.Context, limit int32) ([]Message, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]Message), args.Error(1)
}

func (m *MockMessageRepository) UpdateMessageStatus(ctx context.Context, msg Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockMessageRepository) GetSentMessages(ctx context.Context, limit, offset int32) ([]Message, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]Message), args.Error(1)
}

func (m *MockMessageRepository) CreateMessages(ctx context.Context, msgs []*Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

// MockWebhookSender is a mock of WebhookSender
type MockWebhookSender struct {
	mock.Mock
}

func (m *MockWebhookSender) Send(ctx context.Context, to, content string) (string, error) {
	args := m.Called(ctx, to, content)
	return args.String(0), args.Error(1)
}

// MockCacheService is a mock of CacheService
type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) CacheSentMessage(ctx context.Context, messageID, externalMessageID string, sentAt time.Time) error {
	args := m.Called(ctx, messageID, externalMessageID, sentAt)
	return args.Error(0)
}

func TestMessageService_FetchAndSendPending(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	mockWebhook := new(MockWebhookSender)
	mockCache := new(MockCacheService)
	logger := zap.NewNop()
	service := NewMessageService(mockRepo, mockWebhook, logger, mockCache, 2, 10*time.Second)

	pendingMsg := Message{ID: "msg1", Content: "test", Recipient: "+123", Status: "pending"}

	t.Run("Success Case", func(t *testing.T) {
		mockRepo.On("GetPendingMessages", mock.Anything, int32(10)).Return([]Message{pendingMsg}, nil).Once()
		mockRepo.On("UpdateMessageStatus", mock.Anything, mock.MatchedBy(func(m Message) bool {
			return m.ID == pendingMsg.ID && m.Status == "sending"
		})).Return(nil).Once()
		mockWebhook.On("Send", mock.Anything, pendingMsg.Recipient, pendingMsg.Content).Return("ext-123", nil).Once()
		mockRepo.On("UpdateMessageStatus", mock.Anything, mock.MatchedBy(func(m Message) bool {
			return m.ID == pendingMsg.ID && m.Status == "sent"
		})).Return(nil).Once()
		mockCache.On("CacheSentMessage", mock.Anything, pendingMsg.ID, "ext-123", mock.Anything).Return(nil).Once()

		err := service.FetchAndSendPending(context.Background(), 10)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockWebhook.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("No Pending Messages", func(t *testing.T) {
		mockRepo.On("GetPendingMessages", mock.Anything, int32(5)).Return([]Message{}, nil).Once()

		err := service.FetchAndSendPending(context.Background(), 5)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		// Ensure other mocks were not called
		mockWebhook.AssertNotCalled(t, "Send")
	})

	t.Run("Webhook Fails", func(t *testing.T) {
		webhookErr := errors.New("webhook failed")
		mockRepo.On("GetPendingMessages", mock.Anything, int32(1)).Return([]Message{pendingMsg}, nil).Once()
		mockRepo.On("UpdateMessageStatus", mock.Anything, mock.MatchedBy(func(m Message) bool {
			return m.ID == pendingMsg.ID && m.Status == "sending"
		})).Return(nil).Once()
		mockWebhook.On("Send", mock.Anything, pendingMsg.Recipient, pendingMsg.Content).Return("", webhookErr).Once()
		mockRepo.On("UpdateMessageStatus", mock.Anything, mock.MatchedBy(func(m Message) bool {
			return m.ID == pendingMsg.ID && m.Status == "failed"
		})).Return(nil).Once()

		err := service.FetchAndSendPending(context.Background(), 1)
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
		mockWebhook.AssertExpectations(t)
		mockCache.AssertNotCalled(t, "CacheSentMessage")
	})
}

func TestMessageService_GetAllSentMessages(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	service := NewMessageService(mockRepo, nil, zap.NewNop(), nil, 0, 0)

	t.Run("Success", func(t *testing.T) {
		expectedMessages := []Message{{ID: "1", Status: "sent"}}
		mockRepo.On("GetSentMessages", mock.Anything, int32(10), int32(0)).Return(expectedMessages, nil).Once()

		msgs, err := service.GetAllSentMessages(context.Background(), 10, 0)
		assert.NoError(t, err)
		assert.Equal(t, expectedMessages, msgs)
		mockRepo.AssertExpectations(t)
	})
	t.Run("Empty List", func(t *testing.T) {
		mockRepo.On("GetSentMessages", mock.Anything, int32(10), int32(0)).Return([]Message{}, nil).Once()

		msgs, err := service.GetAllSentMessages(context.Background(), 10, 0)
		assert.NoError(t, err)
		assert.Empty(t, msgs)
		mockRepo.AssertExpectations(t)
	})
	t.Run("Repository Fails", func(t *testing.T) {
		repoErr := errors.New("db error")
		mockRepo.On("GetSentMessages", mock.Anything, int32(10), int32(0)).Return([]Message(nil), repoErr).Once()

		_, err := service.GetAllSentMessages(context.Background(), 10, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), repoErr.Error())
		mockRepo.AssertExpectations(t)
	})
}

func TestMessageService_CreateMessages(t *testing.T) {
	mockRepo := new(MockMessageRepository)
	service := NewMessageService(mockRepo, nil, zap.NewNop(), nil, 0, 0)

	t.Run("Success", func(t *testing.T) {
		recipients := []string{"+111", "+222"}
		content := "hello"
		mockRepo.On("CreateMessages", mock.Anything, mock.MatchedBy(func(msgs []*Message) bool {
			return len(msgs) == 2 && msgs[0].Recipient == "+111"
		})).Return(nil).Once()

		err := service.CreateMessages(context.Background(), content, recipients, 100)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid Content", func(t *testing.T) {
		recipients := []string{"+111"}
		content := "too long"
		err := service.CreateMessages(context.Background(), content, recipients, 5)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContentTooLong)
		mockRepo.AssertNotCalled(t, "CreateMessages")
	})

	t.Run("Repository Fails", func(t *testing.T) {
		repoErr := errors.New("db error")
		recipients := []string{"+111"}
		content := "hello"
		mockRepo.On("CreateMessages", mock.Anything, mock.Anything).Return(repoErr).Once()

		err := service.CreateMessages(context.Background(), content, recipients, 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), repoErr.Error())
		mockRepo.AssertExpectations(t)
	})
}
