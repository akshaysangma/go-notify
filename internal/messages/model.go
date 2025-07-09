package messages

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Domain-specific errors.
var (
	ErrContentTooLong = fmt.Errorf("message content exceeds character limit")
	ErrRecipientEmpty = fmt.Errorf("recipient cannot be empty")
)

// Message represents the message entity in the domain.
type Message struct {
	// The unique identifier for the message.
	ID string `json:"id" example:"a1b2c3d4-e5f6-7890-1234-567890abcdef"`
	// The content of the message to be sent. Should not exceed content length limit.
	Content string `json:"content" example:"Your appointment is confirmed."`
	// The phone number of the recipient.
	Recipient string `json:"recipient" example:"+15551234567"`
	// The current status of the message.
	Status string `json:"status" example:"sent"`
	// The ID returned from the external webhook service.
	ExternalMessageID *string `json:"external_message_id,omitempty" example:"ext-msg-12345"`
	// The reason for the last failure, if any.
	LastFailureReason *string `json:"last_failure_reason,omitempty" example:"Webhook provider timed out"`
	// The timestamp when the message was created.
	CreatedAt time.Time `json:"created_at" example:"2025-07-09T10:00:00Z"`
	// The timestamp when the message was last updated.
	UpdatedAt time.Time `json:"updated_at" example:"2025-07-09T10:01:00Z"`
}

// NewMessage is a constructor for creating a new Message, enforcing domain invariants.
func NewMessage(content, recipient string, charLimit int) (*Message, error) {
	if recipient == "" {
		return nil, ErrRecipientEmpty
	}

	if len(content) > charLimit {
		return nil, fmt.Errorf("%w, limit : %v", ErrContentTooLong, charLimit)
	}

	return &Message{
		ID:        uuid.New().String(),
		Content:   content,
		Recipient: recipient,
		Status:    "pending",
	}, nil
}

// MarkAsSending updates the message status to 'sending'.
func (m *Message) MarkAsSending() {
	m.Status = "sending"
	m.UpdatedAt = time.Now().UTC()
}

// MarkAsSent updates the message status to 'sent' and stores the external ID.
func (m *Message) MarkAsSent(externalID string) {
	m.Status = "sent"
	m.ExternalMessageID = &externalID
	m.LastFailureReason = nil
	m.UpdatedAt = time.Now().UTC()
}

// MarkAsFailed updates the message status to 'failed' and records the reason.
func (m *Message) MarkAsFailed(reason string) {
	m.Status = "failed"
	m.LastFailureReason = &reason
	m.UpdatedAt = time.Now().UTC()
}
