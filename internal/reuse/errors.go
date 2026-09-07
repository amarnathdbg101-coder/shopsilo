package reuse

import (
	"encoding/json"
	"net/http"
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
	JSON(w, status, Response{Success: false, Error: err})
}

func Message(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, Response{Success: true, Message: msg})
}

