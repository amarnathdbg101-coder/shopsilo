// Package servers provides HTTP server construction, lifecycle management, and phased termination.
package servers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	goshutdown "github.com/amarnathdbg101-coder/go/go-shutdown"
	"shopMe/internal/worker"
	"go.uber.org/zap"
)

// NewServer constructs an optimized net/http Server with production timeouts.
func NewServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%s", port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

// StartServer launches the HTTP server in a non-blocking goroutine.
func StartServer(server *http.Server, logger *zap.Logger) {
	go func() {
		logger.Info("server starting", zap.String("port", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()
}

// BuildShutdownCoordinator constructs an orchestrated, phased shutdown coordinator via zero-dep go-shutdown:
// Phase 1 (PhaseHTTP): Ingress HTTP server drain (stops taking new connections, finishes in-flight requests).
// Phase 2 (PhaseWorkers): Background worker drain (cancels context and waits for worker WaitGroups).
func BuildShutdownCoordinator(server *http.Server, stopApp context.CancelFunc, logger *zap.Logger, wg ...*sync.WaitGroup) *goshutdown.Coordinator {
	coordinator := goshutdown.New(15 * time.Second)

	// Phase 1: HTTP Ingress Stop
	coordinator.RegisterHTTP("http_ingress", func(ctx context.Context) error {
		logger.Info("shutting down HTTP server gracefully...")
		return server.Shutdown(ctx)
	})

	// Phase 2: Background Workers Drain
	coordinator.RegisterWorker("background_workers", func(ctx context.Context) error {
		logger.Info("stopping background workers context...")
		stopApp()

		if len(wg) > 0 && wg[0] != nil {
			logger.Info("waiting for background workers to complete cleanly...")
			done := make(chan struct{})
			go func() {
				wg[0].Wait()
				close(done)
			}()
			select {
			case <-done:
				logger.Info("all background workers terminated gracefully")
			case <-ctx.Done():
				logger.Warn("background workers termination timed out")
			}
		}

		logger.Info("draining bounded worker task pool...")
		worker.DrainTaskPool()
		logger.Info("worker task pool drained successfully")
		return nil
	})

	return coordinator
}

// GracefulShutdown coordinates phased, orderly application termination via zero-dep go-shutdown.
func GracefulShutdown(server *http.Server, stopApp context.CancelFunc, logger *zap.Logger, wg ...*sync.WaitGroup) {
	coordinator := BuildShutdownCoordinator(server, stopApp, logger, wg...)

	logger.Info("listening for OS termination signals (SIGINT, SIGTERM)...")
	if err := coordinator.Wait(); err != nil {
		logger.Error("phased shutdown completed with warnings", zap.Error(err))
	} else {
		logger.Info("server exited cleanly via phased shutdown coordinator")
	}
}
