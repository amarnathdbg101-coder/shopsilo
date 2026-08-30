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
	log := logger.New()
	db := utils.ConnectDB(cfg.DbUrl)
	defer db.Close()

	router := routes.RouteSetup(db)

	log.Info("server starting",zap.Int("port",cfg.Port))
	if err := http.ListenAndServe(fmt.Sprintf(":%d",cfg.Port), router); err != nil && err != http.ErrServerClosed {
		log.Fatal("server failed", zap.Error(err))
	}
}
