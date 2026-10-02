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

func (s *UnitTestSuite) Test_CreateSubscriptionWorkflow_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewSubscriptionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateSubscriptionActivity, activity.RegisterOptions{Name: "CreateSubscriptionActivity"})

	sub := &core.Subscription{Name: "Netflix"}
	expectedID := uuid.New()
	env.OnActivity("CreateSubscriptionActivity", mock.Anything, sub).Return(expectedID, nil)

	env.ExecuteWorkflow(CreateSubscriptionWorkflow, sub)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var id uuid.UUID
	s.NoError(env.GetWorkflowResult(&id))
	s.Equal(expectedID, id)
}

func (s *UnitTestSuite) Test_CreateSubscriptionWorkflow_Error() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewSubscriptionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateSubscriptionActivity, activity.RegisterOptions{Name: "CreateSubscriptionActivity"})

	sub := &core.Subscription{Name: "Netflix"}
	env.OnActivity("CreateSubscriptionActivity", mock.Anything, sub).Return(uuid.Nil, errors.New("activity failed"))

	env.ExecuteWorkflow(CreateSubscriptionWorkflow, sub)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_RenewSubscriptionWorkflow_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewSubscriptionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.RenewSubscriptionActivity, activity.RegisterOptions{Name: "RenewSubscriptionActivity"})

	subID := uuid.New()
	expectedTxn := &core.Transaction{ID: uuid.New(), Description: "Subscription: Netflix"}
	env.OnActivity("RenewSubscriptionActivity", mock.Anything, subID).Return(expectedTxn, nil)

	env.ExecuteWorkflow(RenewSubscriptionWorkflow, subID)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var txn *core.Transaction
	s.NoError(env.GetWorkflowResult(&txn))
	s.Equal(expectedTxn.ID, txn.ID)
}

func (s *UnitTestSuite) Test_RenewSubscriptionWorkflow_Error() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewSubscriptionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.RenewSubscriptionActivity, activity.RegisterOptions{Name: "RenewSubscriptionActivity"})

	subID := uuid.New()
	env.OnActivity("RenewSubscriptionActivity", mock.Anything, subID).Return((*core.Transaction)(nil), errors.New("renewal failed"))

	env.ExecuteWorkflow(RenewSubscriptionWorkflow, subID)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ScanAndRenewDueSubscriptionsWorkflow_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewSubscriptionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetDueSubscriptionsActivity, activity.RegisterOptions{Name: "GetDueSubscriptionsActivity"})
	env.RegisterActivityWithOptions(acts.RenewSubscriptionActivity, activity.RegisterOptions{Name: "RenewSubscriptionActivity"})

	asOf := time.Now()
	sub1ID := uuid.New()
	sub2ID := uuid.New()
	dueSubs := []*core.Subscription{
		{ID: sub1ID, Name: "Netflix"},
		{ID: sub2ID, Name: "Spotify"},
	}

	env.OnActivity("GetDueSubscriptionsActivity", mock.Anything, mock.Anything).Return(dueSubs, nil)
	env.OnActivity("RenewSubscriptionActivity", mock.Anything, sub1ID).Return(&core.Transaction{ID: uuid.New()}, nil)
	env.OnActivity("RenewSubscriptionActivity", mock.Anything, sub2ID).Return((*core.Transaction)(nil), errors.New("sub 2 failed"))

	env.ExecuteWorkflow(ScanAndRenewDueSubscriptionsWorkflow, asOf)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var renewedIDs []uuid.UUID
	s.NoError(env.GetWorkflowResult(&renewedIDs))
	s.Equal(1, len(renewedIDs))
	s.Equal(sub1ID, renewedIDs[0])
}

func (s *UnitTestSuite) Test_ScanAndRenewDueSubscriptionsWorkflow_GetDueError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewSubscriptionActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.GetDueSubscriptionsActivity, activity.RegisterOptions{Name: "GetDueSubscriptionsActivity"})

	asOf := time.Now()
	env.OnActivity("GetDueSubscriptionsActivity", mock.Anything, mock.Anything).Return(([]*core.Subscription)(nil), errors.New("query failed"))

	env.ExecuteWorkflow(ScanAndRenewDueSubscriptionsWorkflow, asOf)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}
