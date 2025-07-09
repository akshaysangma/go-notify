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

// SchedulerStatusResponse represents the response for the scheduler status endpoint.
type SchedulerStatusResponse struct {
	Status string `json:"status" example:"running"`
}

// getSchedulerStatus godoc
// @Summary      Get the current status of the scheduler
// @Description  Returns whether the scheduler is currently running or stopped.
// @Tags         scheduler
// @Produce      json
// @Success      200 {object} SchedulerStatusResponse "Current status of the scheduler"
// @Router /api/v1/scheduler [get]
func (h *SchedulerHandler) getSchedulerStatus(w http.ResponseWriter, r *http.Request) {
	resp := SchedulerStatusResponse{
		Status: "stopped",
	}
	if h.scheduler.IsRunning() {
		resp.Status = "running"
	}
	WriteJSONResponse(w, http.StatusOK, resp)
}

// schedulerControl godoc
// @Summary      Control the message sending scheduler (start/stop)
// @Description  Activates or deactivates the scheduler based on the 'action' query parameter.
// @Tags         scheduler
// @Produce      json
// @Param        action query      string  true  "The action to perform: 'start' or 'stop'" Enums(start, stop)
// @Success      202  {object}  SuccessResponse "Action signal sent successfully"
// @Failure      400  {object}  HTTPError "Invalid or missing 'action' parameter"
// @Failure      409  {object}  HTTPError "Scheduler is already in the desired state"
// @Failure      500  {object}  HTTPError "Internal server error while performing the action"
// @Router /api/v1/scheduler [post]
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
		WriteJSONResponse(w, http.StatusAccepted, SuccessResponse{Message: "Scheduler start signal sent."})
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
		WriteJSONResponse(w, http.StatusAccepted, SuccessResponse{Message: "Scheduler stop signal sent."})
	default:
		WriteJSONErrorResponse(w, http.StatusBadRequest, "Invalid or missing 'action' query parameter. Must be 'start' or 'stop'.", fmt.Errorf("action query param missing"))
	}
}
