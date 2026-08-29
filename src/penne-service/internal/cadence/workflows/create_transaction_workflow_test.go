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

func (s *UnitTestSuite) Test_CreateTransactionWorkflow_NoPendingIntent_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.PendingShortcutIntentActivity, activity.RegisterOptions{Name: "PendingShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.CreateTransaction, activity.RegisterOptions{Name: "CreateTransactionActivity"})

	expectedTxnID := uuid.New()
	env.OnActivity("PendingShortcutIntentActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*core.ShortcutIntent)(nil), nil)
	env.OnActivity("CreateTransactionActivity", mock.Anything, mock.Anything).Return(&expectedTxnID, nil)

	env.ExecuteWorkflow(CreateTransactionWorkflow, core.Transaction{})

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result *uuid.UUID
	s.NoError(env.GetWorkflowResult(&result))
	s.NotNil(result)
	s.Equal(expectedTxnID, *result)
}

func (s *UnitTestSuite) Test_CreateTransactionWorkflow_WithPendingIntent_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.PendingShortcutIntentActivity, activity.RegisterOptions{Name: "PendingShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.UpdateShortcutIntentActivity, activity.RegisterOptions{Name: "UpdateShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.CreateTransaction, activity.RegisterOptions{Name: "CreateTransactionActivity"})

	intentID := uuid.New()
	expectedTxnID := uuid.New()
	pendingIntent := &core.ShortcutIntent{ID: intentID}

	env.OnActivity("PendingShortcutIntentActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pendingIntent, nil)
	env.OnActivity("UpdateShortcutIntentActivity", mock.Anything, mock.Anything).Return(&intentID, nil)
	env.OnActivity("CreateTransactionActivity", mock.Anything, mock.Anything).Return(&expectedTxnID, nil)

	env.ExecuteWorkflow(CreateTransactionWorkflow, core.Transaction{})

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result *uuid.UUID
	s.NoError(env.GetWorkflowResult(&result))
	s.NotNil(result)
	s.Equal(expectedTxnID, *result)
}

func (s *UnitTestSuite) Test_CreateTransactionWorkflow_PendingIntentError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.PendingShortcutIntentActivity, activity.RegisterOptions{Name: "PendingShortcutIntentActivity"})

	env.OnActivity("PendingShortcutIntentActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*core.ShortcutIntent)(nil), errors.New("pending intent failed"))

	env.ExecuteWorkflow(CreateTransactionWorkflow, core.Transaction{})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateTransactionWorkflow_UpdateIntentError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.PendingShortcutIntentActivity, activity.RegisterOptions{Name: "PendingShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.UpdateShortcutIntentActivity, activity.RegisterOptions{Name: "UpdateShortcutIntentActivity"})

	intentID := uuid.New()
	pendingIntent := &core.ShortcutIntent{ID: intentID}

	env.OnActivity("PendingShortcutIntentActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(pendingIntent, nil)
	env.OnActivity("UpdateShortcutIntentActivity", mock.Anything, mock.Anything).Return((*uuid.UUID)(nil), errors.New("update intent failed"))

	env.ExecuteWorkflow(CreateTransactionWorkflow, core.Transaction{})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateTransactionWorkflow_CreateTxnError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.PendingShortcutIntentActivity, activity.RegisterOptions{Name: "PendingShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.CreateTransaction, activity.RegisterOptions{Name: "CreateTransactionActivity"})

	env.OnActivity("PendingShortcutIntentActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*core.ShortcutIntent)(nil), nil)
	env.OnActivity("CreateTransactionActivity", mock.Anything, mock.Anything).Return((*uuid.UUID)(nil), errors.New("create transaction failed"))

	env.ExecuteWorkflow(CreateTransactionWorkflow, core.Transaction{})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateTransactionWorkflow_NonZeroCreatedAt() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.PendingShortcutIntentActivity, activity.RegisterOptions{Name: "PendingShortcutIntentActivity"})
	env.RegisterActivityWithOptions(acts.CreateTransaction, activity.RegisterOptions{Name: "CreateTransactionActivity"})

	expectedTxnID := uuid.New()
	env.OnActivity("PendingShortcutIntentActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*core.ShortcutIntent)(nil), nil)
	env.OnActivity("CreateTransactionActivity", mock.Anything, mock.Anything).Return(&expectedTxnID, nil)

	txn := core.Transaction{
		CreatedAt: time.Now(),
	}
	env.ExecuteWorkflow(CreateTransactionWorkflow, txn)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result *uuid.UUID
	s.NoError(env.GetWorkflowResult(&result))
	s.NotNil(result)
	s.Equal(expectedTxnID, *result)
}

