package dto

import "time"

type TelemetryErrorRequest struct {
	ErrorCode           string `json:"error_code"`
	UserFriendlyMsg     string `json:"user_friendly_msg"`
	DeveloperStackTrace string `json:"developer_stack_trace" validate:"required"`
	RequestPath         string `json:"request_path,omitempty"`
	UserID              string `json:"user_id,omitempty"`
	UserRole            string `json:"user_role,omitempty"`
}

type SystemErrorRecord struct {
	ID                  string    `json:"id"`
	ErrorCode           string    `json:"error_code"`
	UserFriendlyMsg     string    `json:"user_friendly_msg"`
	DeveloperStackTrace string    `json:"developer_stack_trace"`
	RequestPath         string    `json:"request_path"`
	UserID              string    `json:"user_id"`
	UserRole            string    `json:"user_role"`
	CreatedAt           time.Time `json:"created_at"`
	ExpiresAt           time.Time `json:"expires_at"`
}
