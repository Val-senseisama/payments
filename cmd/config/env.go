package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	DBUser               string
	DBPassword           string
	DBName               string
	DBPort               string
	DBHost               string
	JWTSecret            string
	JWTRefreshSecret     string
	JWTAccessExpiration  time.Duration
	JWTRefreshExpiration time.Duration
	RedisAddr            string
	RedisPassword        string
	RedisDB              int
}

func InitConfig() Config {
	godotenv.Load()

	return Config{
		Port:                 getEnv("PORT", "8080"),
		DBUser:               getEnv("DB_USER", "postgres"),
		DBPassword:           getEnv("DB_PASSWORD", ""),
		DBName:               getEnv("DB_NAME", "payments"),
		DBPort:               getEnv("DB_PORT", "5432"),
		DBHost:               getEnv("DB_HOST", "127.0.0.1"),
		JWTSecret:            getEnv("JWT_SECRET", "super-secret-access-token-key-change-in-prod"),
		JWTRefreshSecret:     getEnv("JWT_REFRESH_SECRET", "super-secret-refresh-token-key-change-in-prod"),
		JWTAccessExpiration:  time.Duration(getEnvInt("JWT_ACCESS_EXPIRATION_SECONDS", 900)) * time.Second,   // 15 mins
		JWTRefreshExpiration: time.Duration(getEnvInt("JWT_REFRESH_EXPIRATION_SECONDS", 604800)) * time.Second, // 7 days
		RedisAddr:            getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:        getEnv("REDIS_PASSWORD", ""),
		RedisDB:              int(getEnvInt("REDIS_DB", 0)),
	}
}

func getEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int64) int64 {
	value := getEnv(key, "")
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}
