package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	StaticDir       string
	DB              string
	LogLevel        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	JWTSecret       string
	Password        string
}

func Load() (*Config, error) {
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(jwtSecret) < 8 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 8 characters")
	}
	password := getEnv("TODO_PASSWORD", "")
	if password == "" {
		return nil, fmt.Errorf("TODO_PASSWORD is required")
	}

	port := getEnv("TODO_PORT", "7540")
	if _, err := strconv.Atoi(port); err != nil {
		return nil, fmt.Errorf("invalid PORT %q: %w", port, err)
	}
	readTimeout, err := getEnvDuration("READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	writeTimeout, err := getEnvDuration("WRITE_TIMEOUT", 15*time.Second)
	if err != nil {
		return nil, err
	}

	shutdownTimeout, err := getEnvDuration("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	dbUser := getEnv("DB_USER", "")
	dbPass := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "")
	dbHost := getEnv("DB_HOST", "")
	dbPort := getEnv("DB_PORT", "")

	db := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPass, dbHost, dbPort, dbName)
	if dbUser == "" || dbPass == "" || dbName == "" || dbHost == "" || dbPort == "" {
		return nil, fmt.Errorf("DB_* is not set")
	}

	return &Config{
		Port:            port,
		StaticDir:       getEnv("STATIC_DIR", "./web"),
		DB:              db,
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		ShutdownTimeout: shutdownTimeout,
		JWTSecret:       jwtSecret,
		Password:        password,
	}, nil
}

// getEnv env-halper
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvDuration env-halper
func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", key, v, err)
	}
	return d, nil
}
