package main

import (
	"context"
	"shopMe/internal/handler/routes"
	"shopMe/internal/middleware"
	"shopMe/internal/migration"
	"shopMe/internal/servers"
	"shopMe/internal/utils"
	"shopMe/internal/worker"

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

	// Automatically run database migrations to keep schema up-to-date in production (e.g. Render)
	migration.EnsureSchemaColumns(db)
	if err := migration.RunAutoMigrations(cfg.DBURL); err != nil {
		logger.Warn("auto-migration notice", zap.Error(err))
	} else {
		logger.Info("database auto-migrations verified successfully")
	}

	router := routes.RouteSetup(db, logger)

	// Context for background concurrent workers
	appCtx, stopApp := context.WithCancel(context.Background())
	defer stopApp()
	// Background Concurrency Workers
	go worker.StartReservationCleaner(appCtx, db, logger)
	go worker.StartBackupWorker(appCtx, db, logger)
	go worker.StartOutboxWorker(appCtx, db, logger)

	server := servers.NewServer(utils.MustLoad().Port, router)

	servers.StartServer(server, logger)

	// Channel to listen for interrupt signals for graceful shutdown
	servers.GracefulShutdown(server, stopApp, logger)
}
