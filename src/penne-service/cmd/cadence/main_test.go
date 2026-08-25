package main

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/cadence"
	"github.com/barathsurya2004/go-code/penne-service/internal/db"
	"github.com/barathsurya2004/go-code/pkg"
	"go.uber.org/fx"
)

func TestCadenceAppModule(t *testing.T) {
	app := fx.New(
		pkg.Module,
		db.Module,
		cadence.Module,
		fx.Replace(func() (*sql.DB, error) {
			dbMock, _, err := sqlmock.New()
			return dbMock, err
		}),
	)
	if err := app.Err(); err != nil {
		t.Fatalf("expected no error initializing cadence app modules, got %v", err)
	}
}
