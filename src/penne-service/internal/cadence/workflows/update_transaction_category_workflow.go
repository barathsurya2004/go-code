package workflows

import (
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/cadence/workflow"
)

// UpdateTransactionCategoryWorkflow orchestrates updating a transaction's category (envelope attribution)
// and adjusting the spent amount of affected allocations in cadence steps.
func UpdateTransactionCategoryWorkflow(ctx workflow.Context, req core.UpdateTransactionCategoryRequest) error {
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		ScheduleToStartTimeout: 2 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Fetch the existing transaction
	var existingTxn core.Transaction
	err := workflow.ExecuteActivity(ctx, "GetTransactionByIDActivity", req.TransactionID).Get(ctx, &existingTxn)
	if err != nil {
		return err
	}

	oldEnvelopeID := existingTxn.EnvelopeID
	newEnvelopeID := req.NewEnvelopeID

	targetDate := existingTxn.CreatedAt
	if targetDate.IsZero() {
		targetDate = time.Now().UTC()
	}

	amountE5 := existingTxn.AmountE5
	if req.AmountE5 != 0 {
		amountE5 = req.AmountE5
	}

	txnType := existingTxn.Type
	if req.TxnType != "" {
		txnType = req.TxnType
	}

	// Check if category changed
	isCategoryChanged := false
	if (oldEnvelopeID == nil && newEnvelopeID != nil) || (oldEnvelopeID != nil && newEnvelopeID == nil) {
		isCategoryChanged = true
	} else if oldEnvelopeID != nil && newEnvelopeID != nil && *oldEnvelopeID != *newEnvelopeID {
		isCategoryChanged = true
	}

	if isCategoryChanged {
		// Step 2: Deduct spent amount from old envelope allocation if previous type was debit
		if oldEnvelopeID != nil && existingTxn.Type == "debit" {
			err = workflow.ExecuteActivity(ctx, "UpdateAllocationSpentActivity", *oldEnvelopeID, targetDate, -existingTxn.AmountE5).Get(ctx, nil)
			if err != nil {
				return err
			}
		}

		// Step 3: Add spent amount to new envelope allocation if new type is debit
		if newEnvelopeID != nil && txnType == "debit" {
			err = workflow.ExecuteActivity(ctx, "UpdateAllocationSpentActivity", *newEnvelopeID, targetDate, amountE5).Get(ctx, nil)
			if err != nil {
				return err
			}
		}
	} else if newEnvelopeID != nil && txnType == "debit" {
		// Category remained identical, but amount might have changed
		delta := amountE5 - existingTxn.AmountE5
		if delta != 0 {
			err = workflow.ExecuteActivity(ctx, "UpdateAllocationSpentActivity", *newEnvelopeID, targetDate, delta).Get(ctx, nil)
			if err != nil {
				return err
			}
		}
	}

	// Step 4: Update the transaction record with the new category and details
	updatedTxn := existingTxn
	updatedTxn.EnvelopeID = newEnvelopeID
	if req.AmountE5 != 0 {
		updatedTxn.AmountE5 = req.AmountE5
	}
	if req.TxnType != "" {
		updatedTxn.Type = req.TxnType
	}
	if req.PaymentMethod != "" {
		updatedTxn.PaymentMethod = req.PaymentMethod
	}

	err = workflow.ExecuteActivity(ctx, "UpdateTransactionActivity", updatedTxn).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
