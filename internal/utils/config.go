package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	Env         string
	DatabaseUrl string
	JwtSecret   string
}

func MustLoad() Config {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("Port is required")
	}
	env := os.Getenv("ENV")
	if env == "" {
		log.Fatal("env is required")
	}
	database := os.Getenv("DATABASEURL")
	if database == "" {
		log.Fatal("databaseUrl is required")
	}
	jwtsecret := os.Getenv("JWT_SECRET")
	if jwtsecret == ""{
		log.Fatal("jwt secret is required")
	}

	return Config{
		Port:        port,
		Env:         env,
		DatabaseUrl: database,
		JwtSecret: jwtsecret,
	}
}
