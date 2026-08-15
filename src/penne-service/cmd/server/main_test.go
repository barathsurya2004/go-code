package main

import (
	"testing"
)

func TestBuildApp(t *testing.T) {
	app := buildApp()
	if app == nil {
		t.Fatal("expected non-nil fx.App from buildApp")
	}
	if err := app.Err(); err != nil {
		t.Fatalf("expected clean app initialization, got %v", err)
	}
}
