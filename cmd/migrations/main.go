package main

import (
	"log"
	"os"
	"strconv"

	"github.com/Val-senseisama/payments/cmd/config"
	"github.com/Val-senseisama/payments/cmd/db"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg := config.InitConfig()

	database, err := db.NewPostgresSQLStorage(
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)
	if err != nil {
		log.Fatal("DB connection error: ", err)
	}

	driver, err := postgres.WithInstance(database, &postgres.Config{})
	if err != nil {
		log.Fatal("Migrate driver error: ", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://cmd/migrations/sql",
		"postgres",
		driver,
	)

	if err != nil {
		log.Fatal("Migrate error: ", err)
	}

	if len(os.Args) < 2 {
		log.Fatal("Please provide a command: 'up', 'down', or 'force <version>'")
	}

	cmd := os.Args[1]

	if cmd == "up" {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal("Migration up error: ", err)
		}
		log.Println("Migrations applied successfully!")
	}

	if cmd == "down" {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatal("Migration down error: ", err)
		}
		log.Println("Migrations rolled back successfully!")
	}

	if cmd == "force" {
		if len(os.Args) < 3 {
			log.Fatal("Please provide version to force: e.g. go run cmd/migrations/main.go force 1")
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatal("Invalid version number: ", err)
		}
		if err := m.Force(version); err != nil {
			log.Fatal("Force error: ", err)
		}
		log.Printf("Successfully forced migration version to %d!\n", version)
	}
}
