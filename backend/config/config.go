package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config keeps backend runtime configuration.
type Config struct {
	ServerPort                 string
	MigrationsDir              string
	DBHost                     string
	DBPort                     int
	DBUser                     string
	DBPassword                 string
	DBName                     string
	DBSSLMode                  string
	DBMaxConns                 int
	DBMinConns                 int
	DBMaxConnLifetimeMinutes   int
	DBMaxConnIdleMinutes       int
	DBHealthCheckPeriodSeconds int
	QueueWorkerCount           int
	QueueBufferSize            int
	JWTSecret                  string
	JWTIssuer                  string
	JWTAudience                string
	JWTExpiryMinutes           int
}

// Load reads environment values from .env and process env.
func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		ServerPort:                 getEnv("SERVER_PORT", "8080"),
		MigrationsDir:              getEnv("MIGRATIONS_DIR", "migrations"),
		DBHost:                     getEnv("DB_HOST", "localhost"),
		DBPort:                     getEnvAsInt("DB_PORT", 5432),
		DBUser:                     getEnv("DB_USER", "postgres"),
		DBPassword:                 getEnv("DB_PASSWORD", "admin"),
		DBName:                     getEnv("DB_NAME", "recurin"),
		DBSSLMode:                  getEnv("DB_SSLMODE", "disable"),
		DBMaxConns:                 getEnvAsInt("DB_MAX_CONNS", 30),
		DBMinConns:                 getEnvAsInt("DB_MIN_CONNS", 5),
		DBMaxConnLifetimeMinutes:   getEnvAsInt("DB_MAX_CONN_LIFETIME_MINUTES", 30),
		DBMaxConnIdleMinutes:       getEnvAsInt("DB_MAX_CONN_IDLE_MINUTES", 15),
		DBHealthCheckPeriodSeconds: getEnvAsInt("DB_HEALTH_CHECK_PERIOD_SECONDS", 30),
		QueueWorkerCount:           getEnvAsInt("QUEUE_WORKER_COUNT", 8),
		QueueBufferSize:            getEnvAsInt("QUEUE_BUFFER_SIZE", 128),
		JWTSecret:                  getEnv("JWT_SECRET", ""),
		JWTIssuer:                  getEnv("JWT_ISSUER", "recurin"),
		JWTAudience:                getEnv("JWT_AUDIENCE", "recurin-users"),
		JWTExpiryMinutes:           getEnvAsInt("JWT_EXPIRY_MINUTES", 60),
	}

	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required (set it in backend/.env)")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); strings.TrimSpace(value) != "" {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
