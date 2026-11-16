package supervisor

import (
	"context"
	"time"
)

// Strategy defines how child worker crashes affect sibling workers.
type Strategy string

const (
	// StrategyOneForOne restarts only the failed worker.
	StrategyOneForOne Strategy = "ONE_FOR_ONE"

	// StrategyOneForAll terminates and restarts all supervised workers if one fails.
	StrategyOneForAll Strategy = "ONE_FOR_ALL"
)

// RestartPolicy determines when a worker should be restarted after termination.
type RestartPolicy string

const (
	// PolicyPermanent always restarts the worker regardless of exit reason.
	PolicyPermanent RestartPolicy = "PERMANENT"

	// PolicyTransient restarts the worker only if it terminates with an error or panic.
	PolicyTransient RestartPolicy = "TRANSIENT"

	// PolicyTemporary never restarts the worker upon termination.
	PolicyTemporary RestartPolicy = "TEMPORARY"
)

// WorkerFunc represents the supervised long-running background task.
type WorkerFunc func(ctx context.Context) error

// WorkerSpec specifies the operational configuration of an individual worker.
type WorkerSpec struct {
	Name    string
	Func    WorkerFunc
	Policy  RestartPolicy
	Backoff time.Duration
}

// Config specifies the runtime settings for the supervisor tree.
type Config struct {
	Strategy      Strategy
	MaxRestarts   int
	RestartWindow time.Duration

	// Hooks
	OnWorkerPanic      func(name string, panicVal any, stack []byte)
	OnWorkerTerminated func(name string, err error)
}

// Option is a functional option for configuring the supervisor.
type Option func(*Config)

// DefaultConfig returns resilient production defaults for the supervisor.
func DefaultConfig() Config {
	return Config{
		Strategy:      StrategyOneForOne,
		MaxRestarts:   3,
		RestartWindow: 10 * time.Second,
	}
}

// WithStrategy sets the supervisor restart strategy (OneForOne or OneForAll).
func WithStrategy(s Strategy) Option {
	return func(c *Config) {
		c.Strategy = s
	}
}

// WithCrashLoopProtection sets the maximum allowed restarts within the specified time window.
func WithCrashLoopProtection(maxRestarts int, window time.Duration) Option {
	return func(c *Config) {
		if maxRestarts > 0 {
			c.MaxRestarts = maxRestarts
		}
		if window > 0 {
			c.RestartWindow = window
		}
	}
}

// WithOnPanic registers a panic diagnostic callback.
func WithOnPanic(fn func(name string, panicVal any, stack []byte)) Option {
	return func(c *Config) {
		c.OnWorkerPanic = fn
	}
}
