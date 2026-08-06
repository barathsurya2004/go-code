package server

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type mockTxnRepo struct {
	createTransactionFn         func(txn *core.Transaction) error
	getTransactionByUUIDFn      func(uuid string) (*core.Transaction, error)
	getTransactionsByUserUUIDFn func(userUUID string) ([]*core.Transaction, error)
	updateTransactionFn         func(txn *core.Transaction) error
	deleteTransactionFn         func(uuid string) error
}

func (m *mockTxnRepo) CreateTransaction(txn *core.Transaction) error {
	if m.createTransactionFn != nil {
		return m.createTransactionFn(txn)
	}
	return nil
}

func (m *mockTxnRepo) GetTransactionByUUID(uuid string) (*core.Transaction, error) {
	if m.getTransactionByUUIDFn != nil {
		return m.getTransactionByUUIDFn(uuid)
	}
	return nil, nil
}

func (m *mockTxnRepo) GetTransactionsByUserUUID(userUUID string) ([]*core.Transaction, error) {
	if m.getTransactionsByUserUUIDFn != nil {
		return m.getTransactionsByUserUUIDFn(userUUID)
	}
	return nil, nil
}

func (m *mockTxnRepo) UpdateTransaction(txn *core.Transaction) error {
	if m.updateTransactionFn != nil {
		return m.updateTransactionFn(txn)
	}
	return nil
}

func (m *mockTxnRepo) DeleteTransaction(uuid string) error {
	if m.deleteTransactionFn != nil {
		return m.deleteTransactionFn(uuid)
	}
	return nil
}

type mockTokenRepo struct {
	getTokenFn func(token string) (*core.Token, error)
}

func (m *mockTokenRepo) CreateToken(token *core.Token) error { return nil }
func (m *mockTokenRepo) DeleteToken(userUUID string) error  { return nil }
func (m *mockTokenRepo) GetToken(token string) (*core.Token, error) {
	if m.getTokenFn != nil {
		return m.getTokenFn(token)
	}
	return nil, nil
}
func (m *mockTokenRepo) GetTokenWithUserUUID(userUUID string) (*core.Token, error) {
	return nil, nil
}
func (m *mockTokenRepo) UpdateToken(token *core.Token) error { return nil }

func TestNewMCPServer(t *testing.T) {
	logger := zap.NewNop()
	txnRepo := &mockTxnRepo{
		getTransactionsByUserUUIDFn: func(userUUID string) ([]*core.Transaction, error) {
			if userUUID == "error-user" {
				return nil, errors.New("db error")
			}
			return []*core.Transaction{{UUID: "txn-1", UserUUID: userUUID}}, nil
		},
		createTransactionFn: func(txn *core.Transaction) error {
			if txn.UserUUID == "error-user" {
				return errors.New("db error")
			}
			return nil
		},
	}

	srv := NewMCPServer(logger, txnRepo)
	if srv == nil {
		t.Fatal("expected non-nil SSEServer")
	}

	t.Run("ServeHTTP SSE request with canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel to prevent SSE streaming loop from blocking
		req := httptest.NewRequest("GET", "/sse", nil).WithContext(ctx)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for SSE, got %d", rr.Code)
		}
	})

	t.Run("ServeHTTP Message request - empty body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/message?sessionId=test", bytes.NewBufferString("{}"))
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		if rr.Code == 0 {
			t.Error("expected non-zero response code")
		}
	})
}

func TestMCPModule(t *testing.T) {
	app := fx.New(
		Module,
		fx.Provide(
			zap.NewNop,
			func() core.TransactionRepository { return &mockTxnRepo{} },
		),
	)
	if err := app.Err(); err != nil {
		t.Fatalf("expected Module to initialize without error, got %v", err)
	}
}

func TestMCPAuthMiddleWare(t *testing.T) {
	tokenRepo := &mockTokenRepo{}
	middleware := MCPAuthMiddleWare(tokenRepo)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler := middleware(nextHandler)

	t.Run("Missing Authorization Header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sse", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("Invalid Authorization Format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("Authorization", "Basic xyz")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("Missing Bearer Token Value", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("Authorization", "Bearer ")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("Invalid Token Prefix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("Authorization", "Bearer invalid_prefix_123")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("Token Repo Error", func(t *testing.T) {
		tokenRepo.getTokenFn = func(token string) (*core.Token, error) {
			return nil, errors.New("not found")
		}
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("Authorization", "Bearer mcp_testtoken")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("Expired Token", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		tokenRepo.getTokenFn = func(token string) (*core.Token, error) {
			return &core.Token{ExpiresAt: &past}, nil
		}
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("Authorization", "Bearer mcp_testtoken")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("Valid Token", func(t *testing.T) {
		future := time.Now().Add(1 * time.Hour)
		tokenRepo.getTokenFn = func(token string) (*core.Token, error) {
			scopes := []string{"read", "write"}
			return &core.Token{UserUUID: "user-123", ExpiresAt: &future, Scope: scopes}, nil
		}
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("Authorization", "Bearer mcp_testtoken")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})
}
