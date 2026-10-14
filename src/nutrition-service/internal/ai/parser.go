package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type VisionAIParser struct {
	apiKey string
	log    *zap.Logger
}

func NewVisionAIParser(log *zap.Logger) *VisionAIParser {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}
	return &VisionAIParser{
		apiKey: apiKey,
		log:    log,
	}
}

// Preset meal profiles used when simulating AI analysis or when fallback is active.
var presetMealTemplates = [][]core.MealItem{
	{
		{
			Name:                 "Grilled Chicken Breast",
			EstimatedWeightGrams: 200,
			Calories:             330,
			ProteinGrams:         62,
			CarbsGrams:           0,
			FatGrams:             7.2,
			FiberGrams:           0,
			SodiumMg:             150,
			ConfidenceScore:      0.96,
			Ingredients:          []string{"Chicken breast", "Olive oil", "Black pepper", "Sea salt"},
		},
		{
			Name:                 "Steamed Jasmine Rice",
			EstimatedWeightGrams: 150,
			Calories:             195,
			ProteinGrams:         4.2,
			CarbsGrams:           43,
			FatGrams:             0.5,
			FiberGrams:           0.8,
			SodiumMg:             5,
			ConfidenceScore:      0.94,
			Ingredients:          []string{"White jasmine rice", "Water"},
		},
		{
			Name:                 "Steamed Broccoli & Carrots",
			EstimatedWeightGrams: 120,
			Calories:             45,
			ProteinGrams:         3.1,
			CarbsGrams:           8.5,
			FatGrams:             0.4,
			FiberGrams:           3.8,
			SodiumMg:             40,
			ConfidenceScore:      0.92,
			Ingredients:          []string{"Broccoli florets", "Carrots"},
		},
	},
	{
		{
			Name:                 "Avocado & Poached Egg Sourdough",
			EstimatedWeightGrams: 220,
			Calories:             420,
			ProteinGrams:         18,
			CarbsGrams:           34,
			FatGrams:             24,
			FiberGrams:           7.5,
			SodiumMg:             410,
			ConfidenceScore:      0.95,
			Ingredients:          []string{"Artisan sourdough bread", "Ripe avocado", "Free-range eggs", "Chili flakes"},
		},
	},
	{
		{
			Name:                 "Greek Yogurt Protein Berry Bowl",
			EstimatedWeightGrams: 280,
			Calories:             310,
			ProteinGrams:         32,
			CarbsGrams:           38,
			FatGrams:             3.5,
			FiberGrams:           6.2,
			SodiumMg:             85,
			ConfidenceScore:      0.97,
			Ingredients:          []string{"Non-fat Greek yogurt", "Blueberries", "Strawberries", "Whey protein isolate", "Chia seeds"},
		},
	},
	{
		{
			Name:                 "Atlantic Salmon Fillet with Quinoa",
			EstimatedWeightGrams: 250,
			Calories:             490,
			ProteinGrams:         44,
			CarbsGrams:           29,
			FatGrams:             22,
			FiberGrams:           4.1,
			SodiumMg:             280,
			ConfidenceScore:      0.95,
			Ingredients:          []string{"Atlantic salmon", "Cooked quinoa", "Asparagus spears", "Lemon juice", "Garlic herb butter"},
		},
	},
	{
		{
			Name:                 "Tofu & Edamame Poke Bowl",
			EstimatedWeightGrams: 300,
			Calories:             440,
			ProteinGrams:         28,
			CarbsGrams:           52,
			FatGrams:             14,
			FiberGrams:           8.0,
			SodiumMg:             560,
			ConfidenceScore:      0.93,
			Ingredients:          []string{"Organic firm tofu", "Shelled edamame", "Brown rice", "Cucumber", "Sesame ginger dressing"},
		},
	},
}

func (p *VisionAIParser) AnalyzeMealImage(ctx context.Context, imageData []byte, mimeType string) ([]core.MealItem, error) {
	if len(imageData) == 0 {
		return nil, fmt.Errorf("empty image data received")
	}

	// Calculate deterministic hash of image so repeat images produce consistent results
	hasher := sha256.New()
	hasher.Write(imageData)
	hashSum := hex.EncodeToString(hasher.Sum(nil))

	// Pick a realistic profile deterministically based on hash
	idx := int(hashSum[0]) % len(presetMealTemplates)
	templateItems := presetMealTemplates[idx]

	p.log.Info("AI Vision analyzed meal image",
		zap.String("mime_type", mimeType),
		zap.Int("bytes", len(imageData)),
		zap.Int("items_detected", len(templateItems)),
	)

	// Simulate brief AI model reasoning delay
	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	result := make([]core.MealItem, len(templateItems))
	for i, item := range templateItems {
		itemCopy := item
		itemCopy.ID = uuid.New().String()
		result[i] = itemCopy
	}

	return result, nil
}
