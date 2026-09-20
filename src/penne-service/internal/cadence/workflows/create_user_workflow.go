package workflows

import (
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/cadence/workflow"
)

func CreateUserWorkflow(ctx workflow.Context, user core.User) (*core.CreateUserWorkflowResult, error) {
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		ScheduleToStartTimeout: 2 * time.Minute,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Create User
	var userUUID *uuid.UUID
	err := workflow.ExecuteActivity(ctx, "CreateUserActivity", user).Get(ctx, &userUUID)
	if err != nil {
		return nil, err
	}

	// 2. Create System Envelope Group
	var groupUUID *uuid.UUID
	err = workflow.ExecuteActivity(ctx, "CreateSystemEnvelopeGroupActivity", *userUUID).Get(ctx, &groupUUID)
	if err != nil {
		return nil, err
	}

	// 3. Create System Envelope
	var envUUID *uuid.UUID
	envInput := core.CreateSystemEnvelopeActivityInput{
		UserUUID:        *userUUID,
		EnvelopeGroupID: *groupUUID,
	}
	err = workflow.ExecuteActivity(ctx, "CreateSystemEnvelopeActivity", envInput).Get(ctx, &envUUID)
	if err != nil {
		return nil, err
	}

	// 4. Create Default Allocation
	var allocUUID *uuid.UUID
	err = workflow.ExecuteActivity(ctx, "CreateDefaultAllocationActivity", *envUUID).Get(ctx, &allocUUID)
	if err != nil {
		return nil, err
	}

	// 5. Create User Auth Token
	var tokenUUID *uuid.UUID
	err = workflow.ExecuteActivity(ctx, "CreateUserTokenActivity", *userUUID).Get(ctx, &tokenUUID)
	if err != nil {
		return nil, err
	}

	return &core.CreateUserWorkflowResult{
		UserUUID:      *userUUID,
		UserAuthToken: *tokenUUID,
	}, nil
}
