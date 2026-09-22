package workflows

import (
	"errors"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/activities"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/emailparser"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/cadence/workflow"
)

// EmailTransactionInput is the input passed into ProcessEmailWorkflow.
type EmailTransactionInput struct {
	UserID    uuid.UUID `json:"user_id"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	EmailDate time.Time `json:"email_date"`
}

// ProcessEmailWorkflowResult is the result returned by ProcessEmailWorkflow.
type ProcessEmailWorkflowResult struct {
	TransactionID uuid.UUID                   `json:"transaction_id"`
	ParsedDetails emailparser.ParsedEmailResult `json:"parsed_details"`
}

// ProcessEmailWorkflow parses an IDFC FIRST Bank transaction email, extracts transaction details,
// and invokes CreateTransactionWorkflow to record the transaction and attribute any pending shortcut intents.
func ProcessEmailWorkflow(ctx workflow.Context, input EmailTransactionInput) (*ProcessEmailWorkflowResult, error) {
	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		ScheduleToStartTimeout: 2 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Parse the email body and subject
	var parsed emailparser.ParsedEmailResult
	err := workflow.ExecuteActivity(
		ctx,
		"ParseEmailActivity",
		activities.ParseEmailActivityInput{
			Subject: input.Subject,
			Body:    input.Body,
		},
	).Get(ctx, &parsed)
	if err != nil {
		return nil, err
	}

	// Step 2: Form the core.Transaction
	// User requirement: Use subject and date/time passed into the workflow (not the one in the email body)
	createdAt := input.EmailDate
	if createdAt.IsZero() {
		createdAt = utils.NowUTC()
	} else {
		createdAt = createdAt.UTC()
	}

	txn := core.Transaction{
		UserID:        input.UserID,
		AmountE5:      parsed.AmountE5,
		Type:          parsed.Type,
		PaymentMethod: parsed.PaymentMethod,
		CountryISO:    "IN",
		CreatedAt:     createdAt,
	}

	// Step 3: Execute child workflow CreateTransactionWorkflow
	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: 5 * time.Minute,
	}
	childCtx := workflow.WithChildOptions(ctx, cwo)

	var txnID *uuid.UUID
	err = workflow.ExecuteChildWorkflow(childCtx, CreateTransactionWorkflow, txn).Get(childCtx, &txnID)
	if err != nil {
		return nil, err
	}
	if txnID == nil {
		return nil, errors.New("child workflow returned nil transaction ID")
	}

	return &ProcessEmailWorkflowResult{
		TransactionID: *txnID,
		ParsedDetails: parsed,
	}, nil
}
