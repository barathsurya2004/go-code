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

func (s *UnitTestSuite) Test_CreateUserWorkflow_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewUserActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateUserActivity, activity.RegisterOptions{Name: "CreateUserActivity"})
	env.RegisterActivityWithOptions(acts.CreateSystemEnvelopeGroupActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeGroupActivity"})
	env.RegisterActivityWithOptions(acts.CreateSystemEnvelopeActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeActivity"})
	env.RegisterActivityWithOptions(acts.CreateDefaultAllocationActivity, activity.RegisterOptions{Name: "CreateDefaultAllocationActivity"})
	env.RegisterActivityWithOptions(acts.CreateUserTokenActivity, activity.RegisterOptions{Name: "CreateUserTokenActivity"})

	expectedUserUUID := uuid.New()
	expectedGroupUUID := uuid.New()
	expectedEnvUUID := uuid.New()
	expectedAllocUUID := uuid.New()
	expectedTokenUUID := uuid.New()

	env.OnActivity("CreateUserActivity", mock.Anything, mock.Anything).Return(&expectedUserUUID, nil)
	env.OnActivity("CreateSystemEnvelopeGroupActivity", mock.Anything, expectedUserUUID).Return(&expectedGroupUUID, nil)
	env.OnActivity("CreateSystemEnvelopeActivity", mock.Anything, core.CreateSystemEnvelopeActivityInput{
		UserUUID:        expectedUserUUID,
		EnvelopeGroupID: expectedGroupUUID,
	}).Return(&expectedEnvUUID, nil)
	env.OnActivity("CreateDefaultAllocationActivity", mock.Anything, expectedEnvUUID).Return(&expectedAllocUUID, nil)
	env.OnActivity("CreateUserTokenActivity", mock.Anything, expectedUserUUID).Return(&expectedTokenUUID, nil)

	env.ExecuteWorkflow(CreateUserWorkflow, core.User{Name: "Alice"})

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result *core.CreateUserWorkflowResult
	s.NoError(env.GetWorkflowResult(&result))
	s.NotNil(result)
	s.Equal(expectedUserUUID, result.UserUUID)
	s.Equal(expectedTokenUUID, result.UserAuthToken)
}

func (s *UnitTestSuite) Test_CreateUserWorkflow_CreateUserError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewUserActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateUserActivity, activity.RegisterOptions{Name: "CreateUserActivity"})

	env.OnActivity("CreateUserActivity", mock.Anything, mock.Anything).Return((*uuid.UUID)(nil), errors.New("user creation failed"))

	env.ExecuteWorkflow(CreateUserWorkflow, core.User{Name: "Alice"})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateUserWorkflow_CreateEnvelopeGroupError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewUserActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateUserActivity, activity.RegisterOptions{Name: "CreateUserActivity"})
	env.RegisterActivityWithOptions(acts.CreateSystemEnvelopeGroupActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeGroupActivity"})

	expectedUserUUID := uuid.New()
	env.OnActivity("CreateUserActivity", mock.Anything, mock.Anything).Return(&expectedUserUUID, nil)
	env.OnActivity("CreateSystemEnvelopeGroupActivity", mock.Anything, expectedUserUUID).Return((*uuid.UUID)(nil), errors.New("group creation failed"))

	env.ExecuteWorkflow(CreateUserWorkflow, core.User{Name: "Alice"})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateUserWorkflow_CreateEnvelopeError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewUserActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateUserActivity, activity.RegisterOptions{Name: "CreateUserActivity"})
	env.RegisterActivityWithOptions(acts.CreateSystemEnvelopeGroupActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeGroupActivity"})
	env.RegisterActivityWithOptions(acts.CreateSystemEnvelopeActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeActivity"})

	expectedUserUUID := uuid.New()
	expectedGroupUUID := uuid.New()
	env.OnActivity("CreateUserActivity", mock.Anything, mock.Anything).Return(&expectedUserUUID, nil)
	env.OnActivity("CreateSystemEnvelopeGroupActivity", mock.Anything, expectedUserUUID).Return(&expectedGroupUUID, nil)
	env.OnActivity("CreateSystemEnvelopeActivity", mock.Anything, mock.Anything).Return((*uuid.UUID)(nil), errors.New("envelope creation failed"))

	env.ExecuteWorkflow(CreateUserWorkflow, core.User{Name: "Alice"})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateUserWorkflow_CreateAllocationError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewUserActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateUserActivity, activity.RegisterOptions{Name: "CreateUserActivity"})
	env.RegisterActivityWithOptions(acts.CreateSystemEnvelopeGroupActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeGroupActivity"})
	env.RegisterActivityWithOptions(acts.CreateSystemEnvelopeActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeActivity"})
	env.RegisterActivityWithOptions(acts.CreateDefaultAllocationActivity, activity.RegisterOptions{Name: "CreateDefaultAllocationActivity"})

	expectedUserUUID := uuid.New()
	expectedGroupUUID := uuid.New()
	expectedEnvUUID := uuid.New()
	env.OnActivity("CreateUserActivity", mock.Anything, mock.Anything).Return(&expectedUserUUID, nil)
	env.OnActivity("CreateSystemEnvelopeGroupActivity", mock.Anything, expectedUserUUID).Return(&expectedGroupUUID, nil)
	env.OnActivity("CreateSystemEnvelopeActivity", mock.Anything, mock.Anything).Return(&expectedEnvUUID, nil)
	env.OnActivity("CreateDefaultAllocationActivity", mock.Anything, expectedEnvUUID).Return((*uuid.UUID)(nil), errors.New("allocation failed"))

	env.ExecuteWorkflow(CreateUserWorkflow, core.User{Name: "Alice"})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_CreateUserWorkflow_CreateTokenError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	acts := activities.NewUserActivities(core.RepoContainer{}, logger)
	env.RegisterActivityWithOptions(acts.CreateUserActivity, activity.RegisterOptions{Name: "CreateUserActivity"})
	env.RegisterActivityWithOptions(acts.CreateSystemEnvelopeGroupActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeGroupActivity"})
	env.RegisterActivityWithOptions(acts.CreateSystemEnvelopeActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeActivity"})
	env.RegisterActivityWithOptions(acts.CreateDefaultAllocationActivity, activity.RegisterOptions{Name: "CreateDefaultAllocationActivity"})
	env.RegisterActivityWithOptions(acts.CreateUserTokenActivity, activity.RegisterOptions{Name: "CreateUserTokenActivity"})

	expectedUserUUID := uuid.New()
	expectedGroupUUID := uuid.New()
	expectedEnvUUID := uuid.New()
	expectedAllocUUID := uuid.New()
	env.OnActivity("CreateUserActivity", mock.Anything, mock.Anything).Return(&expectedUserUUID, nil)
	env.OnActivity("CreateSystemEnvelopeGroupActivity", mock.Anything, expectedUserUUID).Return(&expectedGroupUUID, nil)
	env.OnActivity("CreateSystemEnvelopeActivity", mock.Anything, mock.Anything).Return(&expectedEnvUUID, nil)
	env.OnActivity("CreateDefaultAllocationActivity", mock.Anything, expectedEnvUUID).Return(&expectedAllocUUID, nil)
	env.OnActivity("CreateUserTokenActivity", mock.Anything, expectedUserUUID).Return((*uuid.UUID)(nil), errors.New("token creation failed"))

	env.ExecuteWorkflow(CreateUserWorkflow, core.User{Name: "Alice"})

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}
