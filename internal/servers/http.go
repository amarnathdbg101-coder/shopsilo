//Package servers use for server build,maintain and terminate.
package servers

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "go.uber.org/zap"
)

func NewServer(port string, handler http.Handler) *http.Server {
    return &http.Server{
        Addr:              fmt.Sprintf(":%s", port),
        Handler:           handler,
        ReadHeaderTimeout: 5 * time.Second,
        ReadTimeout:       15 * time.Second,
        WriteTimeout:      30 * time.Second,
        IdleTimeout:       60 * time.Second,
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

func GracefulShutdown(server *http.Server, stopApp context.CancelFunc, logger *zap.Logger) {
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

    logger.Info("server exited cleanly")
}
