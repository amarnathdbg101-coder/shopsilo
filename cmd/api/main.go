package main

import (
	"context"
	"sync"

	"shopMe/internal/app"
	"shopMe/internal/handler/routes"
	"shopMe/internal/middleware"
	"shopMe/internal/migration"
	"shopMe/internal/servers"
	"shopMe/internal/utils"

	"go.uber.org/zap"
)

func main() {
	logger := middleware.New()
	defer logger.Sync()

	logger.Info("Starting ShopSilo API server...", zap.String("version", "v2.0.0"), zap.String("status", "initializing"))

	cfg := utils.MustLoad()

	db, err := utils.ConnectDB(cfg.DBURL)
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer db.Close()

	// Automatically run embedded database migrations to keep schema up-to-date in production (e.g. Render)
	if err := migration.RunAutoMigrations(db); err != nil {
		logger.Warn("auto-migration notice", zap.Error(err))
	} else {
		logger.Info("database auto-migrations verified successfully")
	}
	migration.EnsureSchemaColumns(db)

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
