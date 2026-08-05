package logger

import (
	"testing"
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
