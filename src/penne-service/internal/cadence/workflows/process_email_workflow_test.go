package workflows

import (
	"errors"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/activities"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/emailparser"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
)

func (s *UnitTestSuite) Test_ProcessEmailWorkflow_BankAccountDebit_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	emailActs := activities.NewEmailActivities(logger)
	txnActs := activities.NewTransactionActivities(core.RepoContainer{}, logger)

	env.RegisterActivityWithOptions(emailActs.ParseEmailActivity, activity.RegisterOptions{Name: "ParseEmailActivity"})
	env.RegisterActivityWithOptions(txnActs.PendingShortcutIntentActivity, activity.RegisterOptions{Name: "PendingShortcutIntentActivity"})
	env.RegisterActivityWithOptions(txnActs.CreateTransaction, activity.RegisterOptions{Name: "CreateTransactionActivity"})
	env.RegisterWorkflow(CreateTransactionWorkflow)

	expectedTxnID := uuid.New()
	env.OnActivity("PendingShortcutIntentActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*core.ShortcutIntent)(nil), nil)
	env.OnActivity("CreateTransactionActivity", mock.Anything, mock.MatchedBy(func(txn core.Transaction) bool {
		return txn.AmountE5 == 1000000 && txn.Type == core.TxnTypeDebit && txn.PaymentMethod == "bank_account" && txn.CountryISO == "IN"
	})).Return(&expectedTxnID, nil)

	userID := uuid.New()
	testEmailDate := time.Date(2026, 9, 22, 10, 30, 0, 0, time.UTC)
	input := EmailTransactionInput{
		UserID:  userID,
		Subject: "IDFC FIRST Bank Debit Alert",
		Body: `Dear Mr. Barath Surya M,
Greetings from IDFC FIRST Bank.
Your A/C XXXXXXX2559 has been debited by INR 10.00 on 22/09/2026 16:08. New balance is INR 33,918.42CR.
Always You First,
Team IDFC FIRST Bank`,
		EmailDate: testEmailDate,
	}

	env.ExecuteWorkflow(ProcessEmailWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result ProcessEmailWorkflowResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal(expectedTxnID, result.TransactionID)
	s.Equal(int64(1000000), result.ParsedDetails.AmountE5)
	s.Equal("XXXXXXX2559", result.ParsedDetails.AccountOrCardNumber)
	s.Equal(emailparser.VariantBankAccount, result.ParsedDetails.Variant)
}

func (s *UnitTestSuite) Test_ProcessEmailWorkflow_CreditCardSpent_Success() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	emailActs := activities.NewEmailActivities(logger)
	txnActs := activities.NewTransactionActivities(core.RepoContainer{}, logger)

	env.RegisterActivityWithOptions(emailActs.ParseEmailActivity, activity.RegisterOptions{Name: "ParseEmailActivity"})
	env.RegisterActivityWithOptions(txnActs.PendingShortcutIntentActivity, activity.RegisterOptions{Name: "PendingShortcutIntentActivity"})
	env.RegisterActivityWithOptions(txnActs.CreateTransaction, activity.RegisterOptions{Name: "CreateTransactionActivity"})
	env.RegisterWorkflow(CreateTransactionWorkflow)

	expectedTxnID := uuid.New()
	env.OnActivity("PendingShortcutIntentActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*core.ShortcutIntent)(nil), nil)
	env.OnActivity("CreateTransactionActivity", mock.Anything, mock.MatchedBy(func(txn core.Transaction) bool {
		return txn.AmountE5 == 117150000 && txn.Type == core.TxnTypeDebit && txn.PaymentMethod == "bank_card" && txn.CountryISO == "IN"
	})).Return(&expectedTxnID, nil)

	userID := uuid.New()
	testEmailDate := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	input := EmailTransactionInput{
		UserID:  userID,
		Subject: "IDFC FIRST Bank Credit Card Alert",
		Body: `Dear Cardmember,

All Stocked Up! INR 1171.50 spent on your IDFC FIRST BANK Credit Card ending XX1110 at AVENUE SUPERMARTS LI on 22 SEP 2026.

Available Limit: INR 38413.57 .`,
		EmailDate: testEmailDate,
	}

	env.ExecuteWorkflow(ProcessEmailWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result ProcessEmailWorkflowResult
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal(expectedTxnID, result.TransactionID)
	s.Equal(int64(117150000), result.ParsedDetails.AmountE5)
	s.Equal("XX1110", result.ParsedDetails.AccountOrCardNumber)
	s.Equal(emailparser.VariantCreditCard, result.ParsedDetails.Variant)
	s.Equal("AVENUE SUPERMARTS LI", result.ParsedDetails.Merchant)
}

func (s *UnitTestSuite) Test_ProcessEmailWorkflow_ParseError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	emailActs := activities.NewEmailActivities(logger)
	env.RegisterActivityWithOptions(emailActs.ParseEmailActivity, activity.RegisterOptions{Name: "ParseEmailActivity"})

	input := EmailTransactionInput{
		UserID:  uuid.New(),
		Subject: "Unrelated newsletter",
		Body:    "Thank you for signing up for our newsletter!",
	}

	env.ExecuteWorkflow(ProcessEmailWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}

func (s *UnitTestSuite) Test_ProcessEmailWorkflow_ChildWorkflowError() {
	env := s.NewTestWorkflowEnvironment()
	logger := zap.NewNop()

	emailActs := activities.NewEmailActivities(logger)
	txnActs := activities.NewTransactionActivities(core.RepoContainer{}, logger)

	env.RegisterActivityWithOptions(emailActs.ParseEmailActivity, activity.RegisterOptions{Name: "ParseEmailActivity"})
	env.RegisterActivityWithOptions(txnActs.PendingShortcutIntentActivity, activity.RegisterOptions{Name: "PendingShortcutIntentActivity"})
	env.RegisterActivityWithOptions(txnActs.CreateTransaction, activity.RegisterOptions{Name: "CreateTransactionActivity"})
	env.RegisterWorkflow(CreateTransactionWorkflow)

	env.OnActivity("PendingShortcutIntentActivity", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return((*core.ShortcutIntent)(nil), nil)
	env.OnActivity("CreateTransactionActivity", mock.Anything, mock.Anything).Return((*uuid.UUID)(nil), errors.New("db insert failed"))

	input := EmailTransactionInput{
		UserID:  uuid.New(),
		Subject: "IDFC Alert",
		Body: `Dear Mr. Barath Surya M,
Your A/C XXXXXXX2559 has been debited by INR 10.00 on 22/09/2026.`,
	}

	env.ExecuteWorkflow(ProcessEmailWorkflow, input)

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}
