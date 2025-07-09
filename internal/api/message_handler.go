package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/akshaysangma/go-notify/internal/messages"
	"go.uber.org/zap"
)

// MessageServicer defines the interface for the message service accepted by message handler.
type MessageServicer interface {
	GetAllSentMessages(ctx context.Context, limit, offset int32) ([]messages.Message, error)
	CreateMessages(ctx context.Context, content string, recipients []string, charLimit int) error
}

// CreateMessagesRequest defines the request body for creating a message for multiple recipients.
type CreateMessagesRequest struct {
	Content    string   `json:"content" example:"This is a message for multiple users."`
	Recipients []string `json:"recipients" example:"['+15551112222', '+15553334444']"`
}

// MessageHandler holds the dependencies for the message-related API handlers.
type MessageHandler struct {
	service              MessageServicer
	allowedContentLength int
	logger               *zap.Logger
}

// NewMessageHandler creates and configures a new MessageHandler using the standard library's ServeMux.
func NewMessageHandler(service MessageServicer, contentLength int, logger *zap.Logger) *MessageHandler {
	h := &MessageHandler{
		service:              service,
		logger:               logger,
		allowedContentLength: contentLength,
	}
	return h
}

// getSentMessages godoc
// @Summary      Retrieve a list of sent messages
// @Description  Gets a paginated list of all messages that have been successfully sent.
// @Tags         messages
// @Produce      json
// @Param        limit   query      int    false  "Number of messages to return" default(20)
// @Param        offset  query      int    false  "Offset for pagination" default(0)
// @Success      200     {array}    messages.Message "A list of sent messages"
// @Failure      500     {object}   HTTPError "Failed to retrieve sent messages"
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

// createMessages godoc
// @Summary      Create a message for multiple recipients
// @Description  Creates a new message with the same content for a list of recipient phone numbers.
// @Tags         messages
// @Accept       json
// @Produce      json
// @Param        message body       CreateMessagesRequest true "Message Content and Recipients"
// @Success      202     {object}   SuccessResponse "Messages have been accepted for processing"
// @Failure      400     {object}   HTTPError "Invalid request body or message content"
// @Failure      500     {object}   HTTPError "Failed to save messages to the database"
// @Router       /api/v1/messages [post]
func (h *MessageHandler) createMessages(w http.ResponseWriter, r *http.Request) {
	var req CreateMessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSONErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	err := h.service.CreateMessages(r.Context(), req.Content, req.Recipients, h.allowedContentLength)
	if err != nil {
		if errors.Is(err, messages.ErrContentTooLong) || errors.Is(err, messages.ErrRecipientEmpty) {
			WriteJSONErrorResponse(w, http.StatusBadRequest, "Invalid message data", err)
			return
		}
		WriteJSONErrorResponse(w, http.StatusInternalServerError, "Could not create messages", err)
		return
	}

	WriteJSONResponse(w, http.StatusAccepted, SuccessResponse{Message: "Messages accepted for creation."})
}
