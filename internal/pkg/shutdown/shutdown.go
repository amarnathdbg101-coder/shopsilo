package goshutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"
)

const (
	// PhaseHTTP stops HTTP and ingress traffic (in-flight requests allowed to finish).
	PhaseHTTP = 1

	// PhaseWorkers stops background queues, event buses, and worker pools.
	PhaseWorkers = 2

	// PhaseStorage closes databases, caches, and storage connections.
	PhaseStorage = 3
)

// Hook defines a clean termination callback with context timeout awareness.
type Hook func(ctx context.Context) error

type hookEntry struct {
	phase int
	name  string
	fn    Hook
}

// Coordinator orchestrates phased, graceful application shutdowns upon receiving OS signals.
type Coordinator struct {
	mu      sync.Mutex
	timeout time.Duration
	signals []os.Signal
	hooks   []hookEntry
	sigChan chan os.Signal
}

// New constructs a shutdown Coordinator.
func New(timeout time.Duration, signals ...os.Signal) *Coordinator {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if len(signals) == 0 {
		signals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT}
	}

	return &Coordinator{
		timeout: timeout,
		signals: signals,
		hooks:   make([]hookEntry, 0),
		sigChan: make(chan os.Signal, 1),
	}
}

// Register adds a shutdown hook to a designated execution phase (ascending numerical order).
func (c *Coordinator) Register(phase int, name string, fn Hook) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hooks = append(c.hooks, hookEntry{
		phase: phase,
		name:  name,
		fn:    fn,
	})
}

// RegisterHTTP adds a hook to PhaseHTTP (Phase 1).
func (c *Coordinator) RegisterHTTP(name string, fn Hook) {
	c.Register(PhaseHTTP, name, fn)
}

// RegisterWorker adds a hook to PhaseWorkers (Phase 2).
func (c *Coordinator) RegisterWorker(name string, fn Hook) {
	c.Register(PhaseWorkers, name, fn)
}

// RegisterStorage adds a hook to PhaseStorage (Phase 3).
func (c *Coordinator) RegisterStorage(name string, fn Hook) {
	c.Register(PhaseStorage, name, fn)
}

// Trigger executes all registered phases sequentially without waiting for an OS signal.
func (c *Coordinator) Trigger(ctx context.Context) error {
	c.mu.Lock()
	// Sort hooks by phase ascending
	sortedHooks := make([]hookEntry, len(c.hooks))
	copy(sortedHooks, c.hooks)
	sort.SliceStable(sortedHooks, func(i, j int) bool {
		return sortedHooks[i].phase < sortedHooks[j].phase
	})
	c.mu.Unlock()

	var errs []string

	for _, h := range sortedHooks {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("shutdown: timeout before phase %d (%s): %w", h.phase, h.name, err)
		}

		if err := h.fn(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("[%s]: %v", h.name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown completed with errors: %s", fmt.Sprintf("%v", errs))
	}

	return nil
}

// Wait blocks until an OS termination signal is received, then executes all phased hooks within the configured timeout.
func (c *Coordinator) Wait() error {
	signal.Notify(c.sigChan, c.signals...)
	sig := <-c.sigChan

	_ = fmt.Sprintf("Received shutdown signal: %v", sig)

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	return c.Trigger(ctx)
}
