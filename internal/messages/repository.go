package messages

import "context"

// MessageRepository defines the contract on Message entities.
type MessageRepository interface {
	// GetPendingMessages retrieves a batch of unsent messages, up to the specified limit.
	GetPendingMessages(ctx context.Context, limit int32) ([]Message, error)

	// UpdateMessageStatus updates a message's status to sent and records its external message ID.
	UpdateMessageStatus(ctx context.Context, msg Message) error

	// GetSentMessages retrieves a paginated list of sent messages.
	GetSentMessages(ctx context.Context, limit, offset int32) ([]Message, error)

	// CreateMessages batch-inserts new messages into the database.
	CreateMessages(ctx context.Context, msgs []*Message) error
}
