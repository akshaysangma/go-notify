package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/akshaysangma/go-notify/internal/scheduler"
	"go.uber.org/zap"
)

const (
	defaultLimit  = 20
	defaultOffset = 0
	maxLimit      = 100
)

// SchedulerHandler holds the dependencies for the message-related API handlers.
type SchedulerHandler struct {
	scheduler *scheduler.MessageDispatchSchedulerImpl
	logger    *zap.Logger
}

// NewSchedulerHandler creates and configures a new SchedulerHandler using the standard library's ServeMux.
func NewSchedulerHandler(scheduler *scheduler.MessageDispatchSchedulerImpl, logger *zap.Logger) *SchedulerHandler {
	h := &SchedulerHandler{
		scheduler: scheduler,
		logger:    logger,
	}
	return h
}

// getSchedulerStatus godoc
// @Summary Get the current status of the scheduler
// @Description Returns whether the scheduler is currently running or stopped.
// @Tags scheduler
// @Produce json
// @Success 200 {object} map[string]string "Current status of the scheduler"
// @Router /api/v1/scheduler/status [get]
func (h *SchedulerHandler) getSchedulerStatus(w http.ResponseWriter, r *http.Request) {
	status := "stopped"
	if h.scheduler.IsRunning() {
		status = "running"
	}
	WriteJSONResponse(w, http.StatusOK, map[string]string{"status": status})
}

// schedulerControl godoc
// @Summary Control the message sending scheduler (start/stop)
// @Description Activates or deactivates the scheduler based on the 'action' query parameter.
// @Tags scheduler
// @Accept json
// @Produce json
// @Param action query string true "The action to perform: 'start' or 'stop'"
// @Success 202 {object} map[string]string "Action signal sent successfully"
// @Failure 400 {object} map[string]string "Invalid or missing 'action' parameter"
// @Router /api/v1/scheduler/control [post]
func (h *SchedulerHandler) schedulerControl(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")

	switch action {
	case "start":
		err := h.scheduler.Start()
		if err != nil {
			if errors.Is(err, scheduler.ErrAlreadyRunning) {
				WriteJSONErrorResponse(w, http.StatusConflict, "Scheduler is already running", err)
				return
			}
			WriteJSONErrorResponse(w, http.StatusInternalServerError, "Failed to start scheduler", err)
			return
		}
		WriteJSONResponse(w, http.StatusAccepted, map[string]string{"message": "Scheduler start signal sent."})
	case "stop":
		err := h.scheduler.Stop()
		if err != nil {
			if errors.Is(err, scheduler.ErrNotRunning) {
				WriteJSONErrorResponse(w, http.StatusConflict, "Scheduler is already stopped", err)
				return
			}
			WriteJSONErrorResponse(w, http.StatusInternalServerError, "Failed to stop scheduler", err)
			return
		}
		WriteJSONResponse(w, http.StatusAccepted, map[string]string{"message": "Scheduler stop signal sent."})
	default:
		WriteJSONErrorResponse(w, http.StatusBadRequest, "Invalid or missing 'action' query parameter. Must be 'start' or 'stop'.", fmt.Errorf("action query param missing"))
	}
}
