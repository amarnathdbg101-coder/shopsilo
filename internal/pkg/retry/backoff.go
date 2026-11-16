package goretry

import (
	"math/rand"
	"time"
)

// JitterMode specifies how randomness is added to avoid thundering herds.
type JitterMode int

const (
	FullJitter JitterMode = iota
	EqualJitter
	NoJitter
)

// Backoff computes delay between retry attempts.
type Backoff interface {
	NextDelay(attempt int) time.Duration
}

type exponentialBackoff struct {
	initial    time.Duration
	max        time.Duration
	multiplier float64
}

// Exponential creates an exponential backoff strategy (initial * multiplier^attempt).
func Exponential(initial, max time.Duration, multiplier float64) Backoff {
	if initial <= 0 {
		initial = 100 * time.Millisecond
	}
	if max <= 0 {
		max = 5 * time.Second
	}
	if multiplier <= 1.0 {
		multiplier = 2.0
	}
	return &exponentialBackoff{
		initial:    initial,
		max:        max,
		multiplier: multiplier,
	}
}

func (e *exponentialBackoff) NextDelay(attempt int) time.Duration {
	mult := 1.0
	for i := 0; i < attempt; i++ {
		mult *= e.multiplier
	}
	calculated := float64(e.initial) * mult
	if calculated > float64(e.max) {
		calculated = float64(e.max)
	}
	return time.Duration(calculated)
}

type constantBackoff struct {
	interval time.Duration
}

// Constant creates a fixed interval backoff strategy.
func Constant(interval time.Duration) Backoff {
	return &constantBackoff{interval: interval}
}

func (c *constantBackoff) NextDelay(attempt int) time.Duration {
	return c.interval
}

type linearBackoff struct {
	initial time.Duration
	step    time.Duration
	max     time.Duration
}

// Linear creates a linear increasing backoff strategy (initial + attempt*step).
func Linear(initial, step, max time.Duration) Backoff {
	return &linearBackoff{
		initial: initial,
		step:    step,
		max:     max,
	}
}

func (l *linearBackoff) NextDelay(attempt int) time.Duration {
	d := l.initial + time.Duration(attempt)*l.step
	if d > l.max {
		d = l.max
	}
	return d
}

// ApplyJitter applies the specified jitter algorithm to a base duration.
func ApplyJitter(d time.Duration, mode JitterMode) time.Duration {
	if d <= 0 {
		return 0
	}

	switch mode {
	case FullJitter:
		// Random between 0 and d
		return time.Duration(rand.Float64() * float64(d))

	case EqualJitter:
		// Half fixed, half random
		half := float64(d) / 2.0
		return time.Duration(half + rand.Float64()*half)

	case NoJitter:
		return d

	default:
		return d
	}
}
