package main

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/fx"
)

// TestCadenceAppModule verifies the fx dependency graph compiles and resolves
// successfully when a mock *sql.DB is provided.
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

// TestBuildAppWithExtraOption verifies that extra fx.Options passed to buildApp
// are correctly merged into the final application.
func TestBuildAppWithExtraOption(t *testing.T) {
	type sentinel struct{}
	var got sentinel

	app := buildApp(
		fx.Replace(func() (*sql.DB, error) {
			dbMock, _, err := sqlmock.New()
			return dbMock, err
		}),
		fx.Provide(func() sentinel { return sentinel{} }),
		fx.Populate(&got),
	)
	if err := app.Err(); err != nil {
		t.Fatalf("expected no error with extra option, got %v", err)
	}
}

// TestMainFunction verifies that main() invokes appFactory().Run() and completes.
func TestMainFunction(t *testing.T) {
	orig := appFactory
	defer func() { appFactory = orig }()

	appFactory = func(opts ...fx.Option) *fx.App {
		return fx.New(
			fx.NopLogger,
			fx.Invoke(func(lc fx.Lifecycle, s fx.Shutdowner) {
				lc.Append(fx.Hook{
					OnStart: func(context.Context) error {
						go func() {
							_ = s.Shutdown()
						}()
						return nil
					},
				})
			}),
		)
	}

	main()
}
