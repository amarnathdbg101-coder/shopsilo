package w3ctrace

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// InjectHTTP writes the active span context into HTTP headers per W3C specification.
func InjectHTTP(ctx context.Context, header http.Header) {
	span := SpanFromContext(ctx)
	if span == nil || !span.Context.IsValid() {
		return
	}
	header.Set(HeaderTraceparent, span.Context.String())
	if span.Context.TraceState != "" {
		header.Set(HeaderTracestate, span.Context.TraceState)
	}
}

// ExtractHTTP parses the incoming W3C traceparent header.
func ExtractHTTP(header http.Header) (SpanContext, error) {
	raw := header.Get(HeaderTraceparent)
	if raw == "" {
		// Fallback check for case variations
		raw = header.Get("Traceparent")
	}
	if raw == "" {
		return SpanContext{}, ErrInvalidTraceparent
	}
	sc, err := ParseTraceparent(raw)
	if err != nil {
		return SpanContext{}, err
	}
	sc.TraceState = header.Get(HeaderTracestate)
	return sc, nil
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Middleware creates a standard net/http middleware that automatically propagates W3C trace contexts.
func Middleware(tracer *Tracer) func(http.Handler) http.Handler {
	if tracer == nil {
		tracer = GetGlobalTracer()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract incoming traceparent if provided by upstream client or reverse proxy
			if incomingSC, err := ExtractHTTP(r.Header); err == nil && incomingSC.IsValid() {
				// Create parent context
				parentSpan := &Span{Context: incomingSC}
				ctx = context.WithValue(ctx, traceCtxKey{}, parentSpan)
			}

			// Start span for current HTTP request
			spanName := fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path)
			ctx, span := tracer.StartSpan(ctx, spanName)
			defer span.End()

			span.SetAttribute("http.method", r.Method)
			span.SetAttribute("http.url", r.URL.String())
			span.SetAttribute("http.host", r.Host)

			// Propagate traceparent downstream in the response headers
			w.Header().Set(HeaderTraceparent, span.Context.String())

			sw := &statusCapturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(sw, r.WithContext(ctx))

			span.SetAttribute("http.status_code", strconv.Itoa(sw.statusCode))
			if sw.statusCode >= 500 {
				span.RecordError(fmt.Errorf("HTTP Server Error %d", sw.statusCode))
			}
		})
	}
}
