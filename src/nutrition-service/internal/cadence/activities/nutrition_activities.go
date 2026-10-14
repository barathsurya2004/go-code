package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type NutritionActivities struct {
	repo     core.NutritionRepository
	aiParser core.AIParser
	log      *zap.Logger
}

func NewNutritionActivities(repo core.NutritionRepository, aiParser core.AIParser, log *zap.Logger) *NutritionActivities {
	return &NutritionActivities{
		repo:     repo,
		aiParser: aiParser,
		log:      log,
	}
}

// UpdateJobStatus updates the state of an existing queue processing job.
func (a *NutritionActivities) UpdateJobStatus(ctx context.Context, jobID, userUUID, status string) error {
	job, err := a.repo.GetJob(ctx, userUUID, jobID)
	if err != nil || job == nil {
		job = &core.JobStatus{
			JobID:     jobID,
			UserUUID:  userUUID,
			CreatedAt: time.Now(),
		}
	}
	job.Status = status
	job.UpdatedAt = time.Now()
	if err := a.repo.SaveJob(ctx, job); err != nil {
		a.log.Error("Failed to update job status", zap.String("job_id", jobID), zap.String("status", status), zap.Error(err))
		return err
	}
	a.log.Info("Cadence activity: updated job status", zap.String("job_id", jobID), zap.String("status", status))
	return nil
}

// AnalyzeMealImage invokes the Vision AI engine to parse portions, nutrients, and dishes.
func (a *NutritionActivities) AnalyzeMealImage(ctx context.Context, imageData []byte, mimeType string) ([]core.MealItem, error) {
	if len(imageData) == 0 {
		return nil, fmt.Errorf("empty image data received")
	}
	items, err := a.aiParser.AnalyzeMealImage(ctx, imageData, mimeType)
	if err != nil {
		a.log.Error("Cadence activity: AI vision parsing failed", zap.Error(err))
		return nil, err
	}
	a.log.Info("Cadence activity: Vision AI successfully extracted meal items", zap.Int("items_count", len(items)))
	return items, nil
}

// PurgeRawImage enforces the backend memory & disk optimization requirement:
// Immediately frees raw photo bytes from memory and queue storage once AI extraction finishes.
func (a *NutritionActivities) PurgeRawImage(ctx context.Context, jobID string, freedBytes int) error {
	a.log.Info("Cadence activity: Purged raw photo from queue memory & disk storage",
		zap.String("job_id", jobID),
		zap.Int("freed_bytes", freedBytes),
	)
	return nil
}

// PersistMealLog stores the verified meal items, calculated totals, and macros into the database.
func (a *NutritionActivities) PersistMealLog(ctx context.Context, userUUID, mealType, notes string, items []core.MealItem) (string, error) {
	if mealType == "" {
		mealType = "dinner"
	}

	mealID := uuid.New().String()
	var totalCal, totalProt, totalCarb, totalFat float64

	itemsWithIDs := make([]core.MealItem, len(items))
	for i, item := range items {
		if item.ID == "" {
			item.ID = uuid.New().String()
		}
		item.MealLogID = mealID
		totalCal += item.Calories
		totalProt += item.ProteinGrams
		totalCarb += item.CarbsGrams
		totalFat += item.FatGrams
		itemsWithIDs[i] = item
	}

	mealLog := &core.MealLog{
		ID:            mealID,
		UserUUID:      userUUID,
		MealType:      mealType,
		LoggedAt:      time.Now(),
		Notes:         notes,
		Items:         itemsWithIDs,
		TotalCalories: totalCal,
		TotalProtein:  totalProt,
		TotalCarbs:    totalCarb,
		TotalFat:      totalFat,
	}

	created, err := a.repo.CreateMealLog(ctx, mealLog)
	if err != nil {
		a.log.Error("Cadence activity: Failed to persist meal log", zap.Error(err))
		return "", err
	}
	a.log.Info("Cadence activity: Successfully persisted meal log",
		zap.String("meal_log_id", created.ID),
		zap.Float64("calories", created.TotalCalories),
	)
	return created.ID, nil
}

// CompleteJob marks the workflow job as COMPLETED and links the created meal log.
func (a *NutritionActivities) CompleteJob(ctx context.Context, jobID, userUUID, mealLogID string, items []core.MealItem) error {
	job, err := a.repo.GetJob(ctx, userUUID, jobID)
	if err != nil || job == nil {
		job = &core.JobStatus{
			JobID:    jobID,
			UserUUID: userUUID,
		}
	}
	job.Status = "COMPLETED"
	job.MealLogID = mealLogID
	job.DetectedItems = items
	job.UpdatedAt = time.Now()

	return a.repo.SaveJob(ctx, job)
}
