package messages

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewMessage tests the constructor for the Message model.
func TestNewMessage(t *testing.T) {
	t.Run("Valid Message", func(t *testing.T) {
		content := "Hello, World!"
		recipient := "+1234567890"
		charLimit := 160
		msg, err := NewMessage(content, recipient, charLimit)

		assert.NoError(t, err)
		assert.NotNil(t, msg)
		assert.NotEmpty(t, msg.ID)
		_, err = uuid.Parse(msg.ID)
		assert.NoError(t, err)
		assert.Equal(t, content, msg.Content)
		assert.Equal(t, recipient, msg.Recipient)
		assert.Equal(t, "pending", msg.Status)
	})

	t.Run("Empty Recipient", func(t *testing.T) {
		_, err := NewMessage("Test", "", 160)
		assert.Error(t, err)
		assert.Equal(t, ErrRecipientEmpty, err)
	})

	t.Run("Content Too Long", func(t *testing.T) {
		_, err := NewMessage("This content is definitely too long.", "recipient", 10)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrContentTooLong)
	})
}

// TestMessageStateTransitions tests the state transition methods of the Message model.
func TestMessageStateTransitions(t *testing.T) {
	msg := &Message{
		ID:        "test-id",
		Status:    "pending",
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("MarkAsSending", func(t *testing.T) {
		initialTime := msg.UpdatedAt
		msg.MarkAsSending()
		assert.Equal(t, "sending", msg.Status)
		assert.True(t, msg.UpdatedAt.After(initialTime))
	})

	t.Run("MarkAsSent", func(t *testing.T) {
		initialTime := msg.UpdatedAt
		externalID := "ext-123"
		msg.MarkAsSent(externalID)
		assert.Equal(t, "sent", msg.Status)
		assert.NotNil(t, msg.ExternalMessageID)
		assert.Equal(t, externalID, *msg.ExternalMessageID)
		assert.Nil(t, msg.LastFailureReason)
		assert.True(t, msg.UpdatedAt.After(initialTime))
	})

	t.Run("MarkAsFailed", func(t *testing.T) {
		initialTime := msg.UpdatedAt
		reason := "webhook timeout"
		msg.MarkAsFailed(reason)
		assert.Equal(t, "failed", msg.Status)
		assert.NotNil(t, msg.LastFailureReason)
		assert.Equal(t, reason, *msg.LastFailureReason)
		assert.True(t, msg.UpdatedAt.After(initialTime))
	})
}
