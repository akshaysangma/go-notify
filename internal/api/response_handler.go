package api

import (
	"encoding/json"
	"net/http"
)

// HTTPError represents a standard error response for all API failures.
type HTTPError struct {
	Error   string `json:"error" example:"Descriptive error message"`
	Details string `json:"details,omitempty" example:"Optional: specific error details"`
}

// SuccessResponse represents a generic success message.
type SuccessResponse struct {
	Message string `json:"message" example:"Action was successful"`
}

// WriteJSONResponse is a helper to write consistent JSON responses.
func WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// WriteJSONErrorResponse is a helper to write consistent JSON error responses.
func WriteJSONErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	resp := HTTPError{
		Error: message,
	}
	if err != nil {
		resp.Details = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
