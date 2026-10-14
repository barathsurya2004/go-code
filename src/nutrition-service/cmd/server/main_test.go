package main

import (
	"testing"

	"go.uber.org/fx"
)

func TestBuildApp(t *testing.T) {
	app := buildApp(
		fx.NopLogger,
	)
	if app == nil {
		t.Fatal("expected non-nil app")
	}
}
