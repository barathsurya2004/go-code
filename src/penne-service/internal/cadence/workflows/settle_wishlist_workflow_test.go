package workflows

import (
	"errors"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/activities"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
)

func (s *UnitTestSuite) Test_SettleWishlistWorkflow_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewWishlistActivities(core.RepoContainer{}, nil, logger)
	env.RegisterActivityWithOptions(acts.ApplyWishlistSurplusActivity, activity.RegisterOptions{Name: "ApplyWishlistSurplusActivity"})

	userUUID := uuid.New()
	expectedResults := []*core.ItemAllocationSimulation{
		{
			ItemID:      uuid.New(),
			AllocatedE5: 10000,
			IsFulfilled: true,
		},
	}

	env.OnActivity("ApplyWishlistSurplusActivity", mock.Anything, userUUID).Return(expectedResults, nil)

	env.ExecuteWorkflow(SettleWishlistWorkflow, userUUID)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var results []*core.ItemAllocationSimulation
	s.NoError(env.GetWorkflowResult(&results))
	s.Equal(1, len(results))
	s.Equal(int64(10000), results[0].AllocatedE5)
}

func (s *UnitTestSuite) Test_SettleWishlistWorkflow_Error() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewWishlistActivities(core.RepoContainer{}, nil, logger)
	env.RegisterActivityWithOptions(acts.ApplyWishlistSurplusActivity, activity.RegisterOptions{Name: "ApplyWishlistSurplusActivity"})

	userUUID := uuid.New()
	env.OnActivity("ApplyWishlistSurplusActivity", mock.Anything, userUUID).Return(([]*core.ItemAllocationSimulation)(nil), errors.New("settlement failed"))

	env.ExecuteWorkflow(SettleWishlistWorkflow, userUUID)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}
