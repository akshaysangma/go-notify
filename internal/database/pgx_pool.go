package database

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// PgxPoolInterface defines the methods of pgxpool.Pool used by PostgresMessageRepository.
type PgxPoolInterface interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}
