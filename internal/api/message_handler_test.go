package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akshaysangma/go-notify/internal/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockMessageService is a mock of the MessageServicer Interface.
type MockMessageService struct {
	mock.Mock
}

func (m *MockMessageService) GetAllSentMessages(ctx context.Context, limit, offset int32) ([]messages.Message, error) {
	args := m.Called(ctx, limit, offset)
	// Handle nil case for messages
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]messages.Message), args.Error(1)
}

func (m *MockMessageService) CreateMessages(ctx context.Context, content string, recipients []string, charLimit int) error {
	args := m.Called(ctx, content, recipients, charLimit)
	return args.Error(0)
}

func TestMessageHandler_getSentMessages(t *testing.T) {
	mockService := new(MockMessageService)
	handler := NewMessageHandler(mockService, 250, zap.NewNop())

	t.Run("Success", func(t *testing.T) {
		expectedMessages := []messages.Message{{ID: "1", Status: "sent"}}
		mockService.On("GetAllSentMessages", mock.Anything, int32(20), int32(0)).Return(expectedMessages, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/sent", nil)
		rr := httptest.NewRecorder()

		handler.getSentMessages(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var body []messages.Message
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, expectedMessages, body)
		mockService.AssertExpectations(t)
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		serviceErr := errors.New("database is down")
		mockService.On("GetAllSentMessages", mock.Anything, int32(20), int32(0)).Return(nil, serviceErr).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/sent", nil)
		rr := httptest.NewRecorder()

		handler.getSentMessages(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		var body HTTPError
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "Failed to retrieve sent messages", body.Error)
		assert.Contains(t, body.Details, serviceErr.Error())
		mockService.AssertExpectations(t)
	})
}

func TestMessageHandler_createMessages(t *testing.T) {
	mockService := new(MockMessageService)
	handler := NewMessageHandler(mockService, 250, zap.NewNop())

	t.Run("Success - Accepted", func(t *testing.T) {
		recipients := []string{"+12345"}
		content := "hello world"
		mockService.On("CreateMessages", mock.Anything, content, recipients, 250).Return(nil).Once()

		reqBody := CreateMessagesRequest{
			Content:    content,
			Recipients: recipients,
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()

		handler.createMessages(rr, req)

		assert.Equal(t, http.StatusAccepted, rr.Code)
		var body SuccessResponse
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "Messages accepted for creation.", body.Message)
		mockService.AssertExpectations(t)
	})

	t.Run("Bad Request - Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBufferString("{not_json}"))
		rr := httptest.NewRecorder()

		handler.createMessages(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockService.AssertNotCalled(t, "CreateMessages")
	})

	t.Run("Bad Request - Service Validation Error", func(t *testing.T) {
		validationErr := messages.ErrContentTooLong
		mockService.On("CreateMessages", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(validationErr).Once()

		reqBody := CreateMessagesRequest{Content: "too long", Recipients: []string{"+1"}}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()

		handler.createMessages(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		var body HTTPError
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid message data", body.Error)
		assert.Contains(t, body.Details, validationErr.Error())
		mockService.AssertExpectations(t)
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		serviceErr := errors.New("db insert failed")
		mockService.On("CreateMessages", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(serviceErr).Once()

		reqBody := CreateMessagesRequest{Content: "content", Recipients: []string{"+1"}}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBuffer(jsonBody))
		rr := httptest.NewRecorder()

		handler.createMessages(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		var body HTTPError
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "Could not create messages", body.Error)
		assert.Contains(t, body.Details, serviceErr.Error())
		mockService.AssertExpectations(t)
	})
}
