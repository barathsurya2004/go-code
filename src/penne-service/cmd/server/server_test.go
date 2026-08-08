package main

import (
	"bytes"
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
		app := NewApplication(txnHandler, userHandler, tokenRepo)
		if app == nil || app.userHandler != userHandler || app.transactionHandler != txnHandler || app.tokenRepo != tokenRepo {
			t.Fatal("expected application initialized with handlers and tokenRepo")
		}
	})

	t.Run("RegisterRoutes & Health Check", func(t *testing.T) {
		router := NewMux()
		app := NewApplication(txnHandler, userHandler, tokenRepo)
		RegisterRoutes(router, log, app)

		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		body, _ := io.ReadAll(rr.Body)
		if string(body) != "OK(deployment check)" {
			t.Errorf("expected body 'OK(deployment check)', got '%s'", string(body))
		}

		// Test user endpoints
		reqUserPost := httptest.NewRequest("POST", "/user", io.NopCloser(bytes.NewReader([]byte(`{"name":"Alice"}`))))
		rrUserPost := httptest.NewRecorder()
		router.ServeHTTP(rrUserPost, reqUserPost)
		if rrUserPost.Code != http.StatusCreated {
			t.Errorf("expected status 201 for POST /user route, got %d", rrUserPost.Code)
		}

		reqUserGet := httptest.NewRequest("GET", "/user?user_uuid=123e4567-e89b-12d3-a456-426614174000", nil)
		rrUserGet := httptest.NewRecorder()
		router.ServeHTTP(rrUserGet, reqUserGet)
		if rrUserGet.Code != http.StatusOK {
			t.Errorf("expected status 200 for GET /user route, got %d", rrUserGet.Code)
		}

		// Test transaction endpoints
		reqTxnPost := httptest.NewRequest("POST", "/transaction", io.NopCloser(bytes.NewReader([]byte(`{"amount_e5":100}`))))
		rrTxnPost := httptest.NewRecorder()
		router.ServeHTTP(rrTxnPost, reqTxnPost)
		if rrTxnPost.Code != http.StatusCreated {
			t.Errorf("expected status 201 for POST /transaction route, got %d", rrTxnPost.Code)
		}

		reqTxnGet := httptest.NewRequest("GET", "/transaction?uuid=123e4567-e89b-12d3-a456-426614174000", nil)
		rrTxnGet := httptest.NewRecorder()
		router.ServeHTTP(rrTxnGet, reqTxnGet)
		if rrTxnGet.Code != http.StatusOK {
			t.Errorf("expected status 200 for GET /transaction route, got %d", rrTxnGet.Code)
		}

		reqTxnsGet := httptest.NewRequest("GET", "/transactions?user_uuid=123e4567-e89b-12d3-a456-426614174000", nil)
		rrTxnsGet := httptest.NewRecorder()
		router.ServeHTTP(rrTxnsGet, reqTxnsGet)
		if rrTxnsGet.Code != http.StatusOK {
			t.Errorf("expected status 200 for GET /transactions route, got %d", rrTxnsGet.Code)
		}

		reqTxnPut := httptest.NewRequest("PUT", "/transaction", io.NopCloser(bytes.NewReader([]byte(`{"uuid":"123e4567-e89b-12d3-a456-426614174000"}`))))
		rrTxnPut := httptest.NewRecorder()
		router.ServeHTTP(rrTxnPut, reqTxnPut)
		if rrTxnPut.Code != http.StatusOK {
			t.Errorf("expected status 200 for PUT /transaction route, got %d", rrTxnPut.Code)
		}

		reqTxnDelete := httptest.NewRequest("DELETE", "/transaction?uuid=123e4567-e89b-12d3-a456-426614174000", nil)
		rrTxnDelete := httptest.NewRecorder()
		router.ServeHTTP(rrTxnDelete, reqTxnDelete)
		if rrTxnDelete.Code != http.StatusOK {
			t.Errorf("expected status 200 for DELETE /transaction route, got %d", rrTxnDelete.Code)
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
