// Package middleware provides HTTP interceptors, rate limiters, and resilience controls.
package middleware

import (
	"github.com/amarnathdbg101-coder/go/go-adaptivelimit"
)

var (
	// GlobalAdaptiveLimiter dynamically tunes HTTP concurrency based on measured latency gradient (TCP Vegas).
	// During low latency, it scales up to 100 concurrent requests; during DB latency spikes, it drops to 5.
	GlobalAdaptiveLimiter = adaptivelimit.New(
		adaptivelimit.WithInitialLimit(20),
		adaptivelimit.WithLimits(5, 100),
	)

	// AdaptiveLimiterMiddleware wraps HTTP router with TCP Vegas load shedding.
	AdaptiveLimiterMiddleware = adaptivelimit.Middleware(GlobalAdaptiveLimiter)
)
