package server

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/mark3labs/mcp-go/mcp"
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

func TestToolHandlers(t *testing.T) {
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

	getHandler := handleGetTransactions(logger, txnRepo)
	createHandler := handleCreateTransaction(logger, txnRepo)

	t.Run("GetTransactions - Missing user_uuid", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{},
			},
		}
		res, err := getHandler(context.Background(), req)
		if err != nil || res == nil || !res.IsError {
			t.Errorf("expected tool error result for missing user_uuid, got %+v", res)
		}
	})

	t.Run("GetTransactions - Repo Error", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{"user_uuid": "error-user"},
			},
		}
		res, err := getHandler(context.Background(), req)
		if err != nil || res == nil || !res.IsError {
			t.Errorf("expected tool error result for repo error, got %+v", res)
		}
	})

	t.Run("GetTransactions - Success", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{"user_uuid": "user-123"},
			},
		}
		res, err := getHandler(context.Background(), req)
		if err != nil || res == nil || res.IsError {
			t.Errorf("expected success result, got %+v", res)
		}
	})

	t.Run("CreateTransaction - Missing Fields", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{"user_uuid": "user-123"},
			},
		}
		res, err := createHandler(context.Background(), req)
		if err != nil || res == nil || !res.IsError {
			t.Errorf("expected tool error result for missing fields, got %+v", res)
		}
	})

	t.Run("CreateTransaction - Repo Error", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"user_uuid":    "error-user",
					"amount_e5":    100.0,
					"country_iso2": "US",
					"category":     "Food",
					"bank_name":    "Chase",
					"txn_type":     "debit",
				},
			},
		}
		res, err := createHandler(context.Background(), req)
		if err != nil || res == nil || !res.IsError {
			t.Errorf("expected tool error result for repo error, got %+v", res)
		}
	})

	t.Run("CreateTransaction - Success (float64, int, int64)", func(t *testing.T) {
		types := []any{float64(100), int(100), int64(100)}
		for _, amount := range types {
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]any{
						"user_uuid":    "user-123",
						"amount_e5":    amount,
						"country_iso2": "US",
						"category":     "Food",
						"bank_name":    "Chase",
						"txn_type":     "debit",
					},
				},
			}
			res, err := createHandler(context.Background(), req)
			if err != nil || res == nil || res.IsError {
				t.Errorf("expected success result for amount type %T, got %+v", amount, res)
			}
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
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler := MCPAuthMiddleWare(nextHandler)

	t.Run("OPTIONS Request CORS Preflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/sse", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for OPTIONS, got %d", rr.Code)
		}
		if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("expected CORS header, got %s", rr.Header().Get("Access-Control-Allow-Origin"))
		}
	})

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

	t.Run("Invalid Token Value", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("Authorization", "Bearer invalid_token_123")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("Valid Token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("Authorization", "Bearer penne_mcp_test_token_123")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})
}

func TestOAuthHandlers(t *testing.T) {
	t.Run("OAuthAuthorizeHandler - Missing redirect_uri", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/oauth/authorize", nil)
		rr := httptest.NewRecorder()
		OAuthAuthorizeHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("OAuthAuthorizeHandler - Success Redirect", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/oauth/authorize?redirect_uri=https://gemini.google.com/callback&state=xyz123", nil)
		rr := httptest.NewRecorder()
		OAuthAuthorizeHandler(rr, req)

		if rr.Code != http.StatusFound {
			t.Errorf("expected 302 redirect, got %d", rr.Code)
		}
		location := rr.Header().Get("Location")
		if location != "https://gemini.google.com/callback?code=fake_auth_code_999&state=xyz123" {
			t.Errorf("unexpected redirect URL: %s", location)
		}
	})

	t.Run("OAuthTokenHandler - OPTIONS Preflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/oauth/token", nil)
		rr := httptest.NewRecorder()
		OAuthTokenHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for OPTIONS, got %d", rr.Code)
		}
	})

	t.Run("OAuthTokenHandler - Success", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/oauth/token", nil)
		rr := httptest.NewRecorder()
		OAuthTokenHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		if rr.Header().Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", rr.Header().Get("Content-Type"))
		}
	})
}

