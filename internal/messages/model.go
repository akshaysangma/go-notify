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
	ID                string    `json:"id"`
	Content           string    `json:"content"`
	Recipient         string    `json:"recipient"`
	Status            string    `json:"status"`
	ExternalMessageID *string   `json:"external_message_id"`
	LastFailureReason *string   `json:"last_failure_reason"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// NewMessage is a constructor for creating a new Message, enforcing domain invariants.
func NewMessage(content, recipient string, charLimit int) (*Message, error) {
	if recipient == "" {
		return nil, ErrRecipientEmpty
	}
	if len(content) > charLimit {
		return nil, ErrContentTooLong
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
