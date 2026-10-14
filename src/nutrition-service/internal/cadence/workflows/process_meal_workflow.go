package workflows

import (
	"time"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"go.uber.org/cadence/workflow"
)

type ProcessMealWorkflowInput struct {
	JobID     string
	UserUUID  string
	ImageData []byte
	MimeType  string
	MealType  string
	Notes     string
}

type ProcessMealWorkflowResult struct {
	MealLogID     string
	DetectedItems []core.MealItem
}

// ProcessMealImageWorkflow coordinates the async AI vision analysis, raw photo purging,
// and meal persistence across resilient Cadence activities.
func ProcessMealImageWorkflow(ctx workflow.Context, input ProcessMealWorkflowInput) (ProcessMealWorkflowResult, error) {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 2 * time.Minute,
		StartToCloseTimeout:    2 * time.Minute,
		ScheduleToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Update status to PROCESSING
	err := workflow.ExecuteActivity(ctx, "UpdateJobStatusActivity", input.JobID, input.UserUUID, "PROCESSING").Get(ctx, nil)
	if err != nil {
		return ProcessMealWorkflowResult{}, err
	}

	// Step 2: AI Vision OCR and Portion Extraction
	var detectedItems []core.MealItem
	err = workflow.ExecuteActivity(ctx, "AnalyzeMealImageActivity", input.ImageData, input.MimeType).Get(ctx, &detectedItems)
	if err != nil {
		_ = workflow.ExecuteActivity(ctx, "UpdateJobStatusActivity", input.JobID, input.UserUUID, "FAILED").Get(ctx, nil)
		return ProcessMealWorkflowResult{}, err
	}

	// Step 3: Backend Optimization: Purge raw photo bytes from storage/memory
	rawBytes := len(input.ImageData)
	_ = workflow.ExecuteActivity(ctx, "PurgeRawImageActivity", input.JobID, rawBytes).Get(ctx, nil)

	// Step 4: Persist Meal Log
	var mealLogID string
	err = workflow.ExecuteActivity(ctx, "PersistMealLogActivity", input.UserUUID, input.MealType, input.Notes, detectedItems).Get(ctx, &mealLogID)
	if err != nil {
		_ = workflow.ExecuteActivity(ctx, "UpdateJobStatusActivity", input.JobID, input.UserUUID, "FAILED").Get(ctx, nil)
		return ProcessMealWorkflowResult{}, err
	}

	// Step 5: Complete Job Status
	err = workflow.ExecuteActivity(ctx, "CompleteJobActivity", input.JobID, input.UserUUID, mealLogID, detectedItems).Get(ctx, nil)
	if err != nil {
		return ProcessMealWorkflowResult{}, err
	}

	return ProcessMealWorkflowResult{
		MealLogID:     mealLogID,
		DetectedItems: detectedItems,
	}, nil
}
