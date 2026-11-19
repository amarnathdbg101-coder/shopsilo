package reuse

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func Success(w http.ResponseWriter, msg string, data interface{}) {
	JSON(w, http.StatusOK, Response{Success: true, Message: msg, Data: data})
}

func Created(w http.ResponseWriter, msg string, data interface{}) {
	JSON(w, http.StatusCreated, Response{Success: true, Message: msg, Data: data})
}

func Error(w http.ResponseWriter, status int, err string) {
	cleanErr := err
	// OWASP Information Leakage Guard:
	// Mask raw database driver/connection errors from being leaked in HTTP 500 responses.
	if status == http.StatusInternalServerError {
		lower := strings.ToLower(err)
		if strings.Contains(lower, "pgx:") || strings.Contains(lower, "sql:") || strings.Contains(lower, "pq:") ||
			strings.Contains(lower, "connection refused") || strings.Contains(lower, "password") ||
			strings.Contains(lower, "syntax error at or near") || strings.Contains(lower, "driver:") {
			cleanErr = "An internal server error occurred. Please try again later."
		}
	}
	JSON(w, status, Response{Success: false, Error: cleanErr})
}

func Message(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, Response{Success: true, Message: msg})
}

