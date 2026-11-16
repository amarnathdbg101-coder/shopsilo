# SPEC: go-w3ctrace

## 1. Scope & Purpose
`go-w3ctrace` implements the W3C TraceContext Recommendation (traceparent and tracestate) to provide distributed tracing across Go microservices and HTTP boundaries. It provides span creation, context propagation, error logging, and standard HTTP middleware.

It does NOT include complex metric aggregations or distributed log storage (use `go-metrics` and `go-auditlog`).

## 2. Public API Contract
```go
type SpanContext struct { TraceID, SpanID, TraceFlags, TraceState string }
type Span struct { ... }
type Tracer struct { ... }
type Exporter interface { Export(span *Span) error }

func ParseTraceparent(header string) (SpanContext, error)
func (sc SpanContext) String() string
func StartSpan(ctx context.Context, name string) (context.Context, *Span)
func SpanFromContext(ctx context.Context) *Span
func InjectHTTP(ctx context.Context, header http.Header)
func ExtractHTTP(header http.Header) (SpanContext, error)
func Middleware(tracer *Tracer) func(http.Handler) http.Handler
```

## 3. Concurrency & Memory Guarantees
- Span attributes and status mutations are protected by `sync.Mutex`.
- Span IDs and Trace IDs are generated using `crypto/rand` CSPRNG.
- Zero external runtime heap allocations for trace context extraction.

## 4. How to Extend
- Implement an OpenTelemetry collector HTTP/gRPC exporter.
- Implement Datadog / Jaeger trace exporter adapters.
