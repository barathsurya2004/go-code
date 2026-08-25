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

	err := workflow.ExecuteActivity(ctx, "CreateTransaction", txn).Get(ctx, &txnID)

	if err != nil {
		return nil, err
	}

	return &txnID, nil

}
