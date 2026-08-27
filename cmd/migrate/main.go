package main

import (
	"log"
	"os"
	"shopMe/internal/utils"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2{
		log.Fatal("use migrate up or down")
		return
	}

	m, err := migrate.New("file://internal/migration",utils.MustLoad().DatabaseUrl)
	if err != nil{
		log.Fatal(err)
	}

	switch os.Args[1]{
	case "up":
		if err := m.Steps(1); err != nil && err != migrate.ErrNoChange{
		log.Fatal(err)
	}
	case "down":
		if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange{
		log.Fatal(err)
	}
	default:
		log.Fatalf("migration command wrong:%v",err)

	}
}
