package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/cmd/server/handlers"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

type dummyUserRepo struct{}

func (d *dummyUserRepo) CreateUser(u *core.User) error               { return nil }
func (d *dummyUserRepo) GetUserByUUID(id string) (*core.User, error) { return nil, nil }

type dummyTokenRepo struct{}

func (d *dummyTokenRepo) CreateToken(t *core.Token) error                           { return nil }
func (d *dummyTokenRepo) DeleteToken(userUUID string) error                        { return nil }
func (d *dummyTokenRepo) GetToken(token string) (*core.Token, error)                { return nil, nil }
func (d *dummyTokenRepo) GetTokenWithUserUUID(userUUID string) (*core.Token, error) { return nil, nil }
func (d *dummyTokenRepo) UpdateToken(t *core.Token) error                           { return nil }

type dummyTxnRepo struct{}

func (d *dummyTxnRepo) CreateTransaction(t *core.Transaction) error                   { return nil }
func (d *dummyTxnRepo) GetTransactionByUUID(id string) (*core.Transaction, error)      { return nil, nil }
func (d *dummyTxnRepo) GetTransactionsByUserUUID(id string) ([]*core.Transaction, error) { return nil, nil }
func (d *dummyTxnRepo) UpdateTransaction(t *core.Transaction) error                   { return nil }
func (d *dummyTxnRepo) DeleteTransaction(id string) error                            { return nil }

func TestServer(t *testing.T) {
	log := zap.NewNop()
	tokenRepo := &dummyTokenRepo{}
	userHandler := handlers.NewUserServiceHandler(&dummyUserRepo{}, tokenRepo, log)
	txnHandler := handlers.NewTransactionServiceHandler(&dummyTxnRepo{}, log)

	t.Run("NewMux", func(t *testing.T) {
		m := NewMux()
		if m == nil {
			t.Fatal("expected non-nil router")
		}
	})

	t.Run("NewApplication", func(t *testing.T) {
		app := NewApplication(txnHandler, userHandler, nil, tokenRepo)
		if app == nil || app.userHandler != userHandler || app.transactionHandler != txnHandler {
			t.Fatal("expected application initialized with handlers")
		}
	})

	t.Run("RegisterRoutes & Health Check", func(t *testing.T) {
		router := NewMux()
		app := NewApplication(txnHandler, userHandler, nil, tokenRepo)
		RegisterRoutes(router, log, app)

		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		body, _ := io.ReadAll(rr.Body)
		if string(body) != "OK" {
			t.Errorf("expected body 'OK', got '%s'", string(body))
		}

		// Test user and transaction route dispatching
		reqUser := httptest.NewRequest("GET", "/user?user_uuid=123e4567-e89b-12d3-a456-426614174000", nil)
		rrUser := httptest.NewRecorder()
		router.ServeHTTP(rrUser, reqUser)
		if rrUser.Code != http.StatusOK {
			t.Errorf("expected status 200 for user route, got %d", rrUser.Code)
		}

		reqTxn := httptest.NewRequest("GET", "/transaction?uuid=123e4567-e89b-12d3-a456-426614174000", nil)
		rrTxn := httptest.NewRecorder()
		router.ServeHTTP(rrTxn, reqTxn)
		if rrTxn.Code != http.StatusOK {
			t.Errorf("expected status 200 for transaction route, got %d", rrTxn.Code)
		}
	})

	t.Run("NewHTTPServer Lifecycle", func(t *testing.T) {
		lc := fxtest.NewLifecycle(t)
		router := NewMux()
		srv := NewHTTPServer(lc, router, log)
		if srv == nil {
			t.Fatal("expected non-nil http.Server")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := lc.Start(ctx); err != nil {
			t.Fatalf("failed to start server lifecycle: %v", err)
		}

		if err := lc.Stop(ctx); err != nil {
			t.Fatalf("failed to stop server lifecycle: %v", err)
		}
	})
}
