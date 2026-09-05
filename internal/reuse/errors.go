package reuse

import (
    "encoding/json"
    "net/http"
)

// Code type for error codes
type Code string

// Common error codes (constants)
const (
    ErrInvalidInput Code = "INVALID_INPUT"
    ErrUnauthorized Code = "UNAUTHORIZED"
    ErrForbidden    Code = "FORBIDDEN"
    ErrNotFound     Code = "NOT_FOUND"
    ErrConflict     Code = "CONFLICT"
    ErrDBFailure    Code = "DB_FAILURE"
    ErrInternal     Code = "INTERNAL_ERROR"
)

// Standard response structure
type Response struct {
    Status  string        `json:"status"`  // "success" or "error"
    Message string        `json:"message"` // human-readable message
    Data    interface{}   `json:"data,omitempty"`
    Error   *ErrorPayload `json:"error,omitempty"`
}

// Error payload structure
type ErrorPayload struct {
    Code    Code   `json:"code"`
    Message string `json:"message"`
}

// Success response
func Success(w http.ResponseWriter, message string, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)

    _ = json.NewEncoder(w).Encode(Response{
        Status:  "success",
        Message: message,
        Data:    data,
    })
}

// Error response
func Error(w http.ResponseWriter, status int, code Code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    _ = json.NewEncoder(w).Encode(Response{
        Status:  "error",
        Message: message,
        Error: &ErrorPayload{
            Code:    code,
            Message: message,
        },
    })
}
