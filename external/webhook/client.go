package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/akshaysangma/go-notify/internal/messages"
)

// WebhookRequest represents the payload for the webhook.
type WebhookRequest struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

// WebhookResponse represents the expected response from the webhook.
type WebhookResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Error     string `json:"error"`
}

// WebhookSiteSender implements the messages.WebhookSender interface.
type WebhookSiteSender struct {
	client         *http.Client
	webhookURL     string
	characterLimit int
}

func NewWebhookSiteSender(url string, charLimit int, timeout time.Duration) *WebhookSiteSender {
	return &WebhookSiteSender{
		client: &http.Client{
			Timeout: timeout,
		},
		webhookURL:     url,
		characterLimit: charLimit,
	}
}

func (s *WebhookSiteSender) Send(ctx context.Context, to, content string) (string, error) {
	if len(content) > s.characterLimit {
		return "", messages.ErrContentTooLong
	}
	if to == "" {
		return "", messages.ErrRecipientEmpty
	}

	requestBody := WebhookRequest{
		To:      to,
		Content: content,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal webhook request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("webhook responded with non-200 status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var webhookResp WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhookResp); err != nil {
		return "", fmt.Errorf("failed to decode webhook response body: %w", err)
	}

	if webhookResp.MessageID == "" {
		return "", fmt.Errorf("webhook response did not contain a message ID")
	}

	return webhookResp.MessageID, nil
}
