package handlers

import (
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func TestModule(t *testing.T) {
	var txnHandler *TransactionServiceHandler
	var userHandler *UserServiceHandler

	app := fx.New(
		Module,
		fx.Provide(
			func() core.TransactionRepository { return &mockTxnRepo{} },
			func() core.UserRepository { return &mockUserRepo{} },
			func() core.TokenRepository { return &mockTokenRepo{} },
			zap.NewNop,
		),
		fx.Populate(&txnHandler, &userHandler),
	)

	if err := app.Err(); err != nil {
		t.Fatalf("expected no error initializing handlers.Module, got %v", err)
	}

	if txnHandler == nil || userHandler == nil {
		t.Fatal("expected populated handlers, got nil")
	}
}
