package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	defaultGeminiModel   = "gemini-3-flash-preview"
)

type VisionAIParser struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	log        *zap.Logger
}

func loadNutritionEnv() {
	// Skip loading host .env files during unit tests so test suites remain hermetic and fast
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return
		}
	}
	if strings.HasSuffix(os.Args[0], ".test") {
		return
	}

	paths := []string{
		"src/nutrition-service/.env",
		".env",
		"../.env",
		"../../.env",
	}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					val = strings.Trim(val, `"'`)
					if os.Getenv(key) == "" {
						_ = os.Setenv(key, val)
					}
				}
			}
		}
	}
}

func NewVisionAIParser(log *zap.Logger) *VisionAIParser {
	loadNutritionEnv()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = defaultGeminiModel
	}
	baseURL := os.Getenv("GEMINI_BASE_URL")
	if baseURL == "" {
		baseURL = defaultGeminiBaseURL
	}

	return &VisionAIParser{
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{Timeout: 35 * time.Second},
		log:        log,
	}
}

// SetBaseURL overrides the base API endpoint (useful for tests and mocking).
func (p *VisionAIParser) SetBaseURL(url string) {
	p.baseURL = url
}

// SetModel overrides the Gemini model identifier.
func (p *VisionAIParser) SetModel(model string) {
	p.model = model
}

// SetHTTPClient overrides the HTTP client.
func (p *VisionAIParser) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}



// Gemini API Request/Response structs
type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inline_data,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string `json:"responseMimeType,omitempty"`
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiCandidatePart struct {
	Text string `json:"text"`
}

type geminiCandidateContent struct {
	Parts []geminiCandidatePart `json:"parts"`
}

type geminiCandidate struct {
	Content geminiCandidateContent `json:"content"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

type geminiNutritionItem struct {
	Name                 string   `json:"name"`
	EstimatedWeightGrams float64  `json:"estimated_weight_grams"`
	Calories             float64  `json:"calories"`
	ProteinGrams         float64  `json:"protein_grams"`
	CarbsGrams           float64  `json:"carbs_grams"`
	FatGrams             float64  `json:"fat_grams"`
	FiberGrams           float64  `json:"fiber_grams"`
	SodiumMg             float64  `json:"sodium_mg"`
	ConfidenceScore      float64  `json:"confidence_score"`
	Ingredients          []string `json:"ingredients"`
}

type geminiNutritionOutput struct {
	Items []geminiNutritionItem `json:"items"`
}

func (p *VisionAIParser) callGeminiVisionAPI(ctx context.Context, imageData []byte, mimeType, userNotes string) ([]core.MealItem, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("gemini api key is not configured")
	}

	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	b64Image := base64.StdEncoding.EncodeToString(imageData)

	var notesContext string
	if strings.TrimSpace(userNotes) != "" {
		notesContext = fmt.Sprintf("\nUSER DESCRIPTION / NOTES: \"%s\"\nPRIORITY: Strictly adhere to the user's notes above to identify the exact dish, ingredients, and preparation method (e.g., whether chicken, beef, tofu, rice, etc.).\n", strings.TrimSpace(userNotes))
	}

	prompt := fmt.Sprintf(`You are an expert clinical sports nutritionist and dietetic AI.
Analyze all food, beverages, and dishes present in this image.%s
MANDATORY MACRONUTRIENT CALIBRATION RULES:
1. CALORIES (KCAL) - BE LENIENT / CONSERVATIVE OVER-ESTIMATION:
   - Always round UP and estimate calories higher than the standard baseline.
   - Account generously for hidden cooking fats, oils, butter, marinades, dressings, and calorie-dense cooking methods.
   - It is vital that the user never undercounts calories so they can keep their caloric intake properly in check.

2. PROTEIN (G) - BE STRICT / CRITICAL UNDER-ESTIMATION:
   - Always be conservative and estimate protein lower than standard textbook values.
   - Only count high-bioavailability complete protein; do not overestimate protein.
   - It is vital that the user never overcounts protein.

3. PORTION SIZING & CONFIDENCE:
   - Estimate portion size in grams, carbohydrates (g), fat (g), fiber (g), and sodium (mg).
   - Confidence score must be a float between 0.0 and 1.0.

Return pure JSON matching this exact schema:
{
  "items": [
    {
      "name": "Grilled Chicken Breast",
      "estimated_weight_grams": 200,
      "calories": 380,
      "protein_grams": 48,
      "carbs_grams": 0,
      "fat_grams": 11,
      "fiber_grams": 0,
      "sodium_mg": 180,
      "confidence_score": 0.95,
      "ingredients": ["Chicken breast", "Olive oil", "Black pepper"]
    }
  ]
}
If no food or meal is detected, or the image cannot be identified as food, return:
{
  "items": []
}
`, notesContext)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
					{
						InlineData: &geminiInlineData{
							MimeType: mimeType,
							Data:     b64Image,
						},
					},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	modelsToTry := []string{p.model}
	if p.model == defaultGeminiModel {
		modelsToTry = append(modelsToTry, "gemini-3.8-flash", "gemini-3.5-flash", "gemini-flash-latest")
	}

	var lastErr error
	for _, targetModel := range modelsToTry {
		for attempt := 1; attempt <= 3; attempt++ {
			endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, targetModel, p.apiKey)
			httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
			if err != nil {
				return nil, fmt.Errorf("failed to create http request: %w", err)
			}
			httpReq.Header.Set("Content-Type", "application/json")

			p.log.Info("Gemini Vision AI: Calling model endpoint",
				zap.String("model", targetModel),
				zap.Int("attempt", attempt),
				zap.String("user_notes", userNotes),
			)

			resp, err := p.httpClient.Do(httpReq)
			if err != nil {
				lastErr = err
				p.log.Warn("Gemini Vision AI: network error, retrying",
					zap.String("model", targetModel),
					zap.Int("attempt", attempt),
					zap.Error(err),
				)
				select {
				case <-time.After(time.Duration(attempt) * time.Second):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}

			respBytes, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				lastErr = err
				continue
			}

			if resp.StatusCode == http.StatusServiceUnavailable {
				lastErr = fmt.Errorf("gemini api returned 503 high demand for model %s: %s", targetModel, string(respBytes))
				p.log.Warn("Gemini Vision AI: transient demand spike (503), retrying with backoff",
					zap.String("model", targetModel),
					zap.Int("attempt", attempt),
					zap.Int("status", resp.StatusCode),
				)
				select {
				case <-time.After(time.Duration(attempt) * 1200 * time.Millisecond):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}

			if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("gemini api returned status %d for model %s: %s", resp.StatusCode, targetModel, string(respBytes))
				p.log.Error("Gemini Vision AI: API non-OK response from Google",
					zap.String("model", targetModel),
					zap.Int("attempt", attempt),
					zap.Int("status_code", resp.StatusCode),
					zap.String("response_body", string(respBytes)),
				)
				break // Non-transient status or quota limit, try next candidate model
			}

			var geminiResp geminiResponse
			if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
				p.log.Error("Gemini Vision AI: failed to decode response JSON", zap.Error(err), zap.String("raw_response", string(respBytes)))
				return nil, fmt.Errorf("failed to decode gemini response: %w", err)
			}

			if geminiResp.Error != nil {
				p.log.Error("Gemini Vision AI: Google returned error in response", zap.String("message", geminiResp.Error.Message), zap.Int("code", geminiResp.Error.Code))
				return nil, fmt.Errorf("gemini error: %s (code %d)", geminiResp.Error.Message, geminiResp.Error.Code)
			}

			if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
				p.log.Error("Gemini Vision AI: returned no content candidates", zap.String("raw_response", string(respBytes)))
				return nil, fmt.Errorf("gemini returned no content candidates")
			}

			rawText := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
			if strings.HasPrefix(rawText, "```json") {
				rawText = strings.TrimPrefix(rawText, "```json")
				rawText = strings.TrimSuffix(rawText, "```")
			} else if strings.HasPrefix(rawText, "```") {
				rawText = strings.TrimPrefix(rawText, "```")
				rawText = strings.TrimSuffix(rawText, "```")
			}
			rawText = strings.TrimSpace(rawText)

			var output geminiNutritionOutput
			if err := json.Unmarshal([]byte(rawText), &output); err != nil {
				p.log.Error("Gemini Vision AI: failed to parse structured nutrition JSON", zap.Error(err), zap.String("generated_text", rawText))
				return nil, fmt.Errorf("failed to parse nutrition json from gemini output: %w", err)
			}

			if len(output.Items) == 0 {
				p.log.Warn("Gemini Vision AI: Model identified 0 food items in the image", zap.String("generated_text", rawText), zap.String("user_notes", userNotes))
				return nil, fmt.Errorf("unable to identify food in image: gemini vision ai detected 0 food items")
			}

			result := make([]core.MealItem, len(output.Items))
			dishSummaries := make([]string, len(output.Items))
			for i, it := range output.Items {
				ingredients := it.Ingredients
				if len(ingredients) == 0 {
					ingredients = []string{it.Name}
				}
				conf := it.ConfidenceScore
				if conf <= 0 {
					conf = 0.90
				}
				result[i] = core.MealItem{
					ID:                   uuid.New().String(),
					Name:                 it.Name,
					EstimatedWeightGrams: it.EstimatedWeightGrams,
					Calories:             it.Calories,
					ProteinGrams:         it.ProteinGrams,
					CarbsGrams:           it.CarbsGrams,
					FatGrams:             it.FatGrams,
					FiberGrams:           it.FiberGrams,
					SodiumMg:             it.SodiumMg,
					ConfidenceScore:      conf,
					Ingredients:          ingredients,
				}
				dishSummaries[i] = fmt.Sprintf("%s (%.0f kcal, %.1fg protein, %.1fg carbs, %.1fg fat)", it.Name, it.Calories, it.ProteinGrams, it.CarbsGrams, it.FatGrams)
			}

			p.log.Info("Gemini Vision AI: Successfully extracted real meal items",
				zap.String("model", targetModel),
				zap.Int("items_detected", len(result)),
				zap.Strings("dishes", dishSummaries),
			)

			return result, nil
		}
	}

	if lastErr != nil {
		p.log.Error("Gemini Vision AI: All model candidates and retries exhausted. Hard failure.", zap.Error(lastErr), zap.String("notes", userNotes))
		return nil, fmt.Errorf("gemini api call failed after retries: %w", lastErr)
	}
	return nil, fmt.Errorf("gemini api call failed after retries")
}

// AnalyzeMealImage invokes real multimodal vision analysis and strictly disallows demo data.
func (p *VisionAIParser) AnalyzeMealImage(ctx context.Context, imageData []byte, mimeType string, notes ...string) ([]core.MealItem, error) {
	if len(imageData) == 0 {
		return nil, fmt.Errorf("empty image data received")
	}

	userNotes := ""
	if len(notes) > 0 {
		userNotes = notes[0]
	}

	if p.apiKey == "" {
		p.log.Error("Gemini Vision AI: GEMINI_API_KEY environment variable is not configured; demo fallback is strictly disabled")
		return nil, fmt.Errorf("GEMINI_API_KEY is not configured on server; meal analysis cannot proceed and demo data is disabled")
	}

	p.log.Info("Gemini Vision AI: Dispatching actual AI vision analysis from Cadence activity",
		zap.String("model", p.model),
		zap.String("mime_type", mimeType),
		zap.String("notes", userNotes),
		zap.Int("bytes", len(imageData)),
	)

	items, err := p.callGeminiVisionAPI(ctx, imageData, mimeType, userNotes)
	if err != nil {
		p.log.Error("Gemini Vision AI: Analysis failed; demo fallback is strictly disabled",
			zap.Error(err),
			zap.String("notes", userNotes),
		)
		return nil, fmt.Errorf("ai vision analysis failed: %w", err)
	}

	if len(items) == 0 {
		p.log.Error("Gemini Vision AI: 0 items identified in image",
			zap.String("notes", userNotes),
		)
		return nil, fmt.Errorf("unable to detect food items in image: 0 items identified")
	}

	p.log.Info("Actual Gemini AI vision analysis succeeded",
		zap.String("model", p.model),
		zap.Int("items_count", len(items)),
	)
	return items, nil
}
