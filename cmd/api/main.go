package main

import (
	"fmt"
	"net/http"
	"shopMe/internal/handler/routes"
	"shopMe/internal/utils"

	"go.uber.org/zap"
)

func main() {

	logger := utils.New()
	db, _ := utils.ConnectDB(utils.MustLoad().DbUrl)
	defer db.Close()
	router := routes.RouteSetup(db, logger)

	logger.Info("server starting", zap.String("port", utils.MustLoad().Port))
	if err := http.ListenAndServe(fmt.Sprintf(":%s", utils.MustLoad().Port), router); err != nil && err != http.ErrServerClosed {
		logger.Fatal("server failed", zap.Error(err))
	}
}
