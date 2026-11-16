package gohealthcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Report aggregates overall health status and component checks.
type Report struct {
	Status    Status                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Duration  float64                `json:"total_duration_ms"`
	Details   map[string]CheckResult `json:"details,omitempty"`
}

// Config tunes the health coordinator.
type Config struct {
	Timeout  time.Duration
	CacheTTL time.Duration
}

// Health coordinates concurrent health checks and delivers HTTP handlers.
type Health struct {
	mu           sync.RWMutex
	readiness    []Checker
	liveness     []Checker
	timeout      time.Duration
	cacheTTL     time.Duration
	cachedReport *Report
	cachedUntil  time.Time
}

// New creates a new Health manager with optional configuration.
func New(cfg ...Config) *Health {
	c := Config{
		Timeout:  2 * time.Second,
		CacheTTL: 1 * time.Second,
	}
	if len(cfg) > 0 {
		if cfg[0].Timeout > 0 {
			c.Timeout = cfg[0].Timeout
		}
		if cfg[0].CacheTTL >= 0 {
			c.CacheTTL = cfg[0].CacheTTL
		}
	}

	return &Health{
		readiness: make([]Checker, 0),
		liveness:  make([]Checker, 0),
		timeout:   c.Timeout,
		cacheTTL:  c.CacheTTL,
	}
}

// RegisterReadiness adds a dependency checker evaluated for /readyz.
func (h *Health) RegisterReadiness(checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.readiness = append(h.readiness, checker)
}

// RegisterLiveness adds an internal checker evaluated for /livez.
func (h *Health) RegisterLiveness(checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.liveness = append(h.liveness, checker)
}

// CheckReadiness executes all registered readiness checks concurrently.
func (h *Health) CheckReadiness(ctx context.Context) Report {
	h.mu.RLock()
	if h.cachedReport != nil && time.Now().Before(h.cachedUntil) {
		report := *h.cachedReport
		h.mu.RUnlock()
		return report
	}
	h.mu.RUnlock()

	report := h.runChecks(ctx, h.readiness)

	if h.cacheTTL > 0 {
		h.mu.Lock()
		h.cachedReport = &report
		h.cachedUntil = time.Now().Add(h.cacheTTL)
		h.mu.Unlock()
	}

	return report
}

// CheckLiveness executes all registered liveness checks concurrently.
func (h *Health) CheckLiveness(ctx context.Context) Report {
	return h.runChecks(ctx, h.liveness)
}

func (h *Health) runChecks(ctx context.Context, checkers []Checker) Report {
	start := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	details := make(map[string]CheckResult, len(checkers))
	var detailsMu sync.Mutex
	var wg sync.WaitGroup

	overallStatus := StatusUp

	for _, checker := range checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()

			cStart := time.Now()
			checkErr := c.Check(timeoutCtx)
			duration := float64(time.Since(cStart).Microseconds()) / 1000.0

			res := CheckResult{
				Status:     StatusUp,
				DurationMs: duration,
			}

			if checkErr != nil {
				res.Status = StatusDown
				res.Error = checkErr.Error()
			}

			detailsMu.Lock()
			details[c.Name()] = res
			if res.Status == StatusDown {
				overallStatus = StatusDown
			}
			detailsMu.Unlock()
		}(checker)
	}

	wg.Wait()

	return Report{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Duration:  float64(time.Since(start).Microseconds()) / 1000.0,
		Details:   details,
	}
}

// LiveHandler returns a standard HTTP handler for Kubernetes liveness probes (/livez).
func (h *Health) LiveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report := h.CheckLiveness(r.Context())
		writeJSON(w, report)
	}
}

// ReadyHandler returns a standard HTTP handler for Kubernetes readiness probes (/readyz).
// Returns HTTP 200 if UP, HTTP 503 if DOWN.
func (h *Health) ReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report := h.CheckReadiness(r.Context())
		writeJSON(w, report)
	}
}

func writeJSON(w http.ResponseWriter, report Report) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if report.Status == StatusUp {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(report)
}
