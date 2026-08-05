package pkg

import (
	"testing"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func TestModule(t *testing.T) {
	var log *zap.Logger
	app := fx.New(
		Module,
		fx.Populate(&log),
	)

	if err := app.Err(); err != nil {
		t.Fatalf("expected no error initializing fx app with Module, got %v", err)
	}

	if log == nil {
		t.Fatal("expected populated zap logger, got nil")
	}
}
