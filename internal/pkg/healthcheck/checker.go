package gohealthcheck

import (
	"context"
	"fmt"
	"runtime"
)

// Status represents the health condition of a component.
type Status string

const (
	StatusUp   Status = "UP"
	StatusDown Status = "DOWN"
)

// CheckResult contains health telemetry for an individual component.
type CheckResult struct {
	Status     Status  `json:"status"`
	DurationMs float64 `json:"duration_ms"`
	Error      string  `json:"error,omitempty"`
}

// Checker defines the contract for any health check component.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// CheckFunc is an adapter allowing a plain function to act as a Checker.
type CheckFunc struct {
	name string
	fn   func(ctx context.Context) error
}

func (c *CheckFunc) Name() string {
	return c.name
}

func (c *CheckFunc) Check(ctx context.Context) error {
	return c.fn(ctx)
}

// NewCheckFunc wraps a function into a Checker.
func NewCheckFunc(name string, fn func(ctx context.Context) error) Checker {
	return &CheckFunc{name: name, fn: fn}
}

// Pinger defines an interface satisfied by database/sql (*sql.DB), Redis, etc.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// NewPingChecker creates a checker for any database or connection satisfying Pinger.
func NewPingChecker(name string, pinger Pinger) Checker {
	return NewCheckFunc(name, func(ctx context.Context) error {
		if pinger == nil {
			return fmt.Errorf("pinger is nil")
		}
		return pinger.PingContext(ctx)
	})
}

// NewMemoryChecker alerts if allocated heap memory exceeds maxAllocBytes.
func NewMemoryChecker(name string, maxAllocBytes uint64) Checker {
	return NewCheckFunc(name, func(ctx context.Context) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		if m.Alloc > maxAllocBytes {
			return fmt.Errorf("memory alloc %d bytes exceeds threshold %d bytes", m.Alloc, maxAllocBytes)
		}
		return nil
	})
}
