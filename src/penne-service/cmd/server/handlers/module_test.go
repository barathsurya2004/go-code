package handlers

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/cadence/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func TestModule(t *testing.T) {
	var txnHandler *TransactionServiceHandler
	var userHandler *UserServiceHandler
	var budgetingHandler *BudgetingServiceHandler
	var authHandler *AuthServiceHandler

	app := fx.New(
		Module,
		fx.Provide(
			func() core.TransactionRepository { return &mockTxnRepo{} },
			func() core.UserRepository { return &mockUserRepo{} },
			func() core.TokenRepository { return &mockTokenRepo{} },
			func() core.EnvelopeGroupRepository { return &mockEnvelopeGroupRepo{} },
			func() core.EnvelopeRepository { return &mockEnvelopeRepo{} },
			func() core.AllocationRepository { return &mockAllocationRepo{} },
			func() core.ShortcutIntentRepository { return &mockShortcutIntentRepo{} },
			func() client.Client { return nil },
			func() core.RepoContainer { return core.RepoContainer{} },
			func() *sql.DB {
				db, _, _ := sqlmock.New()
				return db
			},
			zap.NewNop,
		),
		fx.Populate(&txnHandler, &userHandler, &budgetingHandler, &authHandler),
	)

	if err := app.Err(); err != nil {
		t.Fatalf("expected no error initializing handlers.Module, got %v", err)
	}

	if txnHandler == nil || userHandler == nil || budgetingHandler == nil || authHandler == nil {
		t.Fatal("expected populated handlers, got nil")
	}
}
