package db

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
	"go.uber.org/fx"
)

type Config struct {
	DSN string
}

func CreateConfig() Config {
	return Config{
		DSN: "postgres://postgres:postgres@localhost:5432/penne_app?sslmode=disable",
	}
}

func NewDb(lc fx.Lifecycle, cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return db.PingContext(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return db.Close()
		},
	})

	return db, nil
}
