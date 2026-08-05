package db

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/fx"
)

func TestModule(t *testing.T) {
	var userRepo core.UserRepository
	var txnRepo core.TransactionRepository

	app := fx.New(
		Module,
		// Replace sql.DB provider with a mock DB to prevent network connection during module verification
		fx.Replace(func() (*sql.DB, error) {
			db, _, err := sqlmock.New()
			return db, err
		}),
		fx.Populate(&userRepo, &txnRepo),
	)

	if userRepo == nil || txnRepo == nil {
		t.Fatal("expected populated repositories from db.Module")
	}
	_ = app
}
