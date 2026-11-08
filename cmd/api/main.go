package main

import (
	"context"
	"sync"
	"shopMe/internal/handler/routes"
	"shopMe/internal/middleware"
	"shopMe/internal/migration"
	"shopMe/internal/servers"
	"shopMe/internal/utils"
	"shopMe/internal/worker"

	colorlog "github.com/amarnathdbg101-coder/go/go-colorlog"
	"go.uber.org/zap"
)

func main() {
	colorlog.Info("Starting ShopSilo API server...", "version", "v2.0.0", "status", "initializing")

	logger := middleware.New()
	defer logger.Sync()

	cfg := utils.MustLoad()

	db, err := utils.ConnectDB(cfg.DBURL)
	if err != nil {
		colorlog.Error("Database connection failed", "error", err)
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer db.Close()

	// Automatically run embedded database migrations to keep schema up-to-date in production (e.g. Render)
	if err := migration.RunAutoMigrations(db); err != nil {
		colorlog.Warn("Auto-migration notice", "warning", err)
		logger.Warn("auto-migration notice", zap.Error(err))
	} else {
		colorlog.Info("Database migrations verified", "status", "up-to-date")
		logger.Info("database auto-migrations verified successfully")
	}
	migration.EnsureSchemaColumns(db)

	router := routes.RouteSetup(db, logger)

	// Context and WaitGroup for background concurrent workers
	appCtx, stopApp := context.WithCancel(context.Background())
	defer stopApp()
	var workerWG sync.WaitGroup

	// Background Concurrency Workers
	workerWG.Add(3)
	go func() {
		defer workerWG.Done()
		worker.StartReservationCleaner(appCtx, db, logger)
	}()
	go func() {
		defer workerWG.Done()
		worker.StartBackupWorker(appCtx, db, logger)
	}()
	go func() {
		defer workerWG.Done()
		worker.StartOutboxWorker(appCtx, db, logger)
	}()

	server := servers.NewServer(utils.MustLoad().Port, router)

	servers.StartServer(server, logger)

	// Channel to listen for interrupt signals for graceful shutdown with worker coordination
	servers.GracefulShutdown(server, stopApp, logger, &workerWG)
}
