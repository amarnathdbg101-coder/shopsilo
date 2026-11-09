package main

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/stdlib"
	"shopMe/internal/app"
	"shopMe/internal/handler/routes"
	"shopMe/internal/middleware"
	"shopMe/internal/migration"
	"shopMe/internal/reuse"
	"shopMe/internal/servers"
	"shopMe/internal/utils"

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

	// Initialize asynchronous PostgreSQL batch audit logging engine
	if err := reuse.InitAuditEngine(stdlib.OpenDBFromPool(db)); err != nil {
		logger.Warn("audit engine failed to initialize with postgres store, falling back to memory", zap.Error(err))
	}
	defer reuse.CloseAuditEngine()

	// 1. Centralized Application Container (Repositories, Services, Controllers, Features)
	appContainer := app.NewContainer(db, logger)

	// 2. Centralized HTTP Router
	router := routes.RouteSetup(appContainer)

	// 3. Centralized Background Workers Coordination
	appCtx, stopApp := context.WithCancel(context.Background())
	defer stopApp()
	var workerWG sync.WaitGroup

	appContainer.StartWorkers(appCtx, &workerWG)

	// 4. Server Start & Graceful Shutdown
	server := servers.NewServer(utils.MustLoad().Port, router)
	servers.StartServer(server, logger)
	servers.GracefulShutdown(server, stopApp, logger, &workerWG)
}
