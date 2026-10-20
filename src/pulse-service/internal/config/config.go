package config

import (
	"os"

	"go.uber.org/fx"
)

type Config struct {
	Port        string
	DatabaseURL string
}

func NewConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/pulse_app?sslmode=disable"
	}

	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
	}
}

var Module = fx.Provide(
	NewConfig,
)
