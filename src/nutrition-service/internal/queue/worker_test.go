package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/ai"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/db"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

type failingParser struct{}

func (f *failingParser) AnalyzeMealImage(ctx context.Context, imageData []byte, mimeType string) ([]core.MealItem, error) {
	return nil, fmt.Errorf("simulated AI parsing error")
}

type errorRepo struct {
	core.NutritionRepository
	failSaveJob   bool
	failCreateLog bool
}

func (e *errorRepo) SaveJob(ctx context.Context, job *core.JobStatus) error {
	if e.failSaveJob {
		return fmt.Errorf("db save job error")
	}
	return e.NutritionRepository.SaveJob(ctx, job)
}

func (e *errorRepo) CreateMealLog(ctx context.Context, log *core.MealLog) (*core.MealLog, error) {
	if e.failCreateLog {
		return nil, fmt.Errorf("db create meal log error")
	}
	return e.NutritionRepository.CreateMealLog(ctx, log)
}

func TestAsyncImageQueueWorker(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	aiParser := ai.NewVisionAIParser(logger)
	worker := NewAsyncImageQueueWorker(repo, aiParser, logger)
	worker.SetProcessingDelay(10 * time.Millisecond)

	ctx := context.Background()
	worker.Start(ctx)
	defer worker.Stop()

	// Enqueue valid job
	userUUID := "user-456"
	jobID := "job-123"
	dummyImage := []byte("fake-photo-binary-data-for-testing")

	err := worker.Enqueue(ctx, jobID, userUUID, dummyImage, "image/jpeg", "lunch", "Test note")
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// Wait for processing to complete
	var finalJob *core.JobStatus
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		job, err := repo.GetJob(ctx, userUUID, jobID)
		if err == nil && job.Status == "COMPLETED" {
			finalJob = job
			break
		}
	}

	if finalJob == nil {
		t.Fatal("expected job to reach COMPLETED status within timeout")
	}
	if finalJob.MealLogID == "" {
		t.Error("expected valid meal log ID in completed job")
	}
	if len(finalJob.DetectedItems) == 0 {
		t.Error("expected detected items in completed job")
	}

	// Verify meal was persisted in repo
	daily, err := repo.GetDailyLog(ctx, userUUID, time.Now().Format("2006-01-02"))
	if err != nil || len(daily.Meals) == 0 {
		t.Fatalf("expected logged meal in daily log: %v", err)
	}
	if daily.TotalCalories <= 0 {
		t.Errorf("expected positive calories, got %v", daily.TotalCalories)
	}
}

func TestWorkerFailureBranch(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	failWorker := NewAsyncImageQueueWorker(repo, &failingParser{}, logger)
	failWorker.SetProcessingDelay(10 * time.Millisecond)

	ctx := context.Background()
	failWorker.Start(ctx)
	defer failWorker.Stop()

	userUUID := "user-fail"
	jobID := "job-fail-1"
	dummyImage := []byte("fake-photo-fail")

	err := failWorker.Enqueue(ctx, jobID, userUUID, dummyImage, "image/jpeg", "", "")
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	// Wait for failure status
	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		job, err := repo.GetJob(ctx, userUUID, jobID)
		if err == nil && job.Status == "FAILED" {
			if job.ErrorMessage == "" {
				t.Error("expected error message in failed job")
			}
			return
		}
	}
	t.Fatal("expected job to reach FAILED status")
}

func TestWorkerInitialSaveFailure(t *testing.T) {
	logger := zap.NewNop()
	baseRepo := db.NewMemoryNutritionRepository(logger)
	errRepo := &errorRepo{NutritionRepository: baseRepo, failSaveJob: true}
	worker := NewAsyncImageQueueWorker(errRepo, ai.NewVisionAIParser(logger), logger)

	err := worker.Enqueue(context.Background(), "j1", "u1", []byte("img"), "image/png", "", "")
	if err == nil {
		t.Error("expected error when initial save job fails")
	}
}

func TestWorkerLogCreationFailure(t *testing.T) {
	logger := zap.NewNop()
	baseRepo := db.NewMemoryNutritionRepository(logger)
	errRepo := &errorRepo{NutritionRepository: baseRepo, failCreateLog: true}
	worker := NewAsyncImageQueueWorker(errRepo, ai.NewVisionAIParser(logger), logger)
	worker.SetProcessingDelay(10 * time.Millisecond)

	ctx := context.Background()
	worker.Start(ctx)
	defer worker.Stop()

	err := worker.Enqueue(ctx, "j-log-fail", "u-log-fail", []byte("img"), "image/png", "", "")
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	for i := 0; i < 20; i++ {
		time.Sleep(20 * time.Millisecond)
		job, err := errRepo.GetJob(ctx, "u-log-fail", "j-log-fail")
		if err == nil && job.Status == "FAILED" {
			return
		}
	}
	t.Fatal("expected job to reach FAILED status on log creation error")
}

func TestWorkerQueueFull(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	worker := NewAsyncImageQueueWorker(repo, ai.NewVisionAIParser(logger), logger)

	// Fill channel capacity (100)
	for i := 0; i < 100; i++ {
		_ = worker.Enqueue(context.Background(), fmt.Sprintf("j-%d", i), "u1", []byte("img"), "image/png", "", "")
	}

	// 101st should fail
	err := worker.Enqueue(context.Background(), "j-overflow", "u1", []byte("img"), "image/png", "", "")
	if err == nil {
		t.Error("expected error when queue is full")
	}
}

func TestDetermineMealType(t *testing.T) {
	if determineMealType(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)) != "breakfast" {
		t.Error("expected breakfast")
	}
	if determineMealType(time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC)) != "lunch" {
		t.Error("expected lunch")
	}
	if determineMealType(time.Date(2026, 1, 1, 19, 0, 0, 0, time.UTC)) != "dinner" {
		t.Error("expected dinner")
	}
	if determineMealType(time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)) != "snack" {
		t.Error("expected snack")
	}
}

func TestWorkerEnvDelay(t *testing.T) {
	t.Setenv("AI_PROCESSING_DELAY_MS", "500")
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	w := NewAsyncImageQueueWorker(repo, ai.NewVisionAIParser(logger), logger)
	if w.processingDelay != 500*time.Millisecond {
		t.Errorf("expected 500ms delay, got %v", w.processingDelay)
	}
}

func TestWorkerStopDuringDelay(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	worker := NewAsyncImageQueueWorker(repo, ai.NewVisionAIParser(logger), logger)
	worker.SetProcessingDelay(1 * time.Second)

	ctx := context.Background()
	worker.Start(ctx)

	_ = worker.Enqueue(ctx, "job-stop", "user-stop", []byte("img"), "image/jpeg", "", "")
	time.Sleep(20 * time.Millisecond)
	worker.Stop()
}

type mockCadenceClient struct {
	client.Client
	startErr error
	started  bool
}

func (m *mockCadenceClient) StartWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflowFunc interface{}, args ...interface{}) (*workflow.Execution, error) {
	if m.startErr != nil {
		return nil, m.startErr
	}
	m.started = true
	return &workflow.Execution{ID: options.ID, RunID: "run-mock"}, nil
}

func TestEnqueueWithCadenceClient(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	aiParser := ai.NewVisionAIParser(logger)
	mockCad := &mockCadenceClient{}

	w := NewFXAsyncImageQueueWorker(WorkerParams{
		Repo:          repo,
		AIParser:      aiParser,
		CadenceClient: mockCad,
		Log:           logger,
	})

	ctx := context.Background()
	err := w.Enqueue(ctx, "job-cadence-ok", "user-cadence", []byte("test-bytes"), "image/png", "dinner", "notes")
	if err != nil {
		t.Fatalf("unexpected error enqueueing to Cadence: %v", err)
	}
	if !mockCad.started {
		t.Error("expected Cadence StartWorkflow to be called")
	}

	// Test fallback when Cadence StartWorkflow fails
	mockCadFail := &mockCadenceClient{startErr: fmt.Errorf("cadence unreachable")}
	wFallback := NewAsyncImageQueueWorkerWithCadence(repo, aiParser, mockCadFail, logger)
	wFallback.SetProcessingDelay(10 * time.Millisecond)
	wFallback.Start(ctx)
	defer wFallback.Stop()

	err = wFallback.Enqueue(ctx, "job-cadence-fallback", "user-cadence", []byte("bytes"), "image/jpeg", "lunch", "notes")
	if err != nil {
		t.Fatalf("unexpected error during Cadence fallback: %v", err)
	}
}

