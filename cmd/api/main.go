package main

import (
	"log"
	"net/http"
	"shopMe/internal/routes"
	"shopMe/internal/utils"
)

func main() {
	cfg := utils.MustLoad()

	// 1. Connect to Neon DB
	db := utils.ConnectDB(cfg.DatabaseUrl)
	defer db.Close()

	// 2. Initialize Firebase Admin SDK
	firebaseAuth := utils.InitFirebase()

	// 3. Setup Routes
	router := routes.RouteSetup(db, firebaseAuth)
   
	log.Printf("Server starting on port %s...\n", cfg.Port)
	if err := http.ListenAndServe(":" + cfg.Port, router); err != nil {
		log.Fatalf("Server failed to start: %v\n", err)
	}
}
