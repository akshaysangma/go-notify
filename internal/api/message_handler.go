package api

import (
	"net/http"
	"strconv"

	"github.com/akshaysangma/go-notify/internal/messages"
	"go.uber.org/zap"
)

// MessageHandler holds the dependencies for the message-related API handlers.
type MessageHandler struct {
	service *messages.MessageService
	logger  *zap.Logger
}

// NewMessageHandler creates and configures a new MessageHandler using the standard library's ServeMux.
func NewMessageHandler(service *messages.MessageService, logger *zap.Logger) *MessageHandler {
	h := &MessageHandler{
		service: service,
		logger:  logger,
	}
	return h
}

// getSentMessages godoc
// @Summary Retrieve a list of sent messages
// @Description Gets a paginated list of all messages that have been successfully sent.
// @Tags messages
// @Produce json
// @Param limit query int false "Number of messages to return" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} messages.Message
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/messages/sent [get]
func (h *MessageHandler) getSentMessages(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = defaultOffset
	}

	sentMessages, err := h.service.GetAllSentMessages(r.Context(), int32(limit), int32(offset))
	if err != nil {
		h.logger.Error("Failed to get sent messages", zap.Error(err))
		WriteJSONErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve sent messages", err)
		return
	}

	WriteJSONResponse(w, http.StatusOK, sentMessages)
}
