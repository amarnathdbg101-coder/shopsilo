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

type GoRuntimeMetrics struct {
	AllocatedMemoryMB float64 `json:"allocated_memory_mb"`
	SystemMemoryMB    float64 `json:"system_memory_mb"`
	ActiveGoroutines  int     `json:"active_goroutines"`
	GCCycles          uint32  `json:"gc_cycles"`
	GCPauseMs         float64 `json:"gc_pause_ms"`
}

type DBPoolMetrics struct {
	TotalConnections int   `json:"total_connections"`
	IdleConnections  int   `json:"idle_connections"`
	AcquireCount     int64 `json:"acquire_count"`
	MaxConnections   int   `json:"max_connections"`
}

type AIPerformanceMetrics struct {
	AvgResponseTimeMs float64 `json:"avg_response_time_ms"`
	TokenSavingsPct   float64 `json:"token_savings_pct"`
	CacheHitRatePct   float64 `json:"cache_hit_rate_pct"`
	ActiveModelsCount int     `json:"active_models_count"`
}

type PDFEngineMetrics struct {
	MaxWorkers    int `json:"max_workers"`
	ActiveWorkers int `json:"active_workers"`
}

type SystemPerformanceMetrics struct {
	ServerStatus    string               `json:"server_status"`
	UptimeSeconds   int64                `json:"uptime_seconds"`
	GoRuntime       GoRuntimeMetrics     `json:"go_runtime"`
	Database        DBPoolMetrics        `json:"database"`
	AIEngine        AIPerformanceMetrics `json:"ai_engine"`
	PDFEngine       PDFEngineMetrics     `json:"pdf_engine"`
	ActiveErrors24h int                  `json:"active_errors_24h"`
	GeneratedAt     time.Time            `json:"generated_at"`
}
