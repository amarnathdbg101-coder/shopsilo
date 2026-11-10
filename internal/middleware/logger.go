// Package middleware provides HTTP middlewares for routing, security, and telemetry.
package middleware

import (
	"strings"
	"bytes"
	"net/http"
	"os"
	"time"

	colorlog "github.com/amarnathdbg101-coder/go/go-colorlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HTTPColorLogger is the structured ANSI-colored logger for HTTP ingress requests.
var HTTPColorLogger = colorlog.New(
	colorlog.WithCaller(false),
	colorlog.WithLevel(colorlog.LevelDebug),
)

// AppColorLogger is the general application logger powered by go-colorlog.
var AppColorLogger = colorlog.New(
	colorlog.WithCaller(true),
	colorlog.WithLevel(colorlog.LevelInfo),
)

// New initializes and returns a configured *zap.Logger for legacy backward compatibility.
func New() *zap.Logger {
	var config zap.Config
	if os.Getenv("ENV") == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := config.Build()
	if err != nil {
		return zap.NewExample()
	}
	return logger
}

// Logger intercepts HTTP requests and outputs structured color-coded telemetry
// including method, URL path, response status code, latency, client IP, and bytes written.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		latency := time.Since(start)
		ip := ExtractClientIP(r)
		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}

		keyvals := []any{
			"method", r.Method,
			"path", path,
			"status", wrapped.statusCode,
			"latency", latency.String(),
			"ip", ip,
			"bytes", wrapped.bytesWritten,
		}

		// Correlate with W3C distributed trace context
		if traceparent := wrapped.Header().Get("traceparent"); traceparent != "" {
			parts := strings.Split(traceparent, "-")
			if len(parts) >= 2 {
				keyvals = append(keyvals, "trace_id", parts[1])
			}
		}

		switch {
		case wrapped.statusCode >= 500:
			HTTPColorLogger.Error("HTTP Server Error", keyvals...)
		case wrapped.statusCode >= 400:
			HTTPColorLogger.Warn("HTTP Client Warning", keyvals...)
		default:
			HTTPColorLogger.Info("HTTP Request", keyvals...)
		}
	})
}

// responseWriter wraps http.ResponseWriter to safely intercept status code, bytes written,
// and delegates http.Flusher for WebSocket and SSE streaming support.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// SetHTTPLoggerOutput configures the destination writer for HTTP request logs (useful in tests).
func SetHTTPLoggerOutput(out *bytes.Buffer) {
	HTTPColorLogger = colorlog.New(
		colorlog.WithOutput(out),
		colorlog.WithCaller(false),
		colorlog.WithNoColor(true),
		colorlog.WithLevel(colorlog.LevelDebug),
	)
}

