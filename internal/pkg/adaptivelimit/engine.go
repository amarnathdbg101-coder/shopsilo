package adaptivelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ReleaseFunc is returned by Acquire and must be invoked when the bounded operation completes.
type ReleaseFunc func()

// Limiter implements a latency-driven adaptive concurrency limiter using TCP Vegas congestion avoidance.
type Limiter struct {
	cfg Config

	mu         sync.Mutex
	limit      float64
	inFlight   int
	minRTT     time.Duration
	currentRTT time.Duration
	closed     bool

	totalAcquired uint64
	totalRejected uint64
}

// New creates a new adaptive concurrency limiter instance with the specified options.
func New(opts ...Option) *Limiter {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Limiter{
		cfg:   cfg,
		limit: cfg.InitialLimit,
	}
}

// Acquire requests permission to execute. If system concurrency exceeds dynamic limit,
// it immediately returns ErrLimitExceeded.
func (l *Limiter) Acquire(ctx context.Context) (ReleaseFunc, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, ErrLimiterClosed
	}

	// Dynamic capacity check
	if l.inFlight >= int(l.limit) {
		l.mu.Unlock()
		atomic.AddUint64(&l.totalRejected, 1)
		return nil, ErrLimitExceeded
	}

	l.inFlight++
	atomic.AddUint64(&l.totalAcquired, 1)
	l.mu.Unlock()

	start := time.Now()
	var once sync.Once

	release := func() {
		once.Do(func() {
			rtt := time.Since(start)
			l.onSample(rtt)
		})
	}

	return release, nil
}

// onSample adjusts the concurrency limit based on measured execution latency (TCP Vegas).
func (l *Limiter) onSample(rtt time.Duration) {
	if rtt <= 0 {
		rtt = time.Microsecond
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.inFlight--
	if l.inFlight < 0 {
		l.inFlight = 0
	}

	// Update baseline minRTT (no-load baseline latency)
	if l.minRTT == 0 || rtt < l.minRTT {
		l.minRTT = rtt
	}

	// Update smoothed moving average of RTT
	if l.currentRTT == 0 {
		l.currentRTT = rtt
	} else {
		s := l.cfg.Smoothing
		smoothed := float64(s)*float64(rtt) + float64(1.0-s)*float64(l.currentRTT)
		l.currentRTT = time.Duration(smoothed)
	}

	// Calculate estimated queue size: limit * (1 - minRTT / currentRTT)
	gradient := float64(l.minRTT) / float64(l.currentRTT)
	queueSize := l.limit * (1.0 - gradient)

	// AIMD (Additive Increase, Multiplicative Decrease)
	if queueSize < l.cfg.Alpha {
		// System healthy and queue is low -> gently expand limit
		l.limit += 1.0
	} else if queueSize > l.cfg.Beta {
		// Latency is inflating -> congestion detected! Cut limit back
		l.limit *= l.cfg.BackoffRatio
	}

	// Enforce floor and ceiling bounds
	if l.limit < l.cfg.MinLimit {
		l.limit = l.cfg.MinLimit
	}
	if l.limit > l.cfg.MaxLimit {
		l.limit = l.cfg.MaxLimit
	}
}

// Execute runs fn guarded by the adaptive limiter.
func (l *Limiter) Execute(ctx context.Context, fn func() error) error {
	release, err := l.Acquire(ctx)
	if err != nil {
		return err
	}
	defer release()
	return fn()
}

// Limit returns the current dynamic concurrency limit.
func (l *Limiter) Limit() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.limit
}

// Metrics returns a snapshot of current telemetry.
func (l *Limiter) Metrics() Metrics {
	l.mu.Lock()
	defer l.mu.Unlock()

	return Metrics{
		CurrentLimit:  l.limit,
		InFlight:      l.inFlight,
		MinRTT:        l.minRTT,
		CurrentRTT:    l.currentRTT,
		TotalAcquired: atomic.LoadUint64(&l.totalAcquired),
		TotalRejected: atomic.LoadUint64(&l.totalRejected),
	}
}

// Close closes the limiter.
func (l *Limiter) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	return nil
}
