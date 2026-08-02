package main

import (
	"context"
	"log"

	"github.com/Val-senseisama/payments/cmd/api"
	"github.com/Val-senseisama/payments/cmd/config"
	"github.com/Val-senseisama/payments/cmd/db"
	"github.com/Val-senseisama/payments/internal/domain/audit"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.InitConfig()

	// connect to db

	db, err := db.NewPostgresSQLStorage(
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName)

	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Error connecting to redis: ", err)
	}

	auditStore := audit.NewStore(db)

	server := api.NewAPIServer(":"+cfg.Port, cfg, db, rdb, auditStore)

	if err := server.Run(); err != nil {
		log.Fatal("Error running server: ", err)
	}
}
