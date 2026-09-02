package workflows

import (
	"errors"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/activities"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
)

func (s *UnitTestSuite) Test_CreateShortcutIntentWorkflow_NoMatchedTransaction_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateShortcutIntent, activity.RegisterOptions{Name: "CreateShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.GetTransactionByTimeActivity, activity.RegisterOptions{Name: "GetTransactionByTimeActivity"})

	intentID := uuid.New()
	env.OnActivity("CreateShortcutIntentActivity", mock.Anything, mock.Anything).Return(&intentID, nil)
	env.OnActivity("GetTransactionByTimeActivity", mock.Anything, mock.Anything, mock.Anything).Return((*core.Transaction)(nil), nil)

	env.ExecuteWorkflow(CreateShortcutIntentWorkflow, core.ShortcutIntent{})

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result *core.ShortcutIntent
	s.NoError(env.GetWorkflowResult(&result))
	s.NotNil(result)
	s.Equal(intentID, result.ID)
	s.Equal(core.StatusPending, result.Status)
}

func (s *UnitTestSuite) Test_CreateShortcutIntentWorkflow_WithMatchedTransaction_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateShortcutIntent, activity.RegisterOptions{Name: "CreateShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.GetTransactionByTimeActivity, activity.RegisterOptions{Name: "GetTransactionByTimeActivity"})
	env.RegisterActivityWithOptions(acts.UpdateTransactionActivity, activity.RegisterOptions{Name: "UpdateTransactionActivity"})
	env.RegisterActivityWithOptions(acts.UpdateShortcutIntentActivity, activity.RegisterOptions{Name: "UpdateShortcutIntentActivity"})

	intentID := uuid.New()
	txnID := uuid.New()
	envID := uuid.New()
	matchedTxn := &core.Transaction{ID: txnID}

	env.OnActivity("CreateShortcutIntentActivity", mock.Anything, mock.Anything).Return(&intentID, nil)
	env.OnActivity("GetTransactionByTimeActivity", mock.Anything, mock.Anything, mock.Anything).Return(matchedTxn, nil)
	env.OnActivity("UpdateTransactionActivity", mock.Anything, mock.Anything).Return(nil)
	env.OnActivity("UpdateShortcutIntentActivity", mock.Anything, mock.Anything).Return(&txnID, nil)

	env.ExecuteWorkflow(CreateShortcutIntentWorkflow, core.ShortcutIntent{
		EnvelopeID: &envID,
	})

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result *core.ShortcutIntent
	s.NoError(env.GetWorkflowResult(&result))
	s.NotNil(result)
	s.Equal(intentID, result.ID)
	s.Equal(core.StatusSettled, result.Status)
	s.NotNil(result.TransactionID)
	s.Equal(txnID, *result.TransactionID)
}

func (s *UnitTestSuite) Test_CreateShortcutIntentWorkflow_CreateIntentError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateShortcutIntent, activity.RegisterOptions{Name: "CreateShortcutIntentActivity"})

	env.OnActivity("CreateShortcutIntentActivity", mock.Anything, mock.Anything).Return((*uuid.UUID)(nil), errors.New("create intent failed"))

	env.ExecuteWorkflow(CreateShortcutIntentWorkflow, core.ShortcutIntent{})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateShortcutIntentWorkflow_GetTransactionError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateShortcutIntent, activity.RegisterOptions{Name: "CreateShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.GetTransactionByTimeActivity, activity.RegisterOptions{Name: "GetTransactionByTimeActivity"})

	intentID := uuid.New()
	env.OnActivity("CreateShortcutIntentActivity", mock.Anything, mock.Anything).Return(&intentID, nil)
	env.OnActivity("GetTransactionByTimeActivity", mock.Anything, mock.Anything, mock.Anything).Return((*core.Transaction)(nil), errors.New("get txn failed"))

	env.ExecuteWorkflow(CreateShortcutIntentWorkflow, core.ShortcutIntent{})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateShortcutIntentWorkflow_UpdateTransactionError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateShortcutIntent, activity.RegisterOptions{Name: "CreateShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.GetTransactionByTimeActivity, activity.RegisterOptions{Name: "GetTransactionByTimeActivity"})
	env.RegisterActivityWithOptions(acts.UpdateTransactionActivity, activity.RegisterOptions{Name: "UpdateTransactionActivity"})

	intentID := uuid.New()
	txnID := uuid.New()
	matchedTxn := &core.Transaction{ID: txnID}

	env.OnActivity("CreateShortcutIntentActivity", mock.Anything, mock.Anything).Return(&intentID, nil)
	env.OnActivity("GetTransactionByTimeActivity", mock.Anything, mock.Anything, mock.Anything).Return(matchedTxn, nil)
	env.OnActivity("UpdateTransactionActivity", mock.Anything, mock.Anything).Return(errors.New("update txn failed"))

	env.ExecuteWorkflow(CreateShortcutIntentWorkflow, core.ShortcutIntent{})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateShortcutIntentWorkflow_UpdateIntentError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateShortcutIntent, activity.RegisterOptions{Name: "CreateShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.GetTransactionByTimeActivity, activity.RegisterOptions{Name: "GetTransactionByTimeActivity"})
	env.RegisterActivityWithOptions(acts.UpdateTransactionActivity, activity.RegisterOptions{Name: "UpdateTransactionActivity"})
	env.RegisterActivityWithOptions(acts.UpdateShortcutIntentActivity, activity.RegisterOptions{Name: "UpdateShortcutIntentActivity"})

	intentID := uuid.New()
	txnID := uuid.New()
	matchedTxn := &core.Transaction{ID: txnID}

	env.OnActivity("CreateShortcutIntentActivity", mock.Anything, mock.Anything).Return(&intentID, nil)
	env.OnActivity("GetTransactionByTimeActivity", mock.Anything, mock.Anything, mock.Anything).Return(matchedTxn, nil)
	env.OnActivity("UpdateTransactionActivity", mock.Anything, mock.Anything).Return(nil)
	env.OnActivity("UpdateShortcutIntentActivity", mock.Anything, mock.Anything).Return((*uuid.UUID)(nil), errors.New("update intent failed"))

	env.ExecuteWorkflow(CreateShortcutIntentWorkflow, core.ShortcutIntent{})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateShortcutIntentWorkflow_NonZeroCreatedAt() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateShortcutIntent, activity.RegisterOptions{Name: "CreateShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.GetTransactionByTimeActivity, activity.RegisterOptions{Name: "GetTransactionByTimeActivity"})

	intentID := uuid.New()
	env.OnActivity("CreateShortcutIntentActivity", mock.Anything, mock.Anything).Return(&intentID, nil)
	env.OnActivity("GetTransactionByTimeActivity", mock.Anything, mock.Anything, mock.Anything).Return((*core.Transaction)(nil), nil)

	intent := core.ShortcutIntent{
		CreatedAt: time.Now(),
	}
	env.ExecuteWorkflow(CreateShortcutIntentWorkflow, intent)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result *core.ShortcutIntent
	s.NoError(env.GetWorkflowResult(&result))
	s.NotNil(result)
	s.Equal(intentID, result.ID)
}
