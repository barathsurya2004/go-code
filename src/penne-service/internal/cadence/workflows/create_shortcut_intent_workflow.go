package workflows

import (
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/cadence/workflow"
)

func CreateShortcutIntentWorkflow(ctx workflow.Context, shortcutIntent core.ShortcutIntent) (*core.ShortcutIntent, error) {
	if shortcutIntent.CreatedAt.IsZero() {
		shortcutIntent.CreatedAt = utils.NowUTC()
	} else {
		shortcutIntent.CreatedAt = shortcutIntent.CreatedAt.UTC()
	}
	if shortcutIntent.Status == "" {
		shortcutIntent.Status = core.StatusPending
	}

	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		ScheduleToStartTimeout: 2 * time.Minute,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	var intentID *uuid.UUID
	err := workflow.ExecuteActivity(ctx, "CreateShortcutIntentActivity", shortcutIntent).Get(ctx, &intentID)
	if err != nil {
		return nil, err
	}
	if intentID != nil {
		shortcutIntent.ID = *intentID
	}

	lowTime := shortcutIntent.CreatedAt.Add(-10 * time.Minute).UTC()
	highTime := shortcutIntent.CreatedAt.Add(5 * time.Minute).UTC()

	var matchingTxn *core.Transaction
	err = workflow.ExecuteActivity(ctx, "GetTransactionByTimeActivity", lowTime, highTime).Get(ctx, &matchingTxn)
	if err != nil {
		return nil, err
	}

	if matchingTxn != nil {
		matchingTxn.EnvelopeID = shortcutIntent.EnvelopeID
		matchingTxn.ShortcutIntentID = &shortcutIntent.ID
		err = workflow.ExecuteActivity(ctx, "UpdateTransactionActivity", *matchingTxn).Get(ctx, nil)
		if err != nil {
			return nil, err
		}

		shortcutIntent.Status = core.StatusSettled
		shortcutIntent.TransactionID = &matchingTxn.ID
		err = workflow.ExecuteActivity(ctx, "UpdateShortcutIntentActivity", &shortcutIntent).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
	}

	return &shortcutIntent, nil
}
