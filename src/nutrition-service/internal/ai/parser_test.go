package ai

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestVisionAIParser(t *testing.T) {
	logger := zap.NewNop()
	parser := NewVisionAIParser(logger)
	ctx := context.Background()

	// 1. Empty image error
	_, err := parser.AnalyzeMealImage(ctx, nil, "image/jpeg")
	if err == nil {
		t.Error("expected error for empty image bytes")
	}

	// 2. Valid image analysis
	dummyImage := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x01, 0x02, 0x03, 0x04}
	items, err := parser.AnalyzeMealImage(ctx, dummyImage, "image/png")
	if err != nil {
		t.Fatalf("unexpected error analyzing image: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least 1 item detected")
	}
	if items[0].Calories <= 0 || items[0].ProteinGrams < 0 {
		t.Errorf("invalid extracted nutrition: %+v", items[0])
	}
	if len(items[0].Ingredients) == 0 {
		t.Errorf("expected ingredients detected for future recipe generator, got none")
	}

	// 3. Context cancelled
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = parser.AnalyzeMealImage(cancelledCtx, dummyImage, "image/png")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
