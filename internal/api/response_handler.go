package api

import (
	"encoding/json"
	"net/http"
)

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
	resp := map[string]string{
		"error":   message,
		"details": "",
	}
	if err != nil {
		resp["details"] = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
