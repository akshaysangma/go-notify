package api

import (
	"net/http"

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
	r.mux.HandleFunc("POST /api/v1/scheduler", r.schedulerHandler.schedulerControl)
	r.mux.HandleFunc("GET /api/v1/scheduler", r.schedulerHandler.getSchedulerStatus)

	r.mux.HandleFunc("GET /api/v1/messages/sent", r.messageHandler.getSentMessages)

	r.logger.Info("API routes registered.")
}
