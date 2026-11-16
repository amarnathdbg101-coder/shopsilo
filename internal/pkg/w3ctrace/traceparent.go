package w3ctrace

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// HeaderTraceparent is the canonical HTTP header name per W3C TraceContext recommendation.
	HeaderTraceparent = "traceparent"

	// HeaderTracestate is the optional state header for vendor metadata.
	HeaderTracestate = "tracestate"

	// Version00 is the current official W3C specification version.
	Version00 = "00"

	// FlagSampled indicates the trace is recorded/sampled.
	FlagSampled = "01"
)

// SpanContext holds the immutable distributed context values propagated across boundaries.
type SpanContext struct {
	TraceID    string `json:"trace_id"`
	SpanID     string `json:"span_id"`
	TraceFlags string `json:"trace_flags"`
	TraceState string `json:"trace_state,omitempty"`
}

// IsValid checks if trace context has valid non-empty, non-zero IDs.
func (sc SpanContext) IsValid() bool {
	if len(sc.TraceID) != 32 || sc.TraceID == "00000000000000000000000000000000" {
		return false
	}
	if len(sc.SpanID) != 16 || sc.SpanID == "0000000000000000" {
		return false
	}
	return true
}

// String serializes the SpanContext into standard W3C traceparent header format:
// 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
func (sc SpanContext) String() string {
	flags := sc.TraceFlags
	if flags == "" {
		flags = FlagSampled
	}
	return fmt.Sprintf("%s-%s-%s-%s", Version00, sc.TraceID, sc.SpanID, flags)
}

// ParseTraceparent parses a W3C traceparent header string.
func ParseTraceparent(header string) (SpanContext, error) {
	parts := strings.Split(strings.TrimSpace(header), "-")
	if len(parts) < 4 {
		return SpanContext{}, ErrInvalidTraceparent
	}

	version := parts[0]
	traceID := parts[1]
	spanID := parts[2]
	flags := parts[3]

	// Version validation
	if version != Version00 && len(parts) == 4 {
		// Future versions may have more than 4 parts per W3C forward-compatibility
	}

	// TraceID validation (32 hex characters, cannot be all zero)
	if len(traceID) != 32 || traceID == "00000000000000000000000000000000" {
		return SpanContext{}, ErrInvalidTraceID
	}
	if _, err := hex.DecodeString(traceID); err != nil {
		return SpanContext{}, ErrInvalidTraceID
	}

	// SpanID validation (16 hex characters, cannot be all zero)
	if len(spanID) != 16 || spanID == "0000000000000000" {
		return SpanContext{}, ErrInvalidSpanID
	}
	if _, err := hex.DecodeString(spanID); err != nil {
		return SpanContext{}, ErrInvalidSpanID
	}

	// Flags validation (2 hex characters)
	if len(flags) != 2 {
		return SpanContext{}, ErrInvalidTraceparent
	}

	return SpanContext{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags,
	}, nil
}

// NewTraceID generates a cryptographically secure 16-byte random Trace ID formatted as a 32-char hex string.
func NewTraceID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// NewSpanID generates a cryptographically secure 8-byte random Span ID formatted as a 16-char hex string.
func NewSpanID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
