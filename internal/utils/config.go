package utils

import (
	"log"
	"os"
	

	"github.com/joho/godotenv" // for local .env file
	"github.com/knadh/koanf/v2"
)

var k = koanf.New(".")

type Config struct {
    Port  string    `koanf:"PORT"`
    DbUrl string `koanf:"DATABASE_URL"`
    Jwt   string `koanf:"JWT_SECRET"`
}

func MustLoad() Config {
    // Load .env file only in local dev (ignore error if file not found)
    _ = godotenv.Load(".env")

   port := os.Getenv("PORT")
   if port == ""{
    log.Fatalf("port is required")
    }

    database := os.Getenv("DATABASE_URL")
    if database == ""{
        log.Fatal("database url is require")
    }
    jwt := os.Getenv("JWT_SECRET")
    if jwt == ""{
        log.Fatal("jwt is require")
    }

    return Config{
        Port: port,
        DbUrl: database,
        Jwt: jwt,
    }
}
