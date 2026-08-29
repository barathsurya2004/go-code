package workflows

import (
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/cadence/workflow"
)

func CreateTransactionWorkflow(ctx workflow.Context, txn core.Transaction) (*uuid.UUID, error) {
	if txn.CreatedAt.IsZero() {
		txn.CreatedAt = utils.NowUTC()
	} else {
		txn.CreatedAt = txn.CreatedAt.UTC()
	}

	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		ScheduleToStartTimeout: 2 * time.Minute,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	var txnID uuid.UUID
	lowTime := txn.CreatedAt.Add(-5 * time.Second).UTC()
	highTime := txn.CreatedAt.Add(5 * time.Second).UTC()

	var pendingShortcutIntent *core.ShortcutIntent

	err := workflow.ExecuteActivity(ctx, "PendingShortcutIntentActivity", txn.UserID, lowTime, highTime).Get(ctx, &pendingShortcutIntent)
	if err != nil {
		return nil, err
	}

	if pendingShortcutIntent != nil {
		txn.ShortcutIntentID = &pendingShortcutIntent.ID
		txn.EnvelopeID = pendingShortcutIntent.EnvelopeID
		pendingShortcutIntent.Status = core.StatusSettled
		pendingShortcutIntent.TransactionID = &txn.ID
		err = workflow.ExecuteActivity(
			ctx,
			"UpdateShortcutIntentActivity",
			pendingShortcutIntent,
		).Get(ctx, nil)
		if err != nil {
			return nil, err
		}
		err = workflow.ExecuteActivity(
			ctx,
			"CreateTransactionActivity",
			txn,
		).Get(ctx, &txnID)
		if err != nil {
			return nil, err
		}
	} else {
		err = workflow.ExecuteActivity(
			ctx,
			"CreateTransactionActivity",
			txn,
		).Get(ctx, &txnID)

		if err != nil {
			return nil, err
		}
	}

	return &txnID, nil

}
