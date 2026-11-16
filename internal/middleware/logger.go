package middleware

import (
	"log"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New initializes and returns a configured *zap.Logger
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
		// Fallback to basic logger if build fails
		return zap.NewExample()
	}
	return logger
}


func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("[%s] %s %s %d %s",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			time.Since(start),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
