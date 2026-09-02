package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/cmd/server/handlers"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

type dummyUserRepo struct{}

func (d *dummyUserRepo) CreateUser(u *core.User, Tx *sql.Tx) (uuid.UUID, error) {
	return validTokenUUID, nil
}
func (d *dummyUserRepo) GetUserByUUID(id uuid.UUID) (*core.User, error) { return nil, nil }
func (d *dummyUserRepo) GetUserByEmail(email string) (*core.User, error) {
	return nil, nil
}

type dummyTokenRepo struct{}

var validTokenUUID = uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
var expiredTokenUUID = uuid.MustParse("87654321-e89b-12d3-a456-426614174000")

func (d *dummyTokenRepo) CreateToken(t *core.Token, Tx *sql.Tx) (uuid.UUID, error) {
	return validTokenUUID, nil
}
func (d *dummyTokenRepo) DeleteToken(userUUID uuid.UUID) error { return nil }
func (d *dummyTokenRepo) GetToken(token uuid.UUID) (*core.Token, error) {
	if token == uuid.Nil {
		return nil, errors.New("invalid token")
	}
	if token == expiredTokenUUID {
		past := time.Now().Add(-1 * time.Hour)
		return &core.Token{UserUUID: validTokenUUID, ExpiresAt: &past}, nil
	}
	return &core.Token{UserUUID: validTokenUUID}, nil
}
func (d *dummyTokenRepo) GetActiveTokenWithUserUUID(userUUID uuid.UUID) (*core.Token, error) {
	return nil, nil
}
func (d *dummyTokenRepo) UpdateToken(t *core.Token) error { return nil }

type dummyTxnRepo struct{}

func (d *dummyTxnRepo) CreateTransaction(t *core.Transaction, Tx *sql.Tx) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (d *dummyTxnRepo) GetTransactionByUUID(id uuid.UUID) (*core.Transaction, error) { return nil, nil }
func (d *dummyTxnRepo) GetTransactionsByUserUUID(id uuid.UUID) ([]*core.Transaction, error) {
	return nil, nil
}
func (d *dummyTxnRepo) UpdateTransaction(t *core.Transaction, Tx *sql.Tx) error { return nil }
func (d *dummyTxnRepo) DeleteTransaction(id uuid.UUID) error                     { return nil }
func (d *dummyTxnRepo) GetTransactionByTime(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
	return nil, nil
}
func (d *dummyTxnRepo) GetTransactionByAmountAndTime(userUUID uuid.UUID, amountE5 int64, time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
	return &core.Transaction{ID: uuid.New(), UserID: userUUID, AmountE5: amountE5}, nil
}
func (d *dummyTxnRepo) GetDashboardSummary(uuid uuid.UUID) (*core.DashboardSummary, error) {
	return &core.DashboardSummary{}, nil
}
func (d *dummyTxnRepo) GetTransactionByUserUUIDPaginated(userUUID uuid.UUID, lastTransactionCreatedAt time.Time, lastTransactionID uuid.UUID, limit int) ([]*core.Transaction, error) {
	return nil, nil
}

type dummyShortcutIntentRepo struct{}

func (d *dummyShortcutIntentRepo) CreateShortcutIntent(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (d *dummyShortcutIntentRepo) GetShortcutIntentByID(id uuid.UUID) (*core.ShortcutIntent, error) {
	return nil, nil
}
func (d *dummyShortcutIntentRepo) GetShortcutIntentsByUserUUID(userUUID uuid.UUID) ([]*core.ShortcutIntent, error) {
	return nil, nil
}
func (d *dummyShortcutIntentRepo) UpdateShortcutIntent(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
	return nil
}
func (d *dummyShortcutIntentRepo) DeleteShortcutIntent(id uuid.UUID) error { return nil }
func (d *dummyShortcutIntentRepo) GetPendingRecentShortcutIntent(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*core.ShortcutIntent, error) {
	return nil, nil
}

type dummyEnvelopeGroupRepo struct{}

func (d *dummyEnvelopeGroupRepo) CreateEnvelopeGroup(g *core.EnvelopeGroup, Tx *sql.Tx) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (d *dummyEnvelopeGroupRepo) GetEnvelopeGroupByID(id uuid.UUID) (*core.EnvelopeGroup, error) {
	return nil, nil
}
func (d *dummyEnvelopeGroupRepo) GetEnvelopeGroupsByUserUUID(id uuid.UUID) ([]*core.EnvelopeGroup, error) {
	return nil, nil
}
func (d *dummyEnvelopeGroupRepo) UpdateEnvelopeGroup(g *core.EnvelopeGroup) error { return nil }
func (d *dummyEnvelopeGroupRepo) DeleteEnvelopeGroup(id uuid.UUID) error          { return nil }

type dummyEnvelopeRepo struct{}

func (d *dummyEnvelopeRepo) CreateEnvelope(e *core.Envelope, Tx *sql.Tx) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (d *dummyEnvelopeRepo) GetEnvelopeByID(id uuid.UUID) (*core.Envelope, error) {
	return nil, nil
}
func (d *dummyEnvelopeRepo) GetEnvelopesByUserUUID(id uuid.UUID) ([]*core.Envelope, error) {
	return nil, nil
}
func (d *dummyEnvelopeRepo) UpdateEnvelope(e *core.Envelope) error { return nil }
func (d *dummyEnvelopeRepo) DeleteEnvelope(id uuid.UUID) error     { return nil }
func (d *dummyEnvelopeRepo) GetEnvelopeIdByName(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
	return uuid.Nil, nil
}

type dummyAllocationRepo struct{}

func (d *dummyAllocationRepo) CreateAllocation(a *core.Allocation, Tx *sql.Tx) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (d *dummyAllocationRepo) GetAllocationByID(id uuid.UUID) (*core.Allocation, error) {
	return nil, nil
}
func (d *dummyAllocationRepo) GetAllocationsByEnvelopeID(id uuid.UUID) ([]*core.Allocation, error) {
	return nil, nil
}
func (d *dummyAllocationRepo) GetActiveAllocationsByUserUUID(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error) {
	return nil, nil
}
func (d *dummyAllocationRepo) UpdateAllocation(a *core.Allocation) error { return nil }
func (d *dummyAllocationRepo) DeleteAllocation(id uuid.UUID) error       { return nil }

func TestServer(t *testing.T) {
	log := zap.NewNop()
	tokenRepo := &dummyTokenRepo{}
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	userHandler := handlers.NewUserServiceHandler(&dummyUserRepo{}, tokenRepo, log, &dummyEnvelopeGroupRepo{}, &dummyEnvelopeRepo{}, &dummyAllocationRepo{}, mockDB)
	txnHandler := handlers.NewTransactionServiceHandler(&dummyTxnRepo{}, &dummyShortcutIntentRepo{}, log, mockDB, nil, core.RepoContainer{})
	budgetingHandler := handlers.NewBudgetingServiceHandler(&dummyEnvelopeGroupRepo{}, &dummyEnvelopeRepo{}, &dummyAllocationRepo{}, &dummyTxnRepo{}, &dummyShortcutIntentRepo{}, log, mockDB, nil, core.RepoContainer{})
	authHandler := handlers.NewAuthServiceHandler(&dummyUserRepo{}, tokenRepo, &dummyEnvelopeGroupRepo{}, &dummyEnvelopeRepo{}, &dummyAllocationRepo{}, log, mockDB)
	shortcutIntentRepo := &dummyShortcutIntentRepo{}

	t.Run("NewMux", func(t *testing.T) {
		m := NewMux()
		if m == nil {
			t.Fatal("expected non-nil router")
		}
	})

	t.Run("NewApplication", func(t *testing.T) {
		app := NewApplication(txnHandler, userHandler, budgetingHandler, tokenRepo, authHandler, shortcutIntentRepo)
		if app == nil || app.userHandler != userHandler || app.transactionHandler != txnHandler || app.budgetingHandler != budgetingHandler || app.tokenRepo != tokenRepo || app.authHandler != authHandler || app.shortcutIntentRepo != shortcutIntentRepo {
			t.Fatal("expected application initialized with handlers and tokenRepo")
		}
	})

	t.Run("RegisterRoutes & Health Check", func(t *testing.T) {
		router := NewMux()
		app := NewApplication(txnHandler, userHandler, budgetingHandler, tokenRepo, authHandler, shortcutIntentRepo)
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

		validAuthHeader := "Bearer " + validTokenUUID.String()

		// Test user endpoints
		mock.ExpectBegin()
		mock.ExpectCommit()
		reqUserPost := httptest.NewRequest("POST", "/user", io.NopCloser(bytes.NewReader([]byte(`{"name":"Alice"}`))))
		rrUserPost := httptest.NewRecorder()
		router.ServeHTTP(rrUserPost, reqUserPost)
		if rrUserPost.Code != http.StatusCreated {
			t.Errorf("expected status 201 for POST /user route, got %d", rrUserPost.Code)
		}

		reqUserGet := httptest.NewRequest("GET", "/user?user_uuid="+validTokenUUID.String(), nil)
		reqUserGet.Header.Set("Authorization", validAuthHeader)
		rrUserGet := httptest.NewRecorder()
		router.ServeHTTP(rrUserGet, reqUserGet)
		if rrUserGet.Code != http.StatusOK {
			t.Errorf("expected status 200 for GET /user route, got %d", rrUserGet.Code)
		}

		// Test transaction endpoints
		mock.ExpectBegin()
		mock.ExpectCommit()
		reqTxnPost := httptest.NewRequest("POST", "/transaction", io.NopCloser(bytes.NewReader([]byte(`{"amount_e5":100}`))))
		reqTxnPost.Header.Set("Authorization", validAuthHeader)
		rrTxnPost := httptest.NewRecorder()
		router.ServeHTTP(rrTxnPost, reqTxnPost)
		if rrTxnPost.Code != http.StatusCreated {
			t.Errorf("expected status 201 for POST /transaction route, got %d", rrTxnPost.Code)
		}

		reqTxnGet := httptest.NewRequest("GET", "/transaction?txn_uuid="+validTokenUUID.String(), nil)
		reqTxnGet.Header.Set("Authorization", validAuthHeader)
		rrTxnGet := httptest.NewRecorder()
		router.ServeHTTP(rrTxnGet, reqTxnGet)
		if rrTxnGet.Code != http.StatusOK {
			t.Errorf("expected status 200 for GET /transaction route, got %d", rrTxnGet.Code)
		}

		reqTxnsGet := httptest.NewRequest("GET", "/transactions?user_uuid="+validTokenUUID.String(), nil)
		reqTxnsGet.Header.Set("Authorization", validAuthHeader)
		rrTxnsGet := httptest.NewRecorder()
		router.ServeHTTP(rrTxnsGet, reqTxnsGet)
		if rrTxnsGet.Code != http.StatusOK {
			t.Errorf("expected status 200 for GET /transactions route, got %d", rrTxnsGet.Code)
		}

		reqTxnPut := httptest.NewRequest("PUT", "/transaction", io.NopCloser(bytes.NewReader([]byte(`{"uuid":"`+validTokenUUID.String()+`"}`))))
		reqTxnPut.Header.Set("Authorization", validAuthHeader)
		rrTxnPut := httptest.NewRecorder()
		router.ServeHTTP(rrTxnPut, reqTxnPut)
		if rrTxnPut.Code != http.StatusOK {
			t.Errorf("expected status 200 for PUT /transaction route, got %d", rrTxnPut.Code)
		}

		reqTxnDelete := httptest.NewRequest("DELETE", "/transaction?uuid="+validTokenUUID.String(), nil)
		reqTxnDelete.Header.Set("Authorization", validAuthHeader)
		rrTxnDelete := httptest.NewRecorder()
		router.ServeHTTP(rrTxnDelete, reqTxnDelete)
		if rrTxnDelete.Code != http.StatusOK {
			t.Errorf("expected status 200 for DELETE /transaction route, got %d", rrTxnDelete.Code)
		}

		reqTxnTransfer := httptest.NewRequest("POST", "/transaction/transfer", io.NopCloser(bytes.NewReader([]byte(`{"amount_e5":500}`))))
		reqTxnTransfer.Header.Set("Authorization", validAuthHeader)
		rrTxnTransfer := httptest.NewRecorder()
		router.ServeHTTP(rrTxnTransfer, reqTxnTransfer)
		if rrTxnTransfer.Code != http.StatusOK {
			t.Errorf("expected status 200 for POST /transaction/transfer route, got %d", rrTxnTransfer.Code)
		}
	})

	t.Run("AuthMiddleware", func(t *testing.T) {
		middleware := AuthMiddleware(tokenRepo)
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userUUID, _ := r.Context().Value("user_uuid").(uuid.UUID)
			w.Write([]byte(userUUID.String()))
		})

		handler := middleware(nextHandler)

		t.Run("Health Check Bypass", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200 for health check bypass, got %d", rr.Code)
			}
		})

		t.Run("Create User Bypass", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/user", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200 for create user bypass, got %d", rr.Code)
			}
		})

		t.Run("Missing Authorization Header", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", rr.Code)
			}
		})

		t.Run("Invalid Token Format", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer not-a-valid-uuid")
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", rr.Code)
			}
		})

		t.Run("Token Not Found", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+uuid.Nil.String())
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", rr.Code)
			}
		})

		t.Run("Expired Token", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+expiredTokenUUID.String())
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", rr.Code)
			}
		})

		t.Run("Valid Token", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+validTokenUUID.String())
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}
			body, _ := io.ReadAll(rr.Body)
			if string(body) != validTokenUUID.String() {
				t.Errorf("expected body '%s', got '%s'", validTokenUUID.String(), string(body))
			}
		})
	})

	t.Run("CORSMiddleware Preflight", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		handler := CORSMiddleware(nextHandler)

		// Test with Origin header and OPTIONS
		req1 := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req1.Header.Set("Origin", "https://penne-dashboard.netlify.app")
		rr1 := httptest.NewRecorder()
		handler.ServeHTTP(rr1, req1)

		if rr1.Code != http.StatusOK {
			t.Errorf("expected status 200 for OPTIONS preflight, got %d", rr1.Code)
		}
		if origin := rr1.Header().Get("Access-Control-Allow-Origin"); origin != "https://penne-dashboard.netlify.app" {
			t.Errorf("expected Access-Control-Allow-Origin header 'https://penne-dashboard.netlify.app', got '%s'", origin)
		}

		// Test without Origin header and GET
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)

		if rr2.Code != http.StatusOK {
			t.Errorf("expected status 200 for GET request, got %d", rr2.Code)
		}
		if origin := rr2.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("expected Access-Control-Allow-Origin header '*', got '%s'", origin)
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
