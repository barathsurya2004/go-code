package db

import (
	"context"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"
)

func TestCreateConfig(t *testing.T) {
	cfg := CreateConfig()
	if cfg.DSN == "" {
		t.Error("expected non-empty DSN in CreateConfig")
	}
}

func TestNewDb(t *testing.T) {
	lc := fxtest.NewLifecycle(t)
	cfg := Config{DSN: "postgres://postgres:postgres@localhost:5432/penne_app?sslmode=disable"}

	database, err := NewDb(lc, cfg)
	if err != nil {
		t.Fatalf("expected no error from NewDb, got %v", err)
	}
	if database == nil {
		t.Fatal("expected non-nil *sql.DB")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Execute OnStart (will attempt ping)
	_ = lc.Start(ctx)

	// Execute OnStop (will close database)
	if err := lc.Stop(ctx); err != nil {
		t.Errorf("unexpected error on Stop: %v", err)
	}
}
