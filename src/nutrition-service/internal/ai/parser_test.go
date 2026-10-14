package ai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestVisionAIParser_FallbackAndValidation(t *testing.T) {
	logger := zap.NewNop()
	parser := NewVisionAIParser(logger)
	parser.apiKey = "" // Ensure fallback mode
	ctx := context.Background()

	// 1. Empty image error
	_, err := parser.AnalyzeMealImage(ctx, nil, "image/jpeg")
	if err == nil {
		t.Error("expected error for empty image bytes")
	}

	// 2. Valid image analysis with fallback
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
		t.Errorf("expected ingredients detected, got none")
	}

	// 3. Context cancelled
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = parser.AnalyzeMealImage(cancelledCtx, dummyImage, "image/png")
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestVisionAIParser_LiveGeminiSuccess(t *testing.T) {
	mockResponseJSON := `{
		"candidates": [
			{
				"content": {
					"parts": [
						{
							"text": "{\"items\": [{\"name\": \"Avocado Toast\", \"estimated_weight_grams\": 180, \"calories\": 320, \"protein_grams\": 9.5, \"carbs_grams\": 28, \"fat_grams\": 20, \"fiber_grams\": 6, \"sodium_mg\": 310, \"confidence_score\": 0.96, \"ingredients\": [\"Avocado\", \"Sourdough bread\"]}]}"
						}
					]
				}
			}
		]
	}`

	var receivedRequestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		receivedRequestBody = string(bodyBytes)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponseJSON))
	}))
	defer server.Close()

	logger := zap.NewNop()
	parser := NewVisionAIParser(logger)
	parser.apiKey = "test-key"
	parser.SetBaseURL(server.URL)
	parser.SetModel("test-model")

	dummyImage := []byte{0x01, 0x02, 0x03}
	items, err := parser.AnalyzeMealImage(context.Background(), dummyImage, "image/jpeg", "extra grilled chicken breast")
	if err != nil {
		t.Fatalf("unexpected error from gemini mock: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "Avocado Toast" {
		t.Errorf("expected Avocado Toast, got %s", items[0].Name)
	}
	if items[0].Calories != 320 {
		t.Errorf("expected 320 calories, got %f", items[0].Calories)
	}

	// Verify user notes were included in prompt
	if !strings.Contains(receivedRequestBody, "extra grilled chicken breast") {
		t.Errorf("expected prompt to contain user notes, got %s", receivedRequestBody)
	}
	// Verify calorie & protein calibration instructions
	if !strings.Contains(receivedRequestBody, "CALORIES (KCAL) - BE LENIENT") {
		t.Errorf("expected prompt to contain lenient calorie instructions")
	}
	if !strings.Contains(receivedRequestBody, "PROTEIN (G) - BE STRICT / CRITICAL") {
		t.Errorf("expected prompt to contain critical protein instructions")
	}
}

func TestVisionAIParser_GeminiErrorFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error": {"code": 503, "message": "High demand"}}`))
	}))
	defer server.Close()

	logger := zap.NewNop()
	parser := NewVisionAIParser(logger)
	parser.apiKey = "test-key"
	parser.SetBaseURL(server.URL)

	dummyImage := []byte{0x01, 0x02, 0x03, 0x04}
	// Should fall back gracefully to template matching instead of erroring out
	items, err := parser.AnalyzeMealImage(context.Background(), dummyImage, "image/jpeg")
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected fallback items")
	}
}
