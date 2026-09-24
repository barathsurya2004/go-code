package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
)

type mockWorkflowRun struct {
	client.WorkflowRun
	getFn func(ctx context.Context, valuePtr interface{}) error
}

func (m *mockWorkflowRun) Get(ctx context.Context, valuePtr interface{}) error {
	if m.getFn != nil {
		return m.getFn(ctx, valuePtr)
	}
	return nil
}
func (m *mockWorkflowRun) GetID() string    { return "test-id" }
func (m *mockWorkflowRun) GetRunID() string { return "test-run-id" }

type mockCadenceClient struct {
	client.Client
	executeWorkflowFn func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error)
}

func (m *mockCadenceClient) ExecuteWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
	if m.executeWorkflowFn != nil {
		return m.executeWorkflowFn(ctx, options, workflow, args...)
	}
	return &mockWorkflowRun{}, nil
}

type mockTxnRepo struct {
	createTransactionFn         func(txn *core.Transaction) (uuid.UUID, error)
	getTransactionByUUIDFn      func(id uuid.UUID) (*core.Transaction, error)
	getTransactionsByUserUUIDFn func(userUUID uuid.UUID) ([]*core.Transaction, error)
	updateTransactionFn         func(txn *core.Transaction) error
	deleteTransactionFn         func(id uuid.UUID) error
	getTransactionByTimeFn              func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error)
	getTransactionByAmountAndTimeFn     func(userUUID uuid.UUID, amountE5 int64, time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error)
	getDashboardSummaryFn               func(uuid uuid.UUID) (*core.DashboardSummary, error)
	getTransactionByUserUUIDPaginatedFn func(userUUID uuid.UUID, lastTransactionCreatedAt time.Time, lastTransactionID uuid.UUID, limit int) ([]*core.Transaction, error)
}

func (m *mockTxnRepo) CreateTransaction(txn *core.Transaction, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createTransactionFn != nil {
		return m.createTransactionFn(txn)
	}
	return uuid.Nil, nil
}

func (m *mockTxnRepo) GetTransactionByUUID(id uuid.UUID) (*core.Transaction, error) {
	if m.getTransactionByUUIDFn != nil {
		return m.getTransactionByUUIDFn(id)
	}
	return nil, nil
}

func (m *mockTxnRepo) GetTransactionsByUserUUID(userUUID uuid.UUID) ([]*core.Transaction, error) {
	if m.getTransactionsByUserUUIDFn != nil {
		return m.getTransactionsByUserUUIDFn(userUUID)
	}
	return nil, nil
}

func (m *mockTxnRepo) UpdateTransaction(txn *core.Transaction, Tx *sql.Tx) error {
	if m.updateTransactionFn != nil {
		return m.updateTransactionFn(txn)
	}
	return nil
}

func (m *mockTxnRepo) DeleteTransaction(id uuid.UUID) error {
	if m.deleteTransactionFn != nil {
		return m.deleteTransactionFn(id)
	}
	return nil
}

func (m *mockTxnRepo) GetTransactionByTime(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
	if m.getTransactionByTimeFn != nil {
		return m.getTransactionByTimeFn(time_lowerbound, time_upperbound, Tx)
	}
	return nil, nil
}

func (m *mockTxnRepo) GetTransactionByAmountAndTime(userUUID uuid.UUID, amountE5 int64, time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
	if m.getTransactionByAmountAndTimeFn != nil {
		return m.getTransactionByAmountAndTimeFn(userUUID, amountE5, time_lowerbound, time_upperbound, Tx)
	}
	return nil, nil
}

func (m *mockTxnRepo) GetDashboardSummary(userUUID uuid.UUID) (*core.DashboardSummary, error) {
	if m.getDashboardSummaryFn != nil {
		return m.getDashboardSummaryFn(userUUID)
	}
	return &core.DashboardSummary{}, nil
}

func (m *mockTxnRepo) GetTransactionByUserUUIDPaginated(userUUID uuid.UUID, lastTransactionCreatedAt time.Time, lastTransactionID uuid.UUID, limit int) ([]*core.Transaction, error) {
	if m.getTransactionByUserUUIDPaginatedFn != nil {
		return m.getTransactionByUserUUIDPaginatedFn(userUUID, lastTransactionCreatedAt, lastTransactionID, limit)
	}
	return nil, nil
}

func TestTransactionServiceHandler(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockTxnRepo{}
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	shortcutIntentRepo := &mockShortcutIntentRepo{}
	handler := NewTransactionServiceHandler(repo, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("CreateTransaction - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/transaction", bytes.NewBufferString("invalid json"))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.CreateTransaction(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateTransaction - Missing User UUID", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/transaction", bytes.NewBufferString(`{"amount_e5":100}`))
		rr := httptest.NewRecorder()

		handler.CreateTransaction(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateTransaction - Success without Cadence Client", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/transaction", bytes.NewBufferString(`{"amount_e5":100}`))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.CreateTransaction(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})

	t.Run("CreateTransaction - Success with Cadence Client", func(t *testing.T) {
		expectedUUID := uuid.New()
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						if ptr, ok := valuePtr.(**uuid.UUID); ok {
							*ptr = &expectedUUID
						}
						return nil
					},
				}, nil
			},
		}

		cadenceHandler := NewTransactionServiceHandler(repo, shortcutIntentRepo, logger, db, cc, core.RepoContainer{})
		req := httptest.NewRequest("POST", "/transaction", bytes.NewBufferString(`{"amount_e5":100}`))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		cadenceHandler.CreateTransaction(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})

	t.Run("CreateTransaction - Success with Cadence Client returning nil resultUUID", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						return nil
					},
				}, nil
			},
		}

		cadenceHandler := NewTransactionServiceHandler(repo, shortcutIntentRepo, logger, db, cc, core.RepoContainer{})
		req := httptest.NewRequest("POST", "/transaction", bytes.NewBufferString(`{"amount_e5":100}`))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		cadenceHandler.CreateTransaction(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})

	t.Run("CreateTransaction - ExecuteWorkflow Error", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return nil, errors.New("execute workflow error")
			},
		}

		cadenceHandler := NewTransactionServiceHandler(repo, shortcutIntentRepo, logger, db, cc, core.RepoContainer{})
		req := httptest.NewRequest("POST", "/transaction", bytes.NewBufferString(`{"amount_e5":100}`))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		cadenceHandler.CreateTransaction(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateTransaction - WorkflowRun Get Error", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						return errors.New("workflow get error")
					},
				}, nil
			},
		}

		cadenceHandler := NewTransactionServiceHandler(repo, shortcutIntentRepo, logger, db, cc, core.RepoContainer{})
		req := httptest.NewRequest("POST", "/transaction", bytes.NewBufferString(`{"amount_e5":100}`))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		cadenceHandler.CreateTransaction(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetTransactionByUUID - Missing or Invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/transaction?txn_uuid=invalid", nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionByUUID(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetTransactionByUUID - Repo Error", func(t *testing.T) {
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return nil, errors.New("not found")
		}
		req := httptest.NewRequest("GET", "/transaction?txn_uuid="+validUUID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionByUUID(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("GetTransactionByUUID - Success", func(t *testing.T) {
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return &core.Transaction{ID: id, AmountE5: 500}, nil
		}
		req := httptest.NewRequest("GET", "/transaction?txn_uuid="+validUUID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionByUUID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetTransactionsByUserUUID - Missing User UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/transactions", nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionsByUserUUID(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetTransactionsByUserUUID - Invalid User UUID in Query", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/transactions?user_uuid=invalid", nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionsByUserUUID(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetTransactionsByUserUUID - Repo Error", func(t *testing.T) {
		repo.getTransactionByUserUUIDPaginatedFn = func(userUUID uuid.UUID, lastTransactionCreatedAt time.Time, lastTransactionID uuid.UUID, limit int) ([]*core.Transaction, error) {
			return nil, errors.New("db error")
		}
		req := httptest.NewRequest("GET", "/transactions?user_uuid="+validUUID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionsByUserUUID(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetTransactionsByUserUUID - Success", func(t *testing.T) {
		repo.getTransactionByUserUUIDPaginatedFn = func(userUUID uuid.UUID, lastTransactionCreatedAt time.Time, lastTransactionID uuid.UUID, limit int) ([]*core.Transaction, error) {
			return []*core.Transaction{{ID: uuid.New(), UserID: userUUID}}, nil
		}
		req := httptest.NewRequest("GET", "/transactions?user_uuid="+validUUID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionsByUserUUID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetTransactionsByUserUUID - Success with Context, Limit, RFC3339 Timestamp, and LastID", func(t *testing.T) {
		lastID := uuid.New()
		repo.getTransactionByUserUUIDPaginatedFn = func(userUUID uuid.UUID, lastTransactionCreatedAt time.Time, lastTransactionID uuid.UUID, limit int) ([]*core.Transaction, error) {
			if limit != 15 || lastTransactionID != lastID {
				return nil, errors.New("unexpected params")
			}
			return []*core.Transaction{{ID: uuid.New(), UserID: userUUID}}, nil
		}
		req := httptest.NewRequest("GET", fmt.Sprintf("/transactions?limit=15&lastTransactionCreatedAt=2026-01-01T15:04:05Z&lastTransactionID=%s", lastID.String()), nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.GetTransactionsByUserUUID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetTransactionsByUserUUID - Success with Fallback Timestamp Format and Invalid Limit", func(t *testing.T) {
		repo.getTransactionByUserUUIDPaginatedFn = func(userUUID uuid.UUID, lastTransactionCreatedAt time.Time, lastTransactionID uuid.UUID, limit int) ([]*core.Transaction, error) {
			if limit != 20 {
				return nil, errors.New("expected default limit 20")
			}
			return []*core.Transaction{{ID: uuid.New(), UserID: userUUID}}, nil
		}
		req := httptest.NewRequest("GET", "/transactions?limit=invalid&lastTransactionCreatedAt=not-rfc-date", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.GetTransactionsByUserUUID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString("invalid json"))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Without Context User UUID", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(`{"uuid":"`+validUUID.String()+`"}`))
		rr := httptest.NewRecorder()

		handler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Repo Error", func(t *testing.T) {
		repo.updateTransactionFn = func(txn *core.Transaction) error {
			return errors.New("update failed")
		}
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(`{"uuid":"`+validUUID.String()+`"}`))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Transaction Not Found", func(t *testing.T) {
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return nil, errors.New("not found")
		}
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(`{"id":"`+validUUID.String()+`"}`))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Success", func(t *testing.T) {
		envID := uuid.New()
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return &core.Transaction{ID: id}, nil
		}
		repo.updateTransactionFn = func(txn *core.Transaction) error {
			if txn.AmountE5 != 500 || txn.Type != "expense" || txn.EnvelopeID == nil || *txn.EnvelopeID != envID {
				return errors.New("mismatched update fields")
			}
			return nil
		}
		body := fmt.Sprintf(`{"id":"%s","amount_e5":500,"txn_type":"expense","envelope_id":"%s"}`, validUUID.String(), envID.String())
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(body))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Cadence Success", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{}, nil
			},
		}
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return &core.Transaction{ID: id}, nil
		}
		cadenceHandler := NewTransactionServiceHandler(repo, shortcutIntentRepo, logger, db, cc, core.RepoContainer{})
		body := fmt.Sprintf(`{"id":"%s","amount_e5":500}`, validUUID.String())
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()

		cadenceHandler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Cadence Execute Error", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return nil, errors.New("execute error")
			},
		}
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return &core.Transaction{ID: id}, nil
		}
		cadenceHandler := NewTransactionServiceHandler(repo, shortcutIntentRepo, logger, db, cc, core.RepoContainer{})
		body := fmt.Sprintf(`{"id":"%s","amount_e5":500}`, validUUID.String())
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()

		cadenceHandler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Cadence WorkflowRun Get Error", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						return errors.New("get error")
					},
				}, nil
			},
		}
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return &core.Transaction{ID: id}, nil
		}
		cadenceHandler := NewTransactionServiceHandler(repo, shortcutIntentRepo, logger, db, cc, core.RepoContainer{})
		body := fmt.Sprintf(`{"id":"%s","amount_e5":500}`, validUUID.String())
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()

		cadenceHandler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Fallback Category Change and Delta Allocations", func(t *testing.T) {
		oldEnvID := uuid.New()
		newEnvID := uuid.New()
		allocRepo := &mockAllocationRepo{
			updateSpentFn: func(envelopeID uuid.UUID, targetDate time.Time, amountDeltaE5 int64, Tx *sql.Tx) error {
				return nil
			},
		}
		fallbackHandler := NewTransactionServiceHandler(repo, shortcutIntentRepo, logger, db, nil, core.RepoContainer{Allocation: allocRepo})

		// 1. Category changed with old debit & new debit
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return &core.Transaction{
				ID:         id,
				EnvelopeID: &oldEnvID,
				AmountE5:   1000,
				Type:       "debit",
				CreatedAt:  time.Now().UTC(),
			}, nil
		}
		repo.updateTransactionFn = func(txn *core.Transaction) error {
			return nil
		}
		body := fmt.Sprintf(`{"id":"%s","envelope_id":"%s","amount_e5":1200,"txn_type":"debit","payment_method":"card"}`, validUUID.String(), newEnvID.String())
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		fallbackHandler.UpdateTransaction(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		// 2. Same category with amount delta change
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return &core.Transaction{
				ID:         id,
				EnvelopeID: &newEnvID,
				AmountE5:   1000,
				Type:       "debit",
			}, nil
		}
		bodySame := fmt.Sprintf(`{"id":"%s","envelope_id":"%s","amount_e5":1500,"txn_type":"debit"}`, validUUID.String(), newEnvID.String())
		reqSame := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(bodySame))
		rrSame := httptest.NewRecorder()
		fallbackHandler.UpdateTransaction(rrSame, reqSame)
		if rrSame.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rrSame.Code)
		}

		// 3. Repo update error
		repo.updateTransactionFn = func(txn *core.Transaction) error {
			return errors.New("db update error")
		}
		reqErr := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(bodySame))
		rrErr := httptest.NewRecorder()
		fallbackHandler.UpdateTransaction(rrErr, reqErr)
		if rrErr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rrErr.Code)
		}
	})

	t.Run("UpdateTransaction - Nil Transaction ID", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(`{}`))
		rr := httptest.NewRecorder()
		handler.UpdateTransaction(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("UpdateTransactionCategory - Handler Tests", func(t *testing.T) {
		// Invalid JSON
		req1 := httptest.NewRequest("POST", "/transaction/category", bytes.NewBufferString("invalid json"))
		rr1 := httptest.NewRecorder()
		handler.UpdateTransactionCategory(rr1, req1)
		if rr1.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr1.Code)
		}

		// Nil Transaction ID
		req2 := httptest.NewRequest("POST", "/transaction/category", bytes.NewBufferString(`{}`))
		rr2 := httptest.NewRecorder()
		handler.UpdateTransactionCategory(rr2, req2)
		if rr2.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr2.Code)
		}

		// Valid Request with Cadence Client
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{}, nil
			},
		}
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return &core.Transaction{ID: id}, nil
		}
		cadenceHandler := NewTransactionServiceHandler(repo, shortcutIntentRepo, logger, db, cc, core.RepoContainer{})
		envID := uuid.New()
		body := fmt.Sprintf(`{"transaction_id":"%s","new_envelope_id":"%s"}`, validUUID.String(), envID.String())
		req3 := httptest.NewRequest("POST", "/transaction/category", bytes.NewBufferString(body))
		rr3 := httptest.NewRecorder()
		cadenceHandler.UpdateTransactionCategory(rr3, req3)
		if rr3.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr3.Code)
		}
	})

	t.Run("DeleteTransaction - Missing or Invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/transaction", nil)
		rr := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), "user_uuid", uuid.Nil)
		req = req.WithContext(ctx)

		handler.DeleteTransaction(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("DeleteTransaction - Repo Error", func(t *testing.T) {
		repo.deleteTransactionFn = func(id uuid.UUID) error {
			return errors.New("delete failed")
		}
		req := httptest.NewRequest("DELETE", "/transaction?txn_uuid="+validUUID.String(), nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.DeleteTransaction(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("DeleteTransaction - Success", func(t *testing.T) {
		repo.deleteTransactionFn = func(id uuid.UUID) error {
			return nil
		}
		req := httptest.NewRequest("DELETE", "/transaction?txn_uuid="+validUUID.String(), nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.DeleteTransaction(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("CreateTransactionWorkflow - Success with Pending Shortcut Intent", func(t *testing.T) {
		txnID := uuid.New()
		intentID := uuid.New()

		localTxnRepo := &mockTxnRepo{
			createTransactionFn: func(txn *core.Transaction) (uuid.UUID, error) {
				return txnID, nil
			},
		}
		localShortcutRepo := &mockShortcutIntentRepo{
			getPendingRecentFn: func(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*core.ShortcutIntent, error) {
				return &core.ShortcutIntent{ID: intentID}, nil
			},
			updateFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
				return nil
			},
		}

		h := NewTransactionServiceHandler(localTxnRepo, localShortcutRepo, logger, nil, nil, core.RepoContainer{})
		res, err := h.CreateTransactionWorkflow(&core.Transaction{AmountE5: 1000}, validUUID, nil)
		if err != nil || res == nil || *res != txnID {
			t.Fatalf("expected txnID %v, got %v, err %v", txnID, res, err)
		}
	})

	t.Run("CreateTransactionWorkflow - Error Fetching Pending Shortcut Intent", func(t *testing.T) {
		localTxnRepo := &mockTxnRepo{}
		localShortcutRepo := &mockShortcutIntentRepo{
			getPendingRecentFn: func(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*core.ShortcutIntent, error) {
				return nil, errors.New("db error")
			},
		}

		h := NewTransactionServiceHandler(localTxnRepo, localShortcutRepo, logger, nil, nil, core.RepoContainer{})
		_, err := h.CreateTransactionWorkflow(&core.Transaction{AmountE5: 1000}, validUUID, nil)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("CreateTransactionWorkflow - Error Updating Shortcut Intent", func(t *testing.T) {
		txnID := uuid.New()
		intentID := uuid.New()

		localTxnRepo := &mockTxnRepo{
			createTransactionFn: func(txn *core.Transaction) (uuid.UUID, error) {
				return txnID, nil
			},
		}
		localShortcutRepo := &mockShortcutIntentRepo{
			getPendingRecentFn: func(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*core.ShortcutIntent, error) {
				return &core.ShortcutIntent{ID: intentID}, nil
			},
			updateFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
				return errors.New("update intent error")
			},
		}

		h := NewTransactionServiceHandler(localTxnRepo, localShortcutRepo, logger, nil, nil, core.RepoContainer{})
		_, err := h.CreateTransactionWorkflow(&core.Transaction{AmountE5: 1000}, validUUID, nil)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("CreateTransactionWorkflow - Error Creating Transaction when Pending Shortcut Exists", func(t *testing.T) {
		localShortcutRepo := &mockShortcutIntentRepo{
			getPendingRecentFn: func(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*core.ShortcutIntent, error) {
				return &core.ShortcutIntent{ID: uuid.New(), Status: core.StatusPending}, nil
			},
		}
		localTxnRepo := &mockTxnRepo{
			createTransactionFn: func(txn *core.Transaction) (uuid.UUID, error) {
				return uuid.Nil, errors.New("db create error")
			},
		}

		h := NewTransactionServiceHandler(localTxnRepo, localShortcutRepo, logger, nil, nil, core.RepoContainer{})
		_, err := h.CreateTransactionWorkflow(&core.Transaction{AmountE5: 1000}, validUUID, nil)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("CreateTransactionWorkflow - Success without Pending Shortcut Intent", func(t *testing.T) {
		txnID := uuid.New()
		localShortcutRepo := &mockShortcutIntentRepo{
			getPendingRecentFn: func(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*core.ShortcutIntent, error) {
				return nil, sql.ErrNoRows
			},
		}
		localTxnRepo := &mockTxnRepo{
			createTransactionFn: func(txn *core.Transaction) (uuid.UUID, error) {
				return txnID, nil
			},
		}

		h := NewTransactionServiceHandler(localTxnRepo, localShortcutRepo, logger, nil, nil, core.RepoContainer{})
		res, err := h.CreateTransactionWorkflow(&core.Transaction{AmountE5: 1000}, validUUID, nil)
		if err != nil || res == nil || *res != txnID {
			t.Fatalf("expected txnID %v, got %v, err %v", txnID, res, err)
		}
	})

	t.Run("CreateTransactionWorkflow - Success with Custom CreatedAt", func(t *testing.T) {
		txnID := uuid.New()
		localShortcutRepo := &mockShortcutIntentRepo{
			getPendingRecentFn: func(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*core.ShortcutIntent, error) {
				return nil, sql.ErrNoRows
			},
		}
		localTxnRepo := &mockTxnRepo{
			createTransactionFn: func(txn *core.Transaction) (uuid.UUID, error) {
				return txnID, nil
			},
		}

		h := NewTransactionServiceHandler(localTxnRepo, localShortcutRepo, logger, nil, nil, core.RepoContainer{})
		res, err := h.CreateTransactionWorkflow(&core.Transaction{AmountE5: 1000, CreatedAt: time.Now()}, validUUID, nil)
		if err != nil || res == nil || *res != txnID {
			t.Fatalf("expected txnID %v, got %v, err %v", txnID, res, err)
		}
	})

	t.Run("CreateTransactionWorkflow - Error Creating Transaction without Pending Shortcut Intent", func(t *testing.T) {
		localShortcutRepo := &mockShortcutIntentRepo{
			getPendingRecentFn: func(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*core.ShortcutIntent, error) {
				return nil, sql.ErrNoRows
			},
		}
		localTxnRepo := &mockTxnRepo{
			createTransactionFn: func(txn *core.Transaction) (uuid.UUID, error) {
				return uuid.Nil, errors.New("db error")
			},
		}

		h := NewTransactionServiceHandler(localTxnRepo, localShortcutRepo, logger, nil, nil, core.RepoContainer{})
		_, err := h.CreateTransactionWorkflow(&core.Transaction{AmountE5: 1000}, validUUID, nil)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("DashboardSummaryHandler - Missing User UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dashboard-summary", nil)
		rr := httptest.NewRecorder()

		handler.DashboardSummaryHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("DashboardSummaryHandler - Invalid User UUID Query", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dashboard-summary?user_uuid=invalid", nil)
		rr := httptest.NewRecorder()

		handler.DashboardSummaryHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("DashboardSummaryHandler - Repo Error", func(t *testing.T) {
		localTxnRepo := &mockTxnRepo{
			getDashboardSummaryFn: func(userUUID uuid.UUID) (*core.DashboardSummary, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewTransactionServiceHandler(localTxnRepo, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
		req := httptest.NewRequest("GET", "/api/dashboard-summary?user_uuid="+validUUID.String(), nil)
		rr := httptest.NewRecorder()

		h.DashboardSummaryHandler(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("DashboardSummaryHandler - Success", func(t *testing.T) {
		localTxnRepo := &mockTxnRepo{
			getDashboardSummaryFn: func(userUUID uuid.UUID) (*core.DashboardSummary, error) {
				return &core.DashboardSummary{TotalIncomeE5: 5000}, nil
			},
		}
		h := NewTransactionServiceHandler(localTxnRepo, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
		req := httptest.NewRequest("GET", "/api/dashboard-summary?user_uuid="+validUUID.String(), nil)
		rr := httptest.NewRecorder()

		h.DashboardSummaryHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("DashboardSummaryHandler - Success with Context User UUID", func(t *testing.T) {
		localTxnRepo := &mockTxnRepo{
			getDashboardSummaryFn: func(userUUID uuid.UUID) (*core.DashboardSummary, error) {
				return &core.DashboardSummary{TotalIncomeE5: 5000}, nil
			},
		}
		h := NewTransactionServiceHandler(localTxnRepo, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
		req := httptest.NewRequest("GET", "/api/dashboard-summary", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		h.DashboardSummaryHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})
}

func TestTransactionServiceHandler_ChangeTransactionToTransfer(t *testing.T) {
	logger := zap.NewNop()
	shortcutIntentRepo := &mockShortcutIntentRepo{}
	db, _, _ := sqlmock.New()
	defer db.Close()
	validUserUUID := uuid.New()
	txnID := uuid.New()

	t.Run("Missing User UUID", func(t *testing.T) {
		h := NewTransactionServiceHandler(&mockTxnRepo{}, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
		req := httptest.NewRequest("POST", "/transaction/transfer", bytes.NewBuffer([]byte(`{"amount_e5": 1000}`)))
		rr := httptest.NewRecorder()

		h.ChangeTransactionToTransfer(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("Invalid JSON Payload", func(t *testing.T) {
		h := NewTransactionServiceHandler(&mockTxnRepo{}, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
		req := httptest.NewRequest("POST", "/transaction/transfer", bytes.NewBuffer([]byte(`invalid-json`)))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.ChangeTransactionToTransfer(rr, req.WithContext(ctx))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("Invalid Amount (Zero or Negative)", func(t *testing.T) {
		h := NewTransactionServiceHandler(&mockTxnRepo{}, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
		req := httptest.NewRequest("POST", "/transaction/transfer", bytes.NewBuffer([]byte(`{"amount_e5": 0}`)))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.ChangeTransactionToTransfer(rr, req.WithContext(ctx))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("Transaction Not Found (ErrNoRows)", func(t *testing.T) {
		txnRepo := &mockTxnRepo{
			getTransactionByAmountAndTimeFn: func(userUUID uuid.UUID, amountE5 int64, time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
				return nil, sql.ErrNoRows
			},
		}
		h := NewTransactionServiceHandler(txnRepo, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
		req := httptest.NewRequest("POST", "/transaction/transfer", bytes.NewBuffer([]byte(`{"amount_e5": 1000}`)))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.ChangeTransactionToTransfer(rr, req.WithContext(ctx))
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("Find Transaction DB Error", func(t *testing.T) {
		txnRepo := &mockTxnRepo{
			getTransactionByAmountAndTimeFn: func(userUUID uuid.UUID, amountE5 int64, time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
				return nil, errors.New("db find error")
			},
		}
		h := NewTransactionServiceHandler(txnRepo, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
		req := httptest.NewRequest("POST", "/transaction/transfer", bytes.NewBuffer([]byte(`{"amount_e5": 1000}`)))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.ChangeTransactionToTransfer(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("Update Transaction DB Error", func(t *testing.T) {
		txnRepo := &mockTxnRepo{
			getTransactionByAmountAndTimeFn: func(userUUID uuid.UUID, amountE5 int64, time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
				return &core.Transaction{ID: txnID, UserID: userUUID, AmountE5: amountE5, Type: "debit"}, nil
			},
			updateTransactionFn: func(txn *core.Transaction) error {
				return errors.New("db update error")
			},
		}
		h := NewTransactionServiceHandler(txnRepo, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
		req := httptest.NewRequest("POST", "/transaction/transfer", bytes.NewBuffer([]byte(`{"amount_e5": 1000}`)))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.ChangeTransactionToTransfer(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("Success with Custom CreatedAt", func(t *testing.T) {
		customTime := time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)
		var capturedLower, capturedUpper time.Time
		txnRepo := &mockTxnRepo{
			getTransactionByAmountAndTimeFn: func(userUUID uuid.UUID, amountE5 int64, time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
				capturedLower = time_lowerbound
				capturedUpper = time_upperbound
				return &core.Transaction{ID: txnID, UserID: userUUID, AmountE5: amountE5, Type: "debit", CreatedAt: customTime}, nil
			},
			updateTransactionFn: func(txn *core.Transaction) error {
				if txn.Type != core.TxnTypeTransfer {
					return errors.New("expected type transfer")
				}
				return nil
			},
		}
		h := NewTransactionServiceHandler(txnRepo, shortcutIntentRepo, logger, db, nil, core.RepoContainer{})
		body, _ := json.Marshal(map[string]interface{}{
			"amount_e5":  1000,
			"created_at": customTime.Format(time.RFC3339),
		})
		req := httptest.NewRequest("POST", "/transaction/transfer", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.ChangeTransactionToTransfer(rr, req.WithContext(ctx))
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
		if !capturedLower.Equal(customTime.Add(-5 * time.Minute)) || !capturedUpper.Equal(customTime.Add(5 * time.Minute)) {
			t.Errorf("unexpected time window: [%v, %v]", capturedLower, capturedUpper)
		}

		var respTxn core.Transaction
		if err := json.NewDecoder(rr.Body).Decode(&respTxn); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if respTxn.Type != core.TxnTypeTransfer {
			t.Errorf("expected type %s, got %s", core.TxnTypeTransfer, respTxn.Type)
		}
	})
}

func TestTransactionServiceHandler_ProcessEmailTransaction(t *testing.T) {
	logger := zap.NewNop()
	db, _, _ := sqlmock.New()
	defer db.Close()

	validUserUUID := uuid.New()
	createdTxnID := uuid.New()

	t.Run("Success Bank Account Email", func(t *testing.T) {
		txnRepo := &mockTxnRepo{
			createTransactionFn: func(txn *core.Transaction) (uuid.UUID, error) {
				if txn.AmountE5 != 1000000 {
					t.Errorf("expected amount_e5 1000000, got %d", txn.AmountE5)
				}
				if txn.PaymentMethod != "bank_account" {
					t.Errorf("expected payment_method bank_account, got %s", txn.PaymentMethod)
				}
				if txn.Type != core.TxnTypeDebit {
					t.Errorf("expected type debit, got %s", txn.Type)
				}
				return createdTxnID, nil
			},
		}
		shortcutRepo := &mockShortcutIntentRepo{}
		h := NewTransactionServiceHandler(txnRepo, shortcutRepo, logger, db, nil, core.RepoContainer{})

		body := map[string]interface{}{
			"user_id": validUserUUID.String(),
			"subject": "Debit Alert",
			"body": `Dear Mr. Barath Surya M,
Greetings from IDFC FIRST Bank.
Your A/C XXXXXXX2559 has been debited by INR 10.00 on 22/09/2026 16:08. New balance is INR 33,918.42CR.`,
			"email_date": "2026-09-22T16:08:00Z",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/transaction/email", bytes.NewBuffer(jsonBytes))
		rr := httptest.NewRecorder()

		h.ProcessEmailTransaction(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Success Credit Card Email", func(t *testing.T) {
		txnRepo := &mockTxnRepo{
			createTransactionFn: func(txn *core.Transaction) (uuid.UUID, error) {
				if txn.AmountE5 != 117150000 {
					t.Errorf("expected amount_e5 117150000, got %d", txn.AmountE5)
				}
				if txn.PaymentMethod != "bank_card" {
					t.Errorf("expected payment_method bank_card, got %s", txn.PaymentMethod)
				}
				if txn.Type != core.TxnTypeDebit {
					t.Errorf("expected type debit, got %s", txn.Type)
				}
				return createdTxnID, nil
			},
		}
		shortcutRepo := &mockShortcutIntentRepo{}
		h := NewTransactionServiceHandler(txnRepo, shortcutRepo, logger, db, nil, core.RepoContainer{})

		body := map[string]interface{}{
			"subject": "Credit Card Spend",
			"body": `Dear Cardmember,
All Stocked Up! INR 1171.50 spent on your IDFC FIRST BANK Credit Card ending XX1110 at AVENUE SUPERMARTS LI on 22 SEP 2026.
Available Limit: INR 38413.57 .`,
			"email_date": "2026-09-22T12:00:00Z",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/transaction/email", bytes.NewBuffer(jsonBytes))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.ProcessEmailTransaction(rr, req.WithContext(ctx))
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("Invalid Email Body", func(t *testing.T) {
		txnRepo := &mockTxnRepo{}
		shortcutRepo := &mockShortcutIntentRepo{}
		h := NewTransactionServiceHandler(txnRepo, shortcutRepo, logger, db, nil, core.RepoContainer{})

		body := map[string]interface{}{
			"user_id": validUserUUID.String(),
			"subject": "Spam",
			"body":    "Congratulations, you won a lottery!",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/transaction/email", bytes.NewBuffer(jsonBytes))
		rr := httptest.NewRecorder()

		h.ProcessEmailTransaction(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Missing User UUID", func(t *testing.T) {
		txnRepo := &mockTxnRepo{}
		shortcutRepo := &mockShortcutIntentRepo{}
		h := NewTransactionServiceHandler(txnRepo, shortcutRepo, logger, db, nil, core.RepoContainer{})

		body := map[string]interface{}{
			"subject": "Alert",
			"body":    "Your A/C XXXXXXX2559 has been debited by INR 10.00",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/transaction/email", bytes.NewBuffer(jsonBytes))
		rr := httptest.NewRecorder()

		h.ProcessEmailTransaction(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Invalid Payload JSON", func(t *testing.T) {
		txnRepo := &mockTxnRepo{}
		shortcutRepo := &mockShortcutIntentRepo{}
		h := NewTransactionServiceHandler(txnRepo, shortcutRepo, logger, db, nil, core.RepoContainer{})

		req := httptest.NewRequest("POST", "/transaction/email", bytes.NewBuffer([]byte("{invalid-json")))
		rr := httptest.NewRecorder()

		h.ProcessEmailTransaction(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("ProcessEmailTransaction - Cadence Success", func(t *testing.T) {
		txnRepo := &mockTxnRepo{}
		shortcutRepo := &mockShortcutIntentRepo{}
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						return nil
					},
				}, nil
			},
		}
		h := NewTransactionServiceHandler(txnRepo, shortcutRepo, logger, db, cc, core.RepoContainer{})

		body := map[string]interface{}{
			"user_id": validUserUUID.String(),
			"subject": "Alert",
			"body":    "Your A/C debited by INR 10.00",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/transaction/email", bytes.NewBuffer(jsonBytes))
		rr := httptest.NewRecorder()

		h.ProcessEmailTransaction(rr, req)
		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rr.Code)
		}
	})

	t.Run("ProcessEmailTransaction - Cadence Execute Error", func(t *testing.T) {
		txnRepo := &mockTxnRepo{}
		shortcutRepo := &mockShortcutIntentRepo{}
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return nil, errors.New("execute error")
			},
		}
		h := NewTransactionServiceHandler(txnRepo, shortcutRepo, logger, db, cc, core.RepoContainer{})

		body := map[string]interface{}{
			"user_id": validUserUUID.String(),
			"subject": "Alert",
			"body":    "Your A/C debited by INR 10.00",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/transaction/email", bytes.NewBuffer(jsonBytes))
		rr := httptest.NewRecorder()

		h.ProcessEmailTransaction(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})

	t.Run("ProcessEmailTransaction - Cadence Get Error", func(t *testing.T) {
		txnRepo := &mockTxnRepo{}
		shortcutRepo := &mockShortcutIntentRepo{}
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						return errors.New("workflow error")
					},
				}, nil
			},
		}
		h := NewTransactionServiceHandler(txnRepo, shortcutRepo, logger, db, cc, core.RepoContainer{})

		body := map[string]interface{}{
			"user_id": validUserUUID.String(),
			"subject": "Alert",
			"body":    "Your A/C debited by INR 10.00",
		}
		jsonBytes, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/transaction/email", bytes.NewBuffer(jsonBytes))
		rr := httptest.NewRecorder()

		h.ProcessEmailTransaction(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}


