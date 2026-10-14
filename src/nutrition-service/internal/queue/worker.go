package queue

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/cadence/workflows"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/cadence/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type queuePayload struct {
	jobID     string
	userUUID  string
	imageData []byte
	mimeType  string
	mealType  string
	notes     string
}

type WorkerParams struct {
	fx.In
	Repo          core.NutritionRepository
	AIParser      core.AIParser
	CadenceClient client.Client `optional:"true"`
	Log           *zap.Logger
}

func NewFXAsyncImageQueueWorker(p WorkerParams) *AsyncImageQueueWorker {
	return NewAsyncImageQueueWorkerWithCadence(p.Repo, p.AIParser, p.CadenceClient, p.Log)
}

type AsyncImageQueueWorker struct {
	repo            core.NutritionRepository
	aiParser        core.AIParser
	cadenceClient   client.Client
	queueChan       chan *queuePayload
	stopChan        chan struct{}
	wg              sync.WaitGroup
	log             *zap.Logger
	processingDelay time.Duration
}

func NewAsyncImageQueueWorker(
	repo core.NutritionRepository,
	aiParser core.AIParser,
	log *zap.Logger,
) *AsyncImageQueueWorker {
	return NewAsyncImageQueueWorkerWithCadence(repo, aiParser, nil, log)
}

func NewAsyncImageQueueWorkerWithCadence(
	repo core.NutritionRepository,
	aiParser core.AIParser,
	cadenceClient client.Client,
	log *zap.Logger,
) *AsyncImageQueueWorker {
	delay := 2500 * time.Millisecond
	if envDelay := os.Getenv("AI_PROCESSING_DELAY_MS"); envDelay != "" {
		if ms, err := strconv.Atoi(envDelay); err == nil {
			delay = time.Duration(ms) * time.Millisecond
		}
	}

	return &AsyncImageQueueWorker{
		repo:            repo,
		aiParser:        aiParser,
		cadenceClient:   cadenceClient,
		queueChan:       make(chan *queuePayload, 100),
		stopChan:        make(chan struct{}),
		log:             log,
		processingDelay: delay,
	}
}

// SetCadenceClient attaches a Cadence workflow client.
func (w *AsyncImageQueueWorker) SetCadenceClient(c client.Client) {
	w.cadenceClient = c
}

// SetProcessingDelay overrides the simulated delay (useful for tests or custom environments).
func (w *AsyncImageQueueWorker) SetProcessingDelay(d time.Duration) {
	w.processingDelay = d
}

func (w *AsyncImageQueueWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.workerLoop()
	w.log.Info("Async image queue worker started")
}

func (w *AsyncImageQueueWorker) Stop() {
	close(w.stopChan)
	w.wg.Wait()
	w.log.Info("Async image queue worker stopped")
}

func (w *AsyncImageQueueWorker) Enqueue(
	ctx context.Context,
	jobID, userUUID string,
	imageData []byte,
	mimeType, mealType, notes string,
) error {
	job := &core.JobStatus{
		JobID:        jobID,
		UserUUID:     userUUID,
		Status:       "PENDING",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := w.repo.SaveJob(ctx, job); err != nil {
		return fmt.Errorf("failed to save initial job: %w", err)
	}

	// If Cadence workflow client is available, orchestrate asynchronously on nouri-lifecycle-workers domain
	if w.cadenceClient != nil {
		workflowOptions := client.StartWorkflowOptions{
			ID:                           fmt.Sprintf("nouri-meal-%s", jobID),
			TaskList:                     "nouri-tasklist",
			ExecutionStartToCloseTimeout: 10 * time.Minute,
		}
		_, err := w.cadenceClient.StartWorkflow(ctx, workflowOptions, "ProcessMealImageWorkflow", workflows.ProcessMealWorkflowInput{
			JobID:     jobID,
			UserUUID:  userUUID,
			ImageData: imageData,
			MimeType:  mimeType,
			MealType:  mealType,
			Notes:     notes,
		})
		if err == nil {
			w.log.Info("Dispatched meal processing to Cadence workflow",
				zap.String("domain", "nouri-lifecycle-workers"),
				zap.String("task_list", "nouri-tasklist"),
				zap.String("job_id", jobID),
			)
			return nil
		}
		w.log.Warn("Cadence workflow dispatch failed, falling back to local queue", zap.Error(err))
	}

	payload := &queuePayload{
		jobID:     jobID,
		userUUID:  userUUID,
		imageData: imageData,
		mimeType:  mimeType,
		mealType:  mealType,
		notes:     notes,
	}

	select {
	case w.queueChan <- payload:
		w.log.Info("Enqueued image processing job",
			zap.String("job_id", jobID),
			zap.String("user_uuid", userUUID),
			zap.Int("bytes", len(imageData)),
		)
		return nil
	default:
		return fmt.Errorf("image processing queue is full")
	}
}

func (w *AsyncImageQueueWorker) workerLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopChan:
			return
		case payload, ok := <-w.queueChan:
			if !ok {
				return
			}
			w.processPayload(payload)
		}
	}
}

func (w *AsyncImageQueueWorker) processPayload(p *queuePayload) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Update status to PROCESSING
	job, err := w.repo.GetJob(ctx, p.userUUID, p.jobID)
	if err == nil && job != nil {
		job.Status = "PROCESSING"
		job.UpdatedAt = time.Now()
		_ = w.repo.SaveJob(ctx, job)
	}

	w.log.Info("Processing image in queue",
		zap.String("job_id", p.jobID),
		zap.String("user_uuid", p.userUUID),
	)

	// Simulate realistic AI parsing and OCR pipeline latency without holding open client HTTP connection
	if w.processingDelay > 0 {
		select {
		case <-time.After(w.processingDelay):
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		}
	}

	// Call AI Vision to extract meal items
	items, err := w.aiParser.AnalyzeMealImage(ctx, p.imageData, p.mimeType, p.notes)

	// BACKEND OPTIMIZATION:
	// As requested, once AI extraction is complete, immediately wipe and purge the raw photo
	// from memory/queue storage to optimize server memory & disk footprint!
	rawBytesCount := len(p.imageData)
	p.imageData = nil

	w.log.Info("Backend optimization: purged raw image from queue storage",
		zap.String("job_id", p.jobID),
		zap.Int("freed_bytes", rawBytesCount),
	)

	if err != nil {
		w.log.Error("Failed to extract nutrition from image",
			zap.String("job_id", p.jobID),
			zap.Error(err),
		)
		if job != nil {
			job.Status = "FAILED"
			job.ErrorMessage = err.Error()
			job.UpdatedAt = time.Now()
			_ = w.repo.SaveJob(ctx, job)
		}
		return
	}

	// Create Meal Log
	mealType := p.mealType
	if mealType == "" {
		mealType = determineMealType(time.Now())
	}

	mealLog := &core.MealLog{
		ID:       uuid.New().String(),
		UserUUID: p.userUUID,
		MealType: mealType,
		LoggedAt: time.Now(),
		Notes:    p.notes,
		Items:    items,
	}

	savedLog, err := w.repo.CreateMealLog(ctx, mealLog)
	if err != nil {
		w.log.Error("Failed to persist parsed meal log",
			zap.String("job_id", p.jobID),
			zap.Error(err),
		)
		if job != nil {
			job.Status = "FAILED"
			job.ErrorMessage = err.Error()
			job.UpdatedAt = time.Now()
			_ = w.repo.SaveJob(ctx, job)
		}
		return
	}

	// Update Job to COMPLETED
	if job != nil {
		job.Status = "COMPLETED"
		job.MealLogID = savedLog.ID
		job.DetectedItems = savedLog.Items
		job.UpdatedAt = time.Now()
		_ = w.repo.SaveJob(ctx, job)
	}

	w.log.Info("Successfully processed and logged meal from photo",
		zap.String("job_id", p.jobID),
		zap.String("meal_log_id", savedLog.ID),
		zap.Float64("total_calories", savedLog.TotalCalories),
		zap.Float64("total_protein", savedLog.TotalProtein),
	)
}

func determineMealType(t time.Time) string {
	hour := t.Hour()
	switch {
	case hour >= 5 && hour < 11:
		return "breakfast"
	case hour >= 11 && hour < 16:
		return "lunch"
	case hour >= 16 && hour < 22:
		return "dinner"
	default:
		return "snack"
	}
}
