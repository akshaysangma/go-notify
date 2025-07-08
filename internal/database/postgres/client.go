package postgres

import (
	"context"
	"fmt"

	"github.com/akshaysangma/go-notify/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresDB(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	connConfig, err := pgxpool.ParseConfig(cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database connection string: %w", err)
	}

	connConfig.MaxConns = cfg.MaxConns

	pool, err := pgxpool.NewWithConfig(ctx, connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return pool, nil
}
