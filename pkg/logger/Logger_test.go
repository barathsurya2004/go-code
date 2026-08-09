package logger

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	log, err := NewLogger()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNewLoggerWithConfig_Error(t *testing.T) {
	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{"/invalid_path/nonexistent.log"}
	log, err := NewLoggerWithConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid config, got nil")
	}
	if log != nil {
		t.Fatal("expected nil logger on error")
	}
}
