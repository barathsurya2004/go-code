package activities

import (
	"context"
	"testing"
)

func TestHelloWorldActivity(t *testing.T) {
	res, err := HelloWorldActivity(context.Background(), "Barath")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Hello Barath\n"
	if res != expected {
		t.Errorf("expected %q, got %q", expected, res)
	}
}
