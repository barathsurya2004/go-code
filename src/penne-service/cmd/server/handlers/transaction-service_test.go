package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
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
	getTransactionByTimeFn      func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error)
	getDashboardSummaryFn       func(uuid uuid.UUID) (*core.DashboardSummary, error)
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

func (m *mockTxnRepo) GetDashboardSummary(userUUID uuid.UUID) (*core.DashboardSummary, error) {
	if m.getDashboardSummaryFn != nil {
		return m.getDashboardSummaryFn(userUUID)
	}
	return &core.DashboardSummary{}, nil
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

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
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

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
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

	t.Run("GetTransactionsByUserUUID - Repo Error", func(t *testing.T) {
		repo.getTransactionsByUserUUIDFn = func(userUUID uuid.UUID) ([]*core.Transaction, error) {
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
		repo.getTransactionsByUserUUIDFn = func(userUUID uuid.UUID) ([]*core.Transaction, error) {
			return []*core.Transaction{{ID: uuid.New(), UserID: userUUID}}, nil
		}
		req := httptest.NewRequest("GET", "/transactions?user_uuid="+validUUID.String(), nil)
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
		repo.getTransactionByUUIDFn = func(id uuid.UUID) (*core.Transaction, error) {
			return &core.Transaction{ID: id}, nil
		}
		repo.updateTransactionFn = func(txn *core.Transaction) error {
			return nil
		}
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(`{"id":"`+validUUID.String()+`"}`))
		req = req.WithContext(context.WithValue(req.Context(), "user_uuid", validUUID))
		rr := httptest.NewRecorder()

		handler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
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
}
