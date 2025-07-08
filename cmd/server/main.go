package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/akshaysangma/go-notify/internal/config"
	"github.com/akshaysangma/go-notify/internal/database/postgres"

	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Fatal: failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := config.InitLogger(cfg.App.Environment)
	if err != nil {
		fmt.Printf("Fatal: failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Application starting up...", zap.String("environment", cfg.App.Environment))

	// Initialize PostgreSQL connection pool
	dbPoolCtx, dbPoolCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbPoolCancel()
	pgPool, err := postgres.NewPostgresDB(dbPoolCtx, cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer pgPool.Close()

}
