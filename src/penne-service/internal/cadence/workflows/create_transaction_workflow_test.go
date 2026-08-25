package workflows

import (
	"errors"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/activities"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/cadence/activity"
)

func (s *UnitTestSuite) Test_CreateTransactionWorkflow() {
	env := s.NewTestWorkflowEnvironment()

	acts := activities.NewTransactionActivities(core.RepoContainer{})
	env.RegisterActivityWithOptions(acts.CreateTransaction, activity.RegisterOptions{Name: "CreateTransaction"})

	expectedUUID := uuid.New()
	env.OnActivity("CreateTransaction", mock.Anything, mock.Anything).Return(&expectedUUID, nil)

	env.ExecuteWorkflow(CreateTransactionWorkflow, core.Transaction{})

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result *uuid.UUID
	s.NoError(env.GetWorkflowResult(&result))
	s.NotNil(result)
	s.Equal(expectedUUID, *result)
}

func (s *UnitTestSuite) Test_CreateTransactionWorkflow_Error() {
	env := s.NewTestWorkflowEnvironment()

	acts := activities.NewTransactionActivities(core.RepoContainer{})
	env.RegisterActivityWithOptions(acts.CreateTransaction, activity.RegisterOptions{Name: "CreateTransaction"})

	env.OnActivity("CreateTransaction", mock.Anything, mock.Anything).Return((*uuid.UUID)(nil), errors.New("activity failed"))

	env.ExecuteWorkflow(CreateTransactionWorkflow, core.Transaction{})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}
