package workflows

import (
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/cadence/workflow"
)

func SettleWishlistWorkflow(ctx workflow.Context, userUUID uuid.UUID) ([]*core.ItemAllocationSimulation, error) {
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		ScheduleToStartTimeout: 2 * time.Minute,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	var results []*core.ItemAllocationSimulation
	err := workflow.ExecuteActivity(ctx, "ApplyWishlistSurplusActivity", userUUID).Get(ctx, &results)
	if err != nil {
		return nil, err
	}

	return results, nil
}
