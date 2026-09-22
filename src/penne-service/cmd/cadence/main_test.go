package main

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/fx"
)

func TestCadenceAppModule(t *testing.T) {
	app := buildApp(
		fx.Replace(func() (*sql.DB, error) {
			dbMock, _, err := sqlmock.New()
			return dbMock, err
		}),
	)
	if err := app.Err(); err != nil {
		t.Fatalf("expected no error initializing cadence app modules, got %v", err)
	}
}
