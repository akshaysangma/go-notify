package database

import (
	"context"
	"fmt"

	"github.com/akshaysangma/go-notify/internal/database/sqlc"
	"github.com/akshaysangma/go-notify/internal/messages"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrPoolType = fmt.Errorf("bad pgx pool type")
)

type PostgresMessageRepository struct {
	queries *sqlc.Queries
	pool    PgxPoolInterface // for TX
}

func NewPostgresMessageRepository(pool PgxPoolInterface) (*PostgresMessageRepository, error) {
	if dBTX, ok := pool.(sqlc.DBTX); ok {
		return &PostgresMessageRepository{
			queries: sqlc.New(dBTX),
			pool:    pool,
		}, nil
	}
	return nil, fmt.Errorf("unable to convert pool to dBTX")
}

// mapDBMessageToDomain converts a sqlc.Message to a messages.Message domain model.
func mapDBPendingMessageToDomain(dbMsg *sqlc.GetPendingMessagesRow) (*messages.Message, error) {
	msg := &messages.Message{
		ID:        dbMsg.ID.String(),
		Content:   dbMsg.Content,
		Recipient: dbMsg.RecipientPhoneNumber,
		Status:    string(dbMsg.Status),
		CreatedAt: dbMsg.CreatedAt,
		UpdatedAt: dbMsg.UpdatedAt,
	}

	if dbMsg.ExternalMessageID.Valid {
		msg.ExternalMessageID = &dbMsg.ExternalMessageID.String
	}

	return msg, nil
}

// mapDBMessageToDomain converts a sqlc.Message to a messages.Message domain model.
func mapDBSentMessageToDomain(dbMsg *sqlc.GetAllSentMessagesRow) (*messages.Message, error) {
	msg := &messages.Message{
		ID:        dbMsg.ID.String(),
		Content:   dbMsg.Content,
		Recipient: dbMsg.RecipientPhoneNumber,
		Status:    string(dbMsg.Status),
		CreatedAt: dbMsg.CreatedAt,
		UpdatedAt: dbMsg.UpdatedAt,
	}

	if dbMsg.ExternalMessageID.Valid {
		msg.ExternalMessageID = &dbMsg.ExternalMessageID.String
	}

	return msg, nil
}

func (r *PostgresMessageRepository) GetPendingMessages(ctx context.Context, limit int32) ([]messages.Message, error) {
	pendingMsgs, err := r.queries.GetPendingMessages(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("fail to fetch pending messages from db: %w", err)
	}
	var msgs []messages.Message
	for _, dbMsg := range pendingMsgs {
		msg, err := mapDBPendingMessageToDomain(&dbMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to map db message to domain for ID %s: %w", dbMsg.ID.String(), err)
		}
		msgs = append(msgs, *msg)
	}
	return msgs, nil
}

func (r *PostgresMessageRepository) UpdateMessageStatus(ctx context.Context, msg messages.Message) error {
	updateParams := sqlc.UpdateMessageStatusParams{
		Status: sqlc.NotificationsMessageStatus(msg.Status),
		ID:     uuid.MustParse(msg.ID),
	}

	if msg.ExternalMessageID != nil {
		updateParams.ExternalMessageID = pgtype.Text{String: *msg.ExternalMessageID, Valid: true}
	} else {
		updateParams.ExternalMessageID = pgtype.Text{Valid: false}
	}

	if msg.LastFailureReason != nil {
		updateParams.LastFailureReason = pgtype.Text{String: *msg.LastFailureReason, Valid: true}
	} else {
		updateParams.LastFailureReason = pgtype.Text{Valid: false}
	}

	err := r.queries.UpdateMessageStatus(ctx, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update Message Status: %w", err)
	}
	return nil
}

func (r *PostgresMessageRepository) GetSentMessages(ctx context.Context, limit, offset int32) ([]messages.Message, error) {
	sentMsgs, err := r.queries.GetAllSentMessages(ctx, sqlc.GetAllSentMessagesParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("fail to fetch all sent messages: %w", err)
	}
	var msgs []messages.Message
	for _, dbMsg := range sentMsgs {
		msg, err := mapDBSentMessageToDomain(&dbMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to map db message to domain for ID %s: %w", dbMsg.ID.String(), err)
		}
		msgs = append(msgs, *msg)
	}
	return msgs, nil
}

func (r *PostgresMessageRepository) CreateMessages(ctx context.Context, msgs []*messages.Message) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	for _, msg := range msgs {
		_, err := qtx.CreateMessage(ctx, sqlc.CreateMessageParams{
			ID:                   uuid.MustParse(msg.ID),
			Content:              msg.Content,
			RecipientPhoneNumber: msg.Recipient,
		})
		if err != nil {
			return fmt.Errorf("failed to create message for recipient %s: %w", msg.Recipient, err)
		}
	}

	return tx.Commit(ctx)
}
