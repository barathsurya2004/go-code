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

func (s *UnitTestSuite) Test_UpdateTransactionCategoryWorkflow_ChangeCategory_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})
	env.RegisterActivityWithOptions(acts.UpdateAllocationSpentActivity, activity.RegisterOptions{Name: "UpdateAllocationSpentActivity"})
	env.RegisterActivityWithOptions(acts.UpdateTransactionActivity, activity.RegisterOptions{Name: "UpdateTransactionActivity"})

	txnID := uuid.New()
	oldEnvID := uuid.New()
	newEnvID := uuid.New()
	existingTxn := core.Transaction{
		ID:         txnID,
		EnvelopeID: &oldEnvID,
		AmountE5:   500000,
		Type:       "debit",
		CreatedAt:  time.Now().UTC(),
	}

	req := core.UpdateTransactionCategoryRequest{
		TransactionID: txnID,
		NewEnvelopeID: &newEnvID,
	}

	env.OnActivity("GetTransactionByIDActivity", mock.Anything, txnID).Return(&existingTxn, nil)
	env.OnActivity("UpdateAllocationSpentActivity", mock.Anything, oldEnvID, mock.Anything, int64(-500000)).Return(nil)
	env.OnActivity("UpdateAllocationSpentActivity", mock.Anything, newEnvID, mock.Anything, int64(500000)).Return(nil)
	env.OnActivity("UpdateTransactionActivity", mock.Anything, mock.MatchedBy(func(t core.Transaction) bool {
		return t.ID == txnID && t.EnvelopeID != nil && *t.EnvelopeID == newEnvID
	})).Return(nil)

	env.ExecuteWorkflow(UpdateTransactionCategoryWorkflow, req)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateTransactionCategoryWorkflow_GetTxnError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})

	txnID := uuid.New()
	req := core.UpdateTransactionCategoryRequest{
		TransactionID: txnID,
	}

	env.OnActivity("GetTransactionByIDActivity", mock.Anything, txnID).Return((*core.Transaction)(nil), errors.New("db error"))

	env.ExecuteWorkflow(UpdateTransactionCategoryWorkflow, req)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateTransactionCategoryWorkflow_AssignCategoryToUncategorized_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})
	env.RegisterActivityWithOptions(acts.UpdateAllocationSpentActivity, activity.RegisterOptions{Name: "UpdateAllocationSpentActivity"})
	env.RegisterActivityWithOptions(acts.UpdateTransactionActivity, activity.RegisterOptions{Name: "UpdateTransactionActivity"})

	txnID := uuid.New()
	newEnvID := uuid.New()
	existingTxn := core.Transaction{
		ID:         txnID,
		EnvelopeID: nil,
		AmountE5:   200000,
		Type:       "debit",
		CreatedAt:  time.Now().UTC(),
	}

	req := core.UpdateTransactionCategoryRequest{
		TransactionID: txnID,
		NewEnvelopeID: &newEnvID,
	}

	env.OnActivity("GetTransactionByIDActivity", mock.Anything, txnID).Return(&existingTxn, nil)
	env.OnActivity("UpdateAllocationSpentActivity", mock.Anything, newEnvID, mock.Anything, int64(200000)).Return(nil)
	env.OnActivity("UpdateTransactionActivity", mock.Anything, mock.MatchedBy(func(t core.Transaction) bool {
		return t.ID == txnID && t.EnvelopeID != nil && *t.EnvelopeID == newEnvID
	})).Return(nil)

	env.ExecuteWorkflow(UpdateTransactionCategoryWorkflow, req)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateTransactionCategoryWorkflow_RemoveCategory_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})
	env.RegisterActivityWithOptions(acts.UpdateAllocationSpentActivity, activity.RegisterOptions{Name: "UpdateAllocationSpentActivity"})
	env.RegisterActivityWithOptions(acts.UpdateTransactionActivity, activity.RegisterOptions{Name: "UpdateTransactionActivity"})

	txnID := uuid.New()
	oldEnvID := uuid.New()
	existingTxn := core.Transaction{
		ID:         txnID,
		EnvelopeID: &oldEnvID,
		AmountE5:   300000,
		Type:       "debit",
		CreatedAt:  time.Now().UTC(),
	}

	req := core.UpdateTransactionCategoryRequest{
		TransactionID: txnID,
		NewEnvelopeID: nil,
	}

	env.OnActivity("GetTransactionByIDActivity", mock.Anything, txnID).Return(&existingTxn, nil)
	env.OnActivity("UpdateAllocationSpentActivity", mock.Anything, oldEnvID, mock.Anything, int64(-300000)).Return(nil)
	env.OnActivity("UpdateTransactionActivity", mock.Anything, mock.MatchedBy(func(t core.Transaction) bool {
		return t.ID == txnID && t.EnvelopeID == nil
	})).Return(nil)

	env.ExecuteWorkflow(UpdateTransactionCategoryWorkflow, req)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateTransactionCategoryWorkflow_SameCategory_DeltaSuccess() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})
	env.RegisterActivityWithOptions(acts.UpdateAllocationSpentActivity, activity.RegisterOptions{Name: "UpdateAllocationSpentActivity"})
	env.RegisterActivityWithOptions(acts.UpdateTransactionActivity, activity.RegisterOptions{Name: "UpdateTransactionActivity"})

	txnID := uuid.New()
	envID := uuid.New()
	existingTxn := core.Transaction{
		ID:         txnID,
		EnvelopeID: &envID,
		AmountE5:   100000,
		Type:       "debit",
		CreatedAt:  time.Time{}, // triggers fallback to utils.NowUTC()
	}

	req := core.UpdateTransactionCategoryRequest{
		TransactionID: txnID,
		NewEnvelopeID: &envID,
		AmountE5:      150000,
		TxnType:       "debit",
		PaymentMethod: "card",
	}

	env.OnActivity("GetTransactionByIDActivity", mock.Anything, txnID).Return(&existingTxn, nil)
	env.OnActivity("UpdateAllocationSpentActivity", mock.Anything, envID, mock.Anything, int64(50000)).Return(nil)
	env.OnActivity("UpdateTransactionActivity", mock.Anything, mock.MatchedBy(func(t core.Transaction) bool {
		return t.ID == txnID && t.AmountE5 == 150000
	})).Return(nil)

	env.ExecuteWorkflow(UpdateTransactionCategoryWorkflow, req)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateTransactionCategoryWorkflow_DeductOldError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})
	env.RegisterActivityWithOptions(acts.UpdateAllocationSpentActivity, activity.RegisterOptions{Name: "UpdateAllocationSpentActivity"})

	txnID := uuid.New()
	oldEnvID := uuid.New()
	newEnvID := uuid.New()
	existingTxn := core.Transaction{
		ID:         txnID,
		EnvelopeID: &oldEnvID,
		AmountE5:   500000,
		Type:       "debit",
	}

	req := core.UpdateTransactionCategoryRequest{
		TransactionID: txnID,
		NewEnvelopeID: &newEnvID,
	}

	env.OnActivity("GetTransactionByIDActivity", mock.Anything, txnID).Return(&existingTxn, nil)
	env.OnActivity("UpdateAllocationSpentActivity", mock.Anything, oldEnvID, mock.Anything, int64(-500000)).Return(errors.New("deduct failed"))

	env.ExecuteWorkflow(UpdateTransactionCategoryWorkflow, req)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateTransactionCategoryWorkflow_AddNewError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})
	env.RegisterActivityWithOptions(acts.UpdateAllocationSpentActivity, activity.RegisterOptions{Name: "UpdateAllocationSpentActivity"})

	txnID := uuid.New()
	oldEnvID := uuid.New()
	newEnvID := uuid.New()
	existingTxn := core.Transaction{
		ID:         txnID,
		EnvelopeID: &oldEnvID,
		AmountE5:   500000,
		Type:       "debit",
	}

	req := core.UpdateTransactionCategoryRequest{
		TransactionID: txnID,
		NewEnvelopeID: &newEnvID,
	}

	env.OnActivity("GetTransactionByIDActivity", mock.Anything, txnID).Return(&existingTxn, nil)
	env.OnActivity("UpdateAllocationSpentActivity", mock.Anything, oldEnvID, mock.Anything, int64(-500000)).Return(nil)
	env.OnActivity("UpdateAllocationSpentActivity", mock.Anything, newEnvID, mock.Anything, int64(500000)).Return(errors.New("add failed"))

	env.ExecuteWorkflow(UpdateTransactionCategoryWorkflow, req)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateTransactionCategoryWorkflow_SameCategory_DeltaError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})
	env.RegisterActivityWithOptions(acts.UpdateAllocationSpentActivity, activity.RegisterOptions{Name: "UpdateAllocationSpentActivity"})

	txnID := uuid.New()
	envID := uuid.New()
	existingTxn := core.Transaction{
		ID:         txnID,
		EnvelopeID: &envID,
		AmountE5:   100000,
		Type:       "debit",
	}

	req := core.UpdateTransactionCategoryRequest{
		TransactionID: txnID,
		NewEnvelopeID: &envID,
		AmountE5:      150000,
	}

	env.OnActivity("GetTransactionByIDActivity", mock.Anything, txnID).Return(&existingTxn, nil)
	env.OnActivity("UpdateAllocationSpentActivity", mock.Anything, envID, mock.Anything, int64(50000)).Return(errors.New("delta failed"))

	env.ExecuteWorkflow(UpdateTransactionCategoryWorkflow, req)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateTransactionCategoryWorkflow_UpdateTxnError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})
	env.RegisterActivityWithOptions(acts.UpdateAllocationSpentActivity, activity.RegisterOptions{Name: "UpdateAllocationSpentActivity"})
	env.RegisterActivityWithOptions(acts.UpdateTransactionActivity, activity.RegisterOptions{Name: "UpdateTransactionActivity"})

	txnID := uuid.New()
	envID := uuid.New()
	existingTxn := core.Transaction{
		ID:         txnID,
		EnvelopeID: nil,
		AmountE5:   100000,
		Type:       "debit",
	}

	req := core.UpdateTransactionCategoryRequest{
		TransactionID: txnID,
		NewEnvelopeID: &envID,
	}

	env.OnActivity("GetTransactionByIDActivity", mock.Anything, txnID).Return(&existingTxn, nil)
	env.OnActivity("UpdateAllocationSpentActivity", mock.Anything, envID, mock.Anything, int64(100000)).Return(nil)
	env.OnActivity("UpdateTransactionActivity", mock.Anything, mock.Anything).Return(errors.New("update txn failed"))

	env.ExecuteWorkflow(UpdateTransactionCategoryWorkflow, req)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_UpdateTransactionCategoryWorkflow_NonDebitType() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewTransactionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})
	env.RegisterActivityWithOptions(acts.UpdateTransactionActivity, activity.RegisterOptions{Name: "UpdateTransactionActivity"})

	txnID := uuid.New()
	oldEnvID := uuid.New()
	newEnvID := uuid.New()
	existingTxn := core.Transaction{
		ID:         txnID,
		EnvelopeID: &oldEnvID,
		AmountE5:   500000,
		Type:       "credit",
	}

	req := core.UpdateTransactionCategoryRequest{
		TransactionID: txnID,
		NewEnvelopeID: &newEnvID,
		TxnType:       "credit",
	}

	env.OnActivity("GetTransactionByIDActivity", mock.Anything, txnID).Return(&existingTxn, nil)
	env.OnActivity("UpdateTransactionActivity", mock.Anything, mock.Anything).Return(nil)

	env.ExecuteWorkflow(UpdateTransactionCategoryWorkflow, req)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
}

