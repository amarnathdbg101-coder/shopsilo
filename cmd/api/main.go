package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/routes"
	"shopMe/internal/middleware"
	"shopMe/internal/utils"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	logger := middleware.New()
	defer logger.Sync()

	cfg := utils.MustLoad()

	db, err := utils.ConnectDB(cfg.DBURL)
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer db.Close()

	router := routes.RouteSetup(db, logger)

	// Context for background concurrent workers
	appCtx, stopApp := context.WithCancel(context.Background())
	defer stopApp()

	// Background Concurrency Worker: Periodically expires stale holds and releases reserved inventory
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()

		resRepo := repository.NewReservationRepo(db, logger)

		for {
			select {
			case <-appCtx.Done():
				logger.Info("background reservation cleaner stopped cleanly")
				return
			case <-ticker.C:
				cleanCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				if err := resRepo.ExpireStaleReservations(cleanCtx); err != nil {
					logger.Warn("background reservation cleaner encountered an issue", zap.Error(err))
				}
				cancel()
			}
		}
	}()

	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", cfg.Port),
		Handler:        router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Channel to listen for interrupt signals for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", zap.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	<-stop
	logger.Info("shutting down server gracefully...")
	stopApp() // Signal background worker goroutine to terminate

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited cleanly")
}
