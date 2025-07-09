package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/akshaysangma/go-notify/internal/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRoundTripper is a mock implementation of http.RoundTripper.
type MockRoundTripper struct {
	mock.Mock
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestWebhookSiteSender_Send(t *testing.T) {
	mockRT := new(MockRoundTripper) // Our mock for RoundTripper
	// Create an http.Client that uses our mock RoundTripper
	mockClient := &http.Client{
		Transport: mockRT,
		Timeout:   5 * time.Second, // Match sender's timeout
	}

	sender := NewWebhookSiteSender("http://example.com/webhook", 250, 5*time.Second)
	sender.client = mockClient // Inject the http.Client with the mock Transport

	ctx := context.Background()
	to := "+1234567890"
	content := "Hello, World!"
	expectedMessageID := "webhook-msg-123"

	t.Run("Success - message sent", func(t *testing.T) {
		responseBody := WebhookResponse{MessageID: expectedMessageID, Status: "accepted"}
		jsonBody, _ := json.Marshal(responseBody)
		req, _ := http.NewRequest(http.MethodPost, "http://example.com/webhook", nil) // Create a dummy request for resp.Request
		resp := &http.Response{
			StatusCode: http.StatusAccepted,
			Body:       io.NopCloser(bytes.NewBuffer(jsonBody)),
			Header:     make(http.Header),
			Request:    req, // Must provide a request to avoid nil pointer dereference
		}

		mockRT.On("RoundTrip", mock.AnythingOfType("*http.Request")).Return(resp, nil).Once()

		messageID, err := sender.Send(ctx, to, content)
		assert.NoError(t, err)
		assert.Equal(t, expectedMessageID, messageID)
		mockRT.AssertExpectations(t)
	})

	t.Run("Error - content too long", func(t *testing.T) {
		longContent := "a"
		for i := 0; i < 251; i++ {
			longContent += "a"
		}
		_, err := sender.Send(ctx, to, longContent)
		assert.Error(t, err)
		assert.ErrorIs(t, err, messages.ErrContentTooLong)
		mockRT.AssertNotCalled(t, "RoundTrip") // Ensure no HTTP call was made
	})

	t.Run("Error - recipient empty", func(t *testing.T) {
		_, err := sender.Send(ctx, "", content)
		assert.Error(t, err)
		assert.ErrorIs(t, err, messages.ErrRecipientEmpty)
		mockRT.AssertNotCalled(t, "RoundTrip") // Ensure no HTTP call was made
	})

	t.Run("Error - http client returns error", func(t *testing.T) {
		clientErr := errors.New("network error")
		// Always return a non-nil *http.Response even on error to prevent panic in net/http
		req, _ := http.NewRequest(http.MethodPost, "http://example.com/webhook", nil) // Create a dummy request
		resp := &http.Response{
			StatusCode: http.StatusInternalServerError, // Or any appropriate status for a network error
			Body:       io.NopCloser(bytes.NewBufferString("")),
			Request:    req, // Must provide the request object
		}
		mockRT.On("RoundTrip", mock.AnythingOfType("*http.Request")).Return(resp, clientErr).Once()

		_, err := sender.Send(ctx, to, content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send webhook request")
		assert.ErrorIs(t, err, clientErr)
		mockRT.AssertExpectations(t)
	})

	t.Run("Error - webhook returns non-202 status", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "http://example.com/webhook", nil) // Create a dummy request
		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString(`{"error":"invalid recipient"}`)),
			Header:     make(http.Header),
			Request:    req,
		}
		mockRT.On("RoundTrip", mock.AnythingOfType("*http.Request")).Return(resp, nil).Once()

		_, err := sender.Send(ctx, to, content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook responded with non-200 status code: 400")
		assert.Contains(t, err.Error(), `body: {"error":"invalid recipient"}`)
		mockRT.AssertExpectations(t)
	})

	t.Run("Error - invalid webhook response JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "http://example.com/webhook", nil) // Create a dummy request
		resp := &http.Response{
			StatusCode: http.StatusAccepted,
			Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
			Header:     make(http.Header),
			Request:    req,
		}
		mockRT.On("RoundTrip", mock.AnythingOfType("*http.Request")).Return(resp, nil).Once()

		_, err := sender.Send(ctx, to, content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode webhook response body")
		mockRT.AssertExpectations(t)
	})

	t.Run("Error - webhook response missing message ID", func(t *testing.T) {
		responseBody := WebhookResponse{Status: "accepted"} // Missing MessageID
		jsonBody, _ := json.Marshal(responseBody)
		req, _ := http.NewRequest(http.MethodPost, "http://example.com/webhook", nil) // Create a dummy request
		resp := &http.Response{
			StatusCode: http.StatusAccepted,
			Body:       io.NopCloser(bytes.NewBuffer(jsonBody)),
			Header:     make(http.Header),
			Request:    req,
		}
		mockRT.On("RoundTrip", mock.AnythingOfType("*http.Request")).Return(resp, nil).Once()

		_, err := sender.Send(ctx, to, content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook response did not contain a message ID")
		mockRT.AssertExpectations(t)
	})
}
