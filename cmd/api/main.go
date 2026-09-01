package main

import (
	"fmt"
	"net/http"
	"shopMe/internal/logger"
	"shopMe/internal/routes"
	"shopMe/internal/utils"

	"go.uber.org/zap"
)

func main() {

	cfg := utils.MustLoad()
	logger := logger.New()
	db := utils.ConnectDB(cfg.DbUrl)
	defer db.Close()

	router := routes.RouteSetup(db, logger)

	logger.Info("server starting", zap.String("port",cfg.Port))
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), router); err != nil && err != http.ErrServerClosed {
		logger.Fatal("server failed", zap.Error(err))
	}
}
