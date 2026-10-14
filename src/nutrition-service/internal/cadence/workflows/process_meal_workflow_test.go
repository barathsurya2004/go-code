package workflows

import (
	"context"
	"errors"
	"testing"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
)

type WorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}

func registerActivitiesOnEnv(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(func(ctx context.Context, jobID, userUUID, status string) error {
		return nil
	}, activity.RegisterOptions{Name: "UpdateJobStatusActivity"})

	env.RegisterActivityWithOptions(func(ctx context.Context, imageData []byte, mimeType string) ([]core.MealItem, error) {
		return nil, nil
	}, activity.RegisterOptions{Name: "AnalyzeMealImageActivity"})

	env.RegisterActivityWithOptions(func(ctx context.Context, jobID string, freedBytes int) error {
		return nil
	}, activity.RegisterOptions{Name: "PurgeRawImageActivity"})

	env.RegisterActivityWithOptions(func(ctx context.Context, userUUID, mealType, notes string, items []core.MealItem) (string, error) {
		return "", nil
	}, activity.RegisterOptions{Name: "PersistMealLogActivity"})

	env.RegisterActivityWithOptions(func(ctx context.Context, jobID, userUUID, mealLogID string, items []core.MealItem) error {
		return nil
	}, activity.RegisterOptions{Name: "CompleteJobActivity"})
}

func (s *WorkflowTestSuite) Test_ProcessMealImageWorkflow_Success() {
	env := s.NewTestWorkflowEnvironment()
	registerActivitiesOnEnv(env)

	input := ProcessMealWorkflowInput{
		JobID:     "job-100",
		UserUUID:  "user-100",
		ImageData: []byte("sample-image"),
		MimeType:  "image/jpeg",
		MealType:  "dinner",
		Notes:     "Tofu bowl with avocado",
	}

	detected := []core.MealItem{
		{
			ID:                   "item-1",
			Name:                 "Organic Tofu",
			EstimatedWeightGrams: 150,
			Calories:             180,
			ProteinGrams:         18,
		},
	}

	env.OnActivity("UpdateJobStatusActivity", mock.Anything, input.JobID, input.UserUUID, "PROCESSING").Return(nil)
	env.OnActivity("AnalyzeMealImageActivity", mock.Anything, input.ImageData, input.MimeType).Return(detected, nil)
	env.OnActivity("PurgeRawImageActivity", mock.Anything, input.JobID, len(input.ImageData)).Return(nil)
	env.OnActivity("PersistMealLogActivity", mock.Anything, input.UserUUID, input.MealType, input.Notes, detected).Return("meal-log-success", nil)
	env.OnActivity("CompleteJobActivity", mock.Anything, input.JobID, input.UserUUID, "meal-log-success", detected).Return(nil)

	env.ExecuteWorkflow(ProcessMealImageWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result ProcessMealWorkflowResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal("meal-log-success", result.MealLogID)
	s.Equal(1, len(result.DetectedItems))
	s.Equal("Organic Tofu", result.DetectedItems[0].Name)
}

func (s *WorkflowTestSuite) Test_ProcessMealImageWorkflow_InitialStatusError() {
	env := s.NewTestWorkflowEnvironment()
	registerActivitiesOnEnv(env)

	input := ProcessMealWorkflowInput{
		JobID:    "job-err-1",
		UserUUID: "user-err-1",
	}

	env.OnActivity("UpdateJobStatusActivity", mock.Anything, input.JobID, input.UserUUID, "PROCESSING").Return(errors.New("db down"))

	env.ExecuteWorkflow(ProcessMealImageWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *WorkflowTestSuite) Test_ProcessMealImageWorkflow_AIParseError() {
	env := s.NewTestWorkflowEnvironment()
	registerActivitiesOnEnv(env)

	input := ProcessMealWorkflowInput{
		JobID:     "job-ai-fail",
		UserUUID:  "user-ai-fail",
		ImageData: []byte("corrupt-bytes"),
		MimeType:  "image/png",
	}

	env.OnActivity("UpdateJobStatusActivity", mock.Anything, input.JobID, input.UserUUID, "PROCESSING").Return(nil)
	env.OnActivity("AnalyzeMealImageActivity", mock.Anything, input.ImageData, input.MimeType).Return(nil, errors.New("ai error"))
	env.OnActivity("UpdateJobStatusActivity", mock.Anything, input.JobID, input.UserUUID, "FAILED").Return(nil)

	env.ExecuteWorkflow(ProcessMealImageWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *WorkflowTestSuite) Test_ProcessMealImageWorkflow_PersistError() {
	env := s.NewTestWorkflowEnvironment()
	registerActivitiesOnEnv(env)

	input := ProcessMealWorkflowInput{
		JobID:     "job-persist-fail",
		UserUUID:  "user-persist-fail",
		ImageData: []byte("image-data"),
		MimeType:  "image/jpeg",
	}

	detected := []core.MealItem{{Name: "Eggs", Calories: 140}}

	env.OnActivity("UpdateJobStatusActivity", mock.Anything, input.JobID, input.UserUUID, "PROCESSING").Return(nil)
	env.OnActivity("AnalyzeMealImageActivity", mock.Anything, input.ImageData, input.MimeType).Return(detected, nil)
	env.OnActivity("PurgeRawImageActivity", mock.Anything, input.JobID, len(input.ImageData)).Return(nil)
	env.OnActivity("PersistMealLogActivity", mock.Anything, input.UserUUID, input.MealType, input.Notes, detected).Return("", errors.New("write failed"))
	env.OnActivity("UpdateJobStatusActivity", mock.Anything, input.JobID, input.UserUUID, "FAILED").Return(nil)

	env.ExecuteWorkflow(ProcessMealImageWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *WorkflowTestSuite) Test_ProcessMealImageWorkflow_CompleteJobError() {
	env := s.NewTestWorkflowEnvironment()
	registerActivitiesOnEnv(env)

	input := ProcessMealWorkflowInput{
		JobID:     "job-complete-fail",
		UserUUID:  "user-complete-fail",
		ImageData: []byte("image-data"),
		MimeType:  "image/jpeg",
	}

	detected := []core.MealItem{{Name: "Oatmeal", Calories: 150}}

	env.OnActivity("UpdateJobStatusActivity", mock.Anything, input.JobID, input.UserUUID, "PROCESSING").Return(nil)
	env.OnActivity("AnalyzeMealImageActivity", mock.Anything, input.ImageData, input.MimeType).Return(detected, nil)
	env.OnActivity("PurgeRawImageActivity", mock.Anything, input.JobID, len(input.ImageData)).Return(nil)
	env.OnActivity("PersistMealLogActivity", mock.Anything, input.UserUUID, input.MealType, input.Notes, detected).Return("meal-log-1", nil)
	env.OnActivity("CompleteJobActivity", mock.Anything, input.JobID, input.UserUUID, "meal-log-1", detected).Return(errors.New("complete failed"))

	env.ExecuteWorkflow(ProcessMealImageWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}
