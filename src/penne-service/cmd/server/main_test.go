package main

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/fx"
)

func TestBuildApp(t *testing.T) {
	app := buildApp(
		fx.Replace(func() (*sql.DB, error) {
			db, _, err := sqlmock.New()
			return db, err
		}),
		fx.NopLogger,
	)

	if err := app.Err(); err != nil {
		t.Fatalf("expected buildApp to initialize without error, got %v", err)
	}
}
