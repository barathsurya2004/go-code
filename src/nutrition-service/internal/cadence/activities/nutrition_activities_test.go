package activities

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/ai"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/db"
	"go.uber.org/zap"
)

type errorRepo struct {
	*db.MemoryNutritionRepository
	saveJobErr   error
	createLogErr error
}

func (r *errorRepo) SaveJob(ctx context.Context, job *core.JobStatus) error {
	if r.saveJobErr != nil {
		return r.saveJobErr
	}
	return r.MemoryNutritionRepository.SaveJob(ctx, job)
}

func (r *errorRepo) CreateMealLog(ctx context.Context, log *core.MealLog) (*core.MealLog, error) {
	if r.createLogErr != nil {
		return nil, r.createLogErr
	}
	return r.MemoryNutritionRepository.CreateMealLog(ctx, log)
}

func TestUpdateJobStatus(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	memRepo := db.NewMemoryNutritionRepository(logger)
	aiParser := ai.NewVisionAIParser(logger)
	acts := NewNutritionActivities(memRepo, aiParser, logger)

	// Case 1: Job does not exist yet, creates it
	err := acts.UpdateJobStatus(ctx, "job-1", "user-1", "PROCESSING")
	if err != nil {
		t.Fatalf("unexpected error updating non-existent job: %v", err)
	}

	job, err := memRepo.GetJob(ctx, "user-1", "job-1")
	if err != nil || job == nil {
		t.Fatalf("expected job to be retrieved: %v", err)
	}
	if job.Status != "PROCESSING" {
		t.Errorf("expected status PROCESSING, got %s", job.Status)
	}

	// Case 2: Update existing job
	err = acts.UpdateJobStatus(ctx, "job-1", "user-1", "COMPLETED")
	if err != nil {
		t.Fatalf("unexpected error updating existing job: %v", err)
	}
	job, _ = memRepo.GetJob(ctx, "user-1", "job-1")
	if job.Status != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got %s", job.Status)
	}

	// Case 3: Repo error
	errRepo := &errorRepo{
		MemoryNutritionRepository: memRepo,
		saveJobErr:                errors.New("db error"),
	}
	errActs := NewNutritionActivities(errRepo, aiParser, logger)
	err = errActs.UpdateJobStatus(ctx, "job-2", "user-1", "PROCESSING")
	if err == nil {
		t.Error("expected error when repo fails")
	}
}

func TestAnalyzeMealImage(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	memRepo := db.NewMemoryNutritionRepository(logger)
	aiParser := ai.NewVisionAIParser(logger)
	acts := NewNutritionActivities(memRepo, aiParser, logger)

	// Case 1: Empty image data
	_, err := acts.AnalyzeMealImage(ctx, nil, "image/jpeg")
	if err == nil {
		t.Error("expected error for empty image data")
	}

	// Case 2: Valid image data
	items, err := acts.AnalyzeMealImage(ctx, []byte("fake-image-bytes"), "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected error analyzing image: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected detected items, got 0")
	}

	// Case 3: AI parser error
	failingParser := &mockFailingParser{}
	errActs := NewNutritionActivities(memRepo, failingParser, logger)
	_, err = errActs.AnalyzeMealImage(ctx, []byte("fake-bytes"), "image/png")
	if err == nil {
		t.Error("expected error when AI parser fails")
	}
}

type mockFailingParser struct{}

func (m *mockFailingParser) AnalyzeMealImage(ctx context.Context, imageData []byte, mimeType string) ([]core.MealItem, error) {
	return nil, errors.New("ai model timeout")
}

func TestPurgeRawImage(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	acts := NewNutritionActivities(nil, nil, logger)

	err := acts.PurgeRawImage(ctx, "job-purge-1", 102400)
	if err != nil {
		t.Fatalf("unexpected error purging raw image: %v", err)
	}
}

func TestPersistMealLog(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	memRepo := db.NewMemoryNutritionRepository(logger)
	acts := NewNutritionActivities(memRepo, nil, logger)

	sampleItems := []core.MealItem{
		{
			ID:                   "item-1",
			Name:                 "Grilled Salmon",
			EstimatedWeightGrams: 200,
			Calories:             400,
			ProteinGrams:         44,
			CarbsGrams:           0,
			FatGrams:             24,
		},
		{
			ID:                   "", // Should generate UUID
			Name:                 "Steamed Brown Rice",
			EstimatedWeightGrams: 150,
			Calories:             165,
			ProteinGrams:         3.5,
			CarbsGrams:           35,
			FatGrams:             1.5,
		},
	}

	// Case 1: Default mealType when empty
	mealID, err := acts.PersistMealLog(ctx, "user-test", "", "Delicious healthy dinner", sampleItems)
	if err != nil {
		t.Fatalf("unexpected error persisting meal log: %v", err)
	}
	if mealID == "" {
		t.Fatal("expected non-empty meal log ID")
	}

	summary, err := memRepo.GetDailyLog(ctx, "user-test", time.Now().Format("2006-01-02"))
	if err != nil || len(summary.Meals) == 0 {
		t.Fatalf("expected to retrieve saved log in summary: %v", err)
	}
	savedMeal := summary.Meals[len(summary.Meals)-1]
	if savedMeal.MealType != "dinner" {
		t.Errorf("expected default meal type 'dinner', got %s", savedMeal.MealType)
	}
	if savedMeal.TotalCalories != 565 {
		t.Errorf("expected 565 calories, got %f", savedMeal.TotalCalories)
	}
	if savedMeal.TotalProtein != 47.5 {
		t.Errorf("expected 47.5g protein, got %f", savedMeal.TotalProtein)
	}

	// Case 2: Custom mealType
	customMealID, err := acts.PersistMealLog(ctx, "user-test", "breakfast", "Oats", sampleItems)
	if err != nil {
		t.Fatalf("unexpected error with custom mealType: %v", err)
	}
	if customMealID == "" {
		t.Fatal("expected non-empty custom meal ID")
	}

	// Case 3: Repo error
	errRepo := &errorRepo{
		MemoryNutritionRepository: memRepo,
		createLogErr:              errors.New("db write failed"),
	}
	errActs := NewNutritionActivities(errRepo, nil, logger)
	_, err = errActs.PersistMealLog(ctx, "user-test", "lunch", "notes", sampleItems)
	if err == nil {
		t.Error("expected error when repo fails")
	}
}

func TestCompleteJob(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	memRepo := db.NewMemoryNutritionRepository(logger)
	acts := NewNutritionActivities(memRepo, nil, logger)

	items := []core.MealItem{
		{Name: "Greek Yogurt", Calories: 130, ProteinGrams: 15},
	}

	// Case 1: Existing job
	_ = memRepo.SaveJob(ctx, &core.JobStatus{
		JobID:     "job-comp-1",
		UserUUID:  "user-comp",
		Status:    "PROCESSING",
		CreatedAt: time.Now(),
	})

	err := acts.CompleteJob(ctx, "job-comp-1", "user-comp", "meal-log-xyz", items)
	if err != nil {
		t.Fatalf("unexpected error completing job: %v", err)
	}

	job, _ := memRepo.GetJob(ctx, "user-comp", "job-comp-1")
	if job.Status != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", job.Status)
	}
	if job.MealLogID != "meal-log-xyz" {
		t.Errorf("expected mealLogID meal-log-xyz, got %s", job.MealLogID)
	}
	if len(job.DetectedItems) != 1 {
		t.Errorf("expected 1 item, got %d", len(job.DetectedItems))
	}

	// Case 2: Non-existing job
	err = acts.CompleteJob(ctx, "job-comp-new", "user-comp", "meal-log-abc", items)
	if err != nil {
		t.Fatalf("unexpected error completing non-existent job: %v", err)
	}

	// Case 3: Repo error
	errRepo := &errorRepo{
		MemoryNutritionRepository: memRepo,
		saveJobErr:                errors.New("save error"),
	}
	errActs := NewNutritionActivities(errRepo, nil, logger)
	err = errActs.CompleteJob(ctx, "job-err", "user-comp", "meal-log-1", items)
	if err == nil {
		t.Error("expected error when repo save fails")
	}
}
