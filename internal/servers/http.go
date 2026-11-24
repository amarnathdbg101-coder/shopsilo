//Package servers use for server build,maintain and terminate.
package servers

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "go.uber.org/zap"
)

func NewServer(port string, handler http.Handler) *http.Server {
    return &http.Server{
        Addr:              fmt.Sprintf("0.0.0.0:%s", port),
        Handler:           handler,
        ReadHeaderTimeout: 10 * time.Second,
        ReadTimeout:       300 * time.Second, // 5 minutes for large CSV/Excel uploads (100k+ rows)
        WriteTimeout:      600 * time.Second, // 10 minutes for ingesting lakhs of products in DB
        IdleTimeout:       300 * time.Second,
        MaxHeaderBytes:    1 << 20,
    }
}

func StartServer(server *http.Server, logger *zap.Logger) {
    go func() {
        logger.Info("server starting", zap.String("port", server.Addr))
        if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            logger.Fatal("server failed", zap.Error(err))
        }
    }()
}

func GracefulShutdown(server *http.Server, stopApp context.CancelFunc, logger *zap.Logger, wg ...*sync.WaitGroup) {
    // Channel to listen for interrupt signals
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

    <-stop
    logger.Info("shutting down server gracefully...")
    stopApp() // stop background workers

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        logger.Error("server forced to shutdown", zap.Error(err))
    }

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
        case <-time.After(5 * time.Second):
            logger.Warn("background workers termination timed out, exiting")
        }
    }

    logger.Info("server exited cleanly")
}
