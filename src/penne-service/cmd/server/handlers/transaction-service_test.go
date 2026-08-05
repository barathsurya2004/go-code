package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
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

func TestTransactionServiceHandler(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockTxnRepo{}
	handler := NewTransactionServiceHandler(repo, logger)

	t.Run("CreateTransaction - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/transaction", bytes.NewBufferString("invalid json"))
		rr := httptest.NewRecorder()

		handler.CreateTransaction(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateTransaction - Repo Error", func(t *testing.T) {
		repo.createTransactionFn = func(txn *core.Transaction) error {
			return errors.New("db error")
		}
		req := httptest.NewRequest("POST", "/transaction", bytes.NewBufferString(`{"amount_e5":100}`))
		rr := httptest.NewRecorder()

		handler.CreateTransaction(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateTransaction - Success", func(t *testing.T) {
		repo.createTransactionFn = func(txn *core.Transaction) error {
			return nil
		}
		req := httptest.NewRequest("POST", "/transaction", bytes.NewBufferString(`{"amount_e5":100}`))
		rr := httptest.NewRecorder()

		handler.CreateTransaction(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})

	t.Run("GetTransactionByUUID - Missing UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/transaction", nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionByUUID(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetTransactionByUUID - Repo Error", func(t *testing.T) {
		repo.getTransactionByUUIDFn = func(uuid string) (*core.Transaction, error) {
			return nil, errors.New("not found")
		}
		req := httptest.NewRequest("GET", "/transaction?uuid=txn-123", nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionByUUID(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("GetTransactionByUUID - Success", func(t *testing.T) {
		repo.getTransactionByUUIDFn = func(uuid string) (*core.Transaction, error) {
			return &core.Transaction{UUID: uuid, AmountE5: 500}, nil
		}
		req := httptest.NewRequest("GET", "/transaction?uuid=txn-123", nil)
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
		repo.getTransactionsByUserUUIDFn = func(userUUID string) ([]*core.Transaction, error) {
			return nil, errors.New("db error")
		}
		req := httptest.NewRequest("GET", "/transactions?user_uuid=user-123", nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionsByUserUUID(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetTransactionsByUserUUID - Success", func(t *testing.T) {
		repo.getTransactionsByUserUUIDFn = func(userUUID string) ([]*core.Transaction, error) {
			return []*core.Transaction{{UUID: "txn-1", UserUUID: userUUID}}, nil
		}
		req := httptest.NewRequest("GET", "/transactions?user_uuid=user-123", nil)
		rr := httptest.NewRecorder()

		handler.GetTransactionsByUserUUID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString("invalid json"))
		rr := httptest.NewRecorder()

		handler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Repo Error", func(t *testing.T) {
		repo.updateTransactionFn = func(txn *core.Transaction) error {
			return errors.New("update failed")
		}
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(`{"uuid":"txn-123"}`))
		rr := httptest.NewRecorder()

		handler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("UpdateTransaction - Success", func(t *testing.T) {
		repo.updateTransactionFn = func(txn *core.Transaction) error {
			return nil
		}
		req := httptest.NewRequest("PUT", "/transaction", bytes.NewBufferString(`{"uuid":"txn-123"}`))
		rr := httptest.NewRecorder()

		handler.UpdateTransaction(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("DeleteTransaction - Missing UUID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/transaction", nil)
		rr := httptest.NewRecorder()

		handler.DeleteTransaction(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("DeleteTransaction - Repo Error", func(t *testing.T) {
		repo.deleteTransactionFn = func(uuid string) error {
			return errors.New("delete failed")
		}
		req := httptest.NewRequest("DELETE", "/transaction?uuid=txn-123", nil)
		rr := httptest.NewRecorder()

		handler.DeleteTransaction(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("DeleteTransaction - Success", func(t *testing.T) {
		repo.deleteTransactionFn = func(uuid string) error {
			return nil
		}
		req := httptest.NewRequest("DELETE", "/transaction?uuid=txn-123", nil)
		rr := httptest.NewRecorder()

		handler.DeleteTransaction(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})
}
