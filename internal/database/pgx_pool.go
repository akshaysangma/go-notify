package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var (
	// ErrPoolType is returned when database conn pool cant converted to DBTX.
	ErrPoolType = fmt.Errorf("bad pgx pool type")
)

// PgxPoolInterface defines the methods of pgxpool.Pool used by PostgresMessageRepository.
// Helps with Mocking and Dependency inversion.
type PgxPoolInterface interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}
