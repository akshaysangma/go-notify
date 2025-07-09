package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/akshaysangma/go-notify/external/redis"
	"github.com/akshaysangma/go-notify/external/webhook"
	"github.com/akshaysangma/go-notify/internal/api"
	"github.com/akshaysangma/go-notify/internal/config"
	"github.com/akshaysangma/go-notify/internal/database"
	"github.com/akshaysangma/go-notify/internal/database/postgres"
	"github.com/akshaysangma/go-notify/internal/messages"
	"github.com/akshaysangma/go-notify/internal/scheduler"

	"go.uber.org/zap"
)

// @title Go Notify API
// @version 1.0
// @description This is a service for automatically sending scheduled messages.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
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

	// Upper limit on workers
	workerPoolSize := 2 * runtime.NumCPU()
	workerPoolSize = min(cfg.Scheduler.MessageRate, workerPoolSize)

	msgRepo := database.NewPostgresMessageRepository(pgPool)
	webhookSiteSenderClient := webhook.NewWebhookSiteSender(cfg.Webhook.URL, cfg.Webhook.CharacterLimit, cfg.Server.WriteTimeout)
	redisClient := redis.NewRedisService(cfg.Redis.Address, logger)
	msgService := messages.NewMessageService(msgRepo, webhookSiteSenderClient, logger, redisClient, workerPoolSize)
	msgdispatchScheduler := scheduler.NewMessageDispatchSchedulerImpl(msgService, logger, cfg.Scheduler)
	// Initial Start

	logger.Info("Starting message dispatching scheduler...")
	msgdispatchScheduler.Start()

	mux := http.NewServeMux()
	messageH := api.NewMessageHandler(msgService, cfg.Webhook.CharacterLimit, logger)
	schedulerH := api.NewSchedulerHandler(msgdispatchScheduler, logger)

	routes := api.NewRouterDependecies(mux, messageH, schedulerH, logger)
	routes.RegisterRoutes()

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start HTTP server
	go func() {
		logger.Info("HTTP server starting...", zap.Int("port", cfg.Server.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed to start", zap.Error(err))
		}
	}()

	// Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop // Block until a signal is received

	logger.Info("Shutdown signal received. Starting graceful shutdown...")

	// 1. Shut down the scheduler first
	if msgdispatchScheduler.IsRunning() {
		logger.Info("Stopping message scheduler gracefully...")
		if err := msgdispatchScheduler.Stop(); err != nil {
			logger.Error("Error stopping scheduler", zap.Error(err))
		} else {
			logger.Info("Message scheduler stopped.")
		}
	} else {
		logger.Info("Message scheduler was not running.")
	}

	// 2. Shut down the HTTP server
	// Create a context with a timeout for the server shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.WriteTimeout+cfg.Server.IdleTimeout)
	defer shutdownCancel()

	logger.Info("Stopping HTTP server gracefully...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server graceful shutdown failed", zap.Error(err))
	} else {
		logger.Info("HTTP server stopped.")
	}

	logger.Info("Application shutdown complete.")
}
