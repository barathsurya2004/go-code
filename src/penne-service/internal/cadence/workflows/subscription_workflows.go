package workflows

import (
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/cadence/workflow"
)

func CreateSubscriptionWorkflow(ctx workflow.Context, sub *core.Subscription) (uuid.UUID, error) {
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		StartToCloseTimeout:    2 * time.Minute,
		ScheduleToStartTimeout: 1 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var id uuid.UUID
	err := workflow.ExecuteActivity(ctx, "CreateSubscriptionActivity", sub).Get(ctx, &id)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func RenewSubscriptionWorkflow(ctx workflow.Context, subscriptionID uuid.UUID) (*core.Transaction, error) {
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		ScheduleToStartTimeout: 2 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var txn *core.Transaction
	err := workflow.ExecuteActivity(ctx, "RenewSubscriptionActivity", subscriptionID).Get(ctx, &txn)
	if err != nil {
		return nil, err
	}

	return txn, nil
}

func ScanAndRenewDueSubscriptionsWorkflow(ctx workflow.Context, asOf time.Time) ([]uuid.UUID, error) {
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 15 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		ScheduleToStartTimeout: 2 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var dueSubs []*core.Subscription
	err := workflow.ExecuteActivity(ctx, "GetDueSubscriptionsActivity", asOf).Get(ctx, &dueSubs)
	if err != nil {
		return nil, err
	}

	var renewedIDs []uuid.UUID
	for _, sub := range dueSubs {
		var txn *core.Transaction
		err := workflow.ExecuteActivity(ctx, "RenewSubscriptionActivity", sub.ID).Get(ctx, &txn)
		if err == nil {
			renewedIDs = append(renewedIDs, sub.ID)
		}
	}

	return renewedIDs, nil
}
