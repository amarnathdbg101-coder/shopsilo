package gocb

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// State represents the lifecycle status of the CircuitBreaker.
type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateHalfOpen:
		return "HALF_OPEN"
	case StateOpen:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config configures circuit breaker thresholds.
type Config struct {
	Name                   string
	MaxConsecutiveFailures int
	CooldownDuration       time.Duration
	HalfOpenMaxProbes      int
	OnStateChange          func(name string, from, to State)
}

// DefaultConfig provides sensible defaults for general microservice operations.
var DefaultConfig = Config{
	Name:                   "default",
	MaxConsecutiveFailures: 5,
	CooldownDuration:       10 * time.Second,
	HalfOpenMaxProbes:      1,
	OnStateChange:          nil,
}

// CircuitBreaker guards arbitrary operations against cascading failure cascades.
type CircuitBreaker struct {
	mu               sync.Mutex
	cfg              Config
	state            State
	consecutiveFails int
	halfOpenProbes   int
	openedAt         time.Time
}

// New constructs a new CircuitBreaker.
func New(cfg Config) *CircuitBreaker {
	if cfg.Name == "" {
		cfg.Name = "default"
	}
	if cfg.MaxConsecutiveFailures <= 0 {
		cfg.MaxConsecutiveFailures = 5
	}
	if cfg.CooldownDuration <= 0 {
		cfg.CooldownDuration = 10 * time.Second
	}
	if cfg.HalfOpenMaxProbes <= 0 {
		cfg.HalfOpenMaxProbes = 1
	}

	return &CircuitBreaker{
		cfg:   cfg,
		state: StateClosed,
	}
}

// Execute wraps an arbitrary action with circuit breaker protection.
func (cb *CircuitBreaker) Execute(ctx context.Context, action func(ctx context.Context) error) error {
	return cb.ExecuteWithFallback(ctx, action, nil)
}

// ExecuteWithFallback executes the action, or invokes the fallback if the circuit is open or action fails.
func (cb *CircuitBreaker) ExecuteWithFallback(
	ctx context.Context,
	action func(ctx context.Context) error,
	fallback func(ctx context.Context, err error) error,
) error {
	if err := cb.beforeExecution(); err != nil {
		if fallback != nil {
			return fallback(ctx, err)
		}
		return err
	}

	// Execute action safely with panic recovery
	var (
		actionErr error
		panicked  bool
	)

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				actionErr = fmt.Errorf("circuitbreaker: action panic: %v", r)
			}
		}()
		actionErr = action(ctx)
	}()

	if panicked || actionErr != nil {
		cb.afterExecution(false)
		if fallback != nil {
			return fallback(ctx, actionErr)
		}
		return actionErr
	}

	cb.afterExecution(true)
	return nil
}

func (cb *CircuitBreaker) beforeExecution() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		if now.Sub(cb.openedAt) >= cb.cfg.CooldownDuration {
			// Cooldown passed; transition to HalfOpen
			cb.transitionTo(StateHalfOpen)
			cb.halfOpenProbes = 1
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		if cb.halfOpenProbes >= cb.cfg.HalfOpenMaxProbes {
			return ErrTooManyRequests
		}
		cb.halfOpenProbes++
		return nil
	}

	return nil
}

func (cb *CircuitBreaker) afterExecution(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		if cb.state == StateHalfOpen || cb.consecutiveFails > 0 {
			cb.consecutiveFails = 0
			cb.halfOpenProbes = 0
			cb.transitionTo(StateClosed)
		}
		return
	}

	// Execution failed
	cb.consecutiveFails++
	if cb.state == StateHalfOpen || cb.consecutiveFails >= cb.cfg.MaxConsecutiveFailures {
		cb.openedAt = time.Now()
		cb.transitionTo(StateOpen)
	}
}

func (cb *CircuitBreaker) transitionTo(newState State) {
	if cb.state == newState {
		return
	}
	oldState := cb.state
	cb.state = newState
	if cb.cfg.OnStateChange != nil {
		cb.cfg.OnStateChange(cb.cfg.Name, oldState, newState)
	}
}

// State returns the current State of the CircuitBreaker.
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset forcibly resets the breaker to Closed state with cleared metrics.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveFails = 0
	cb.halfOpenProbes = 0
	cb.transitionTo(StateClosed)
}
