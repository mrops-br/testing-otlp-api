package response

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// JSON sends a JSON response
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Error sends an error response
func Error(w http.ResponseWriter, status int, err error) {
	errorType := "error"
	switch status {
	case http.StatusNotFound:
		errorType = "not_found"
	case http.StatusBadRequest:
		errorType = "bad_request"
	case http.StatusInternalServerError:
		errorType = "internal_server_error"
	}

	JSON(w, status, ErrorResponse{
		Error:   errorType,
		Message: err.Error(),
	})
}
