package grpcserver

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/ai"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/db"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/queue"
	nutritionv1 "github.com/barathsurya2004/go-code/pkg/proto/nutritionv1"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

func TestNutritionGRPCServer(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	parser := ai.NewVisionAIParser(logger)
	worker := queue.NewAsyncImageQueueWorker(repo, parser, logger)
	server := NewNutritionGRPCServer(repo, worker, logger)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-uuid", "user-test-789"))

	// 1. CheckHealth
	health, err := server.CheckHealth(ctx, &nutritionv1.HealthCheckRequest{})
	if err != nil || health.Status != "SERVING" {
		t.Fatalf("health check failed: %v", err)
	}

	// Test fallback uuid
	emptyCtx := context.Background()
	u1 := extractUserUUID(emptyCtx, "fallback-user-1")
	if u1 != "fallback-user-1" {
		t.Errorf("expected fallback user, got %s", u1)
	}
	uDefault := extractUserUUID(emptyCtx, "")
	if uDefault == "" {
		t.Errorf("expected default uuid, got empty")
	}

	// 2. Goals
	goals, err := server.GetGoals(ctx, &nutritionv1.GetGoalsRequest{})
	if err != nil {
		t.Fatalf("failed to get goals: %v", err)
	}
	if goals.CaloriesTarget != 2200 {
		t.Errorf("expected default 2200 calories, got %v", goals.CaloriesTarget)
	}

	updatedGoals, err := server.SetGoals(ctx, &nutritionv1.SetGoalsRequest{
		Goals: &nutritionv1.NutritionGoals{
			CaloriesTarget: 2400,
			ProteinGrams:   170,
			CarbsGrams:     230,
			FatGrams:       65,
			FiberGrams:     35,
			SodiumMg:       2200,
		},
	})
	if err != nil || updatedGoals.CaloriesTarget != 2400 {
		t.Fatalf("failed to update goals: %v", err)
	}

	// Invalid SetGoals
	if _, err := server.SetGoals(ctx, &nutritionv1.SetGoalsRequest{}); err == nil {
		t.Error("expected error for empty SetGoalsRequest")
	}

	// 3. AddManualMeal
	manualMeal, err := server.AddManualMeal(ctx, &nutritionv1.AddManualMealRequest{
		MealType: "breakfast",
		LoggedAt: time.Now().Format(time.RFC3339),
		Notes:    "Morning oatmeal and eggs",
		Items: []*nutritionv1.MealItem{
			{
				Name:                 "Steel Cut Oats with Berries",
				EstimatedWeightGrams: 200,
				Calories:             320,
				ProteinGrams:         12,
				CarbsGrams:           58,
				FatGrams:             6,
				FiberGrams:           8,
				SodiumMg:             120,
				Ingredients:          []string{"Oats", "Blueberries", "Almond milk"},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to add manual meal: %v", err)
	}
	if manualMeal.TotalCalories != 320 {
		t.Errorf("expected 320 total calories, got %v", manualMeal.TotalCalories)
	}

	// 4. GetDailyLog
	today := time.Now().Format("2006-01-02")
	daily, err := server.GetDailyLog(ctx, &nutritionv1.GetDailyLogRequest{Date: today})
	if err != nil {
		t.Fatalf("failed to get daily log: %v", err)
	}
	if daily.TotalCalories != 320 || len(daily.Meals) != 1 {
		t.Errorf("expected 1 meal with 320 cal, got %+v", daily)
	}

	// 5. UpdateMealItem
	itemToUpdate := manualMeal.Items[0]
	updatedItem, err := server.UpdateMealItem(ctx, &nutritionv1.UpdateMealItemRequest{
		ItemId:               itemToUpdate.Id,
		Name:                 "Oats with Scoop of Protein",
		EstimatedWeightGrams: 250,
		Calories:             420,
		ProteinGrams:         35,
		CarbsGrams:           60,
		FatGrams:             7,
		FiberGrams:           9,
	})
	if err != nil {
		t.Fatalf("failed to update meal item: %v", err)
	}
	if updatedItem.Calories != 420 {
		t.Errorf("expected 420 cal, got %v", updatedItem.Calories)
	}

	// Invalid update
	if _, err := server.UpdateMealItem(ctx, &nutritionv1.UpdateMealItemRequest{ItemId: "fake"}); err == nil {
		t.Error("expected error updating fake meal item")
	}

	// 6. ProcessImage
	// Empty image error
	if _, err := server.ProcessImage(ctx, &nutritionv1.ProcessImageRequest{}); err == nil {
		t.Error("expected error for empty image data")
	}

	// Valid image
	jobResp, err := server.ProcessImage(ctx, &nutritionv1.ProcessImageRequest{
		ImageData: []byte("fake-image-bytes"),
		MimeType:  "image/jpeg",
		MealType:  "dinner",
		Notes:     "Pasted from mobile clipboard",
	})
	if err != nil {
		t.Fatalf("failed to process image: %v", err)
	}
	if jobResp.JobId == "" || jobResp.Status != "PENDING" {
		t.Errorf("invalid job response: %+v", jobResp)
	}

	// 7. GetJobStatus
	jobStatus, err := server.GetJobStatus(ctx, &nutritionv1.GetJobStatusRequest{JobId: jobResp.JobId})
	if err != nil {
		t.Fatalf("failed to get job status: %v", err)
	}
	if jobStatus.JobId != jobResp.JobId {
		t.Errorf("expected job id %s, got %s", jobResp.JobId, jobStatus.JobId)
	}

	// Invalid job status
	if _, err := server.GetJobStatus(ctx, &nutritionv1.GetJobStatusRequest{JobId: "nonexistent"}); err == nil {
		t.Error("expected error for nonexistent job")
	}

	// 8. DeleteMealItem & DeleteMealLog
	delItemResp, err := server.DeleteMealItem(ctx, &nutritionv1.DeleteMealItemRequest{ItemId: updatedItem.Id})
	if err != nil || !delItemResp.Success {
		t.Fatalf("failed to delete meal item: %v", err)
	}
	if _, err := server.DeleteMealItem(ctx, &nutritionv1.DeleteMealItemRequest{ItemId: "fake"}); err == nil {
		t.Error("expected error deleting fake item")
	}

	delMealResp, err := server.DeleteMealLog(ctx, &nutritionv1.DeleteMealLogRequest{MealLogId: manualMeal.Id})
	// Note: since the only item was deleted above, the meal might already have been cleaned up
	if err != nil {
		// Test deleting a fresh meal
		fresh, _ := server.AddManualMeal(ctx, &nutritionv1.AddManualMealRequest{
			MealType: "snack",
			Items: []*nutritionv1.MealItem{
				{Name: "Apple", Calories: 80, ProteinGrams: 0.5},
			},
		})
		delFresh, err2 := server.DeleteMealLog(ctx, &nutritionv1.DeleteMealLogRequest{MealLogId: fresh.Id})
		if err2 != nil || !delFresh.Success {
			t.Fatalf("failed to delete fresh meal: %v", err2)
		}
	} else if !delMealResp.Success {
		t.Error("expected success deleting meal log")
	}

	if _, err := server.DeleteMealLog(ctx, &nutritionv1.DeleteMealLogRequest{MealLogId: "fake"}); err == nil {
		t.Error("expected error deleting fake meal")
	}
}

func TestStartGRPCServerLifecycle(t *testing.T) {
	os.Setenv("GRPC_PORT", "50099")
	defer os.Unsetenv("GRPC_PORT")

	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	parser := ai.NewVisionAIParser(logger)
	worker := queue.NewAsyncImageQueueWorker(repo, parser, logger)
	server := NewNutritionGRPCServer(repo, worker, logger)

	lc := fxtest.NewLifecycle(t)
	grpcServer, err := StartGRPCServer(server, worker, lc, logger)
	if err != nil {
		t.Fatalf("failed to start grpc server: %v", err)
	}
	if grpcServer == nil {
		t.Fatal("expected grpc server instance")
	}

	lc.RequireStart()
	lc.RequireStop()
}

type errQueueWorker struct {
	queue.AsyncImageQueueWorker
}

func (e *errQueueWorker) Enqueue(ctx context.Context, jobID, userUUID string, imageData []byte, mimeType, mealType, notes string) error {
	return os.ErrInvalid
}

func TestNutritionGRPCServer_EdgeCases(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	worker := queue.NewAsyncImageQueueWorker(repo, ai.NewVisionAIParser(logger), logger)
	server := NewNutritionGRPCServer(repo, worker, logger)
	ctx := context.Background()

	// 1. SetGoals with nil goals
	if _, err := server.SetGoals(ctx, &nutritionv1.SetGoalsRequest{Goals: nil}); err == nil {
		t.Error("expected error when SetGoals payload is nil")
	}

	// 2. ProcessImage with empty image
	if _, err := server.ProcessImage(ctx, &nutritionv1.ProcessImageRequest{ImageData: nil}); err == nil {
		t.Error("expected error when image data is empty")
	}

	// 3. ProcessImage with worker error
	errWorkerServer := NewNutritionGRPCServer(repo, &errQueueWorker{}, logger)
	if _, err := errWorkerServer.ProcessImage(ctx, &nutritionv1.ProcessImageRequest{ImageData: []byte("abc")}); err == nil {
		t.Error("expected error when worker fails to enqueue")
	}

	// 4. AddManualMeal with custom LoggedAt
	manualResp, err := server.AddManualMeal(ctx, &nutritionv1.AddManualMealRequest{
		MealType: "breakfast",
		LoggedAt: time.Now().Format(time.RFC3339),
		Items: []*nutritionv1.MealItem{
			{Name: "Bagel", Calories: 250, ProteinGrams: 9},
		},
	})
	if err != nil || manualResp == nil {
		t.Fatalf("unexpected error adding manual meal: %v", err)
	}

	// 5. GetJobStatus with populated items
	job := &core.JobStatus{
		JobID:    "job-pop",
		UserUUID: "default-user",
		Status:   "COMPLETED",
		DetectedItems: []core.MealItem{
			{ID: "i1", Name: "Soup", Calories: 150},
		},
	}
	_ = repo.SaveJob(ctx, job)
	jStatus, err := server.GetJobStatus(ctx, &nutritionv1.GetJobStatusRequest{UserUuid: "default-user", JobId: "job-pop"})
	if err != nil || jStatus.Status != "COMPLETED" || len(jStatus.DetectedItems) != 1 {
		t.Errorf("unexpected job status response: %v, %v", jStatus, err)
	}

	// 6. Repo errors for GetGoals, SetGoals, GetDailyLog
	errRepoServer := NewNutritionGRPCServer(&mockErrRepo{}, worker, logger)
	if _, err := errRepoServer.GetGoals(ctx, &nutritionv1.GetGoalsRequest{}); err == nil {
		t.Error("expected error from GetGoals when repo fails")
	}
	if _, err := errRepoServer.SetGoals(ctx, &nutritionv1.SetGoalsRequest{Goals: &nutritionv1.NutritionGoals{}}); err == nil {
		t.Error("expected error from SetGoals when repo fails")
	}
	if _, err := errRepoServer.GetDailyLog(ctx, &nutritionv1.GetDailyLogRequest{}); err == nil {
		t.Error("expected error from GetDailyLog when repo fails")
	}
}

type mockErrRepo struct {
	core.NutritionRepository
}

func (m *mockErrRepo) GetGoals(ctx context.Context, userUUID string) (*core.NutritionGoals, error) {
	return nil, os.ErrInvalid
}

func (m *mockErrRepo) SetGoals(ctx context.Context, goals *core.NutritionGoals) (*core.NutritionGoals, error) {
	return nil, os.ErrInvalid
}

func (m *mockErrRepo) GetDailyLog(ctx context.Context, userUUID, date string) (*core.DailySummary, error) {
	return nil, os.ErrInvalid
}


