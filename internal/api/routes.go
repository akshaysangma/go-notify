package api

import (
	"net/http"

	_ "github.com/akshaysangma/go-notify/docs"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

type RouterDependecies struct {
	mux              *http.ServeMux
	messageHandler   *MessageHandler
	schedulerHandler *SchedulerHandler
	logger           *zap.Logger
}

func NewRouterDependecies(mux *http.ServeMux,
	msgHandler *MessageHandler,
	schHandler *SchedulerHandler,
	logger *zap.Logger) *RouterDependecies {
	return &RouterDependecies{
		mux:              mux,
		logger:           logger,
		messageHandler:   msgHandler,
		schedulerHandler: schHandler,
	}
}

func (r *RouterDependecies) RegisterRoutes() {
	// Scheduler releated APIs
	r.mux.HandleFunc("POST /api/v1/scheduler", r.schedulerHandler.schedulerControl)
	r.mux.HandleFunc("GET /api/v1/scheduler", r.schedulerHandler.getSchedulerStatus)

	// Messages related APIs
	r.mux.HandleFunc("GET /api/v1/messages/sent", r.messageHandler.getSentMessages)
	r.mux.HandleFunc("POST /api/v1/messages", r.messageHandler.createMessages)

	// Swagger UI
	r.mux.HandleFunc("GET /swagger/", httpSwagger.WrapHandler)
	r.logger.Info("API routes registered.")
}
