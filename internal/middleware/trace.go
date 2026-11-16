// Package middleware provides HTTP interceptors, rate limiters, and resilience controls.
package middleware

import (
	"shopMe/internal/pkg/w3ctrace"
)

var (
	// GlobalTracer is the centralized W3C distributed tracer for ShopSilo.
	GlobalTracer = w3ctrace.NewTracer("shopsilo-backend", w3ctrace.NewMemoryExporter())

	// TraceMiddleware automatically extracts upstream W3C traceparent headers or generates
	// fresh compliant trace IDs, attaches them to request context, and writes traceparent to response.
	TraceMiddleware = w3ctrace.Middleware(GlobalTracer)
)
