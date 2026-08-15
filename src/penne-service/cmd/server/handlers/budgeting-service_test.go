package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockEnvelopeGroupRepo struct {
	createFn    func(group *core.EnvelopeGroup) (uuid.UUID, error)
	getByIDFn   func(id uuid.UUID) (*core.EnvelopeGroup, error)
	getByUserFn func(userUUID uuid.UUID) ([]*core.EnvelopeGroup, error)
	updateFn    func(group *core.EnvelopeGroup) error
	deleteFn    func(id uuid.UUID) error
}

func (m *mockEnvelopeGroupRepo) CreateEnvelopeGroup(group *core.EnvelopeGroup, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(group)
	}
	return uuid.Nil, nil
}

func (m *mockEnvelopeGroupRepo) GetEnvelopeGroupByID(id uuid.UUID) (*core.EnvelopeGroup, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}

func (m *mockEnvelopeGroupRepo) GetEnvelopeGroupsByUserUUID(userUUID uuid.UUID) ([]*core.EnvelopeGroup, error) {
	if m.getByUserFn != nil {
		return m.getByUserFn(userUUID)
	}
	return nil, nil
}

func (m *mockEnvelopeGroupRepo) UpdateEnvelopeGroup(group *core.EnvelopeGroup) error {
	if m.updateFn != nil {
		return m.updateFn(group)
	}
	return nil
}

func (m *mockEnvelopeGroupRepo) DeleteEnvelopeGroup(id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

type mockEnvelopeRepo struct {
	createFn    func(env *core.Envelope) (uuid.UUID, error)
	getByIDFn   func(id uuid.UUID) (*core.Envelope, error)
	getByUserFn func(userUUID uuid.UUID) ([]*core.Envelope, error)
	updateFn    func(env *core.Envelope) error
	deleteFn    func(id uuid.UUID) error
	getByNameFn func(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error)
}

func (m *mockEnvelopeRepo) CreateEnvelope(env *core.Envelope, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(env)
	}
	return uuid.Nil, nil
}

func (m *mockEnvelopeRepo) GetEnvelopeByID(id uuid.UUID) (*core.Envelope, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}

func (m *mockEnvelopeRepo) GetEnvelopesByUserUUID(userUUID uuid.UUID) ([]*core.Envelope, error) {
	if m.getByUserFn != nil {
		return m.getByUserFn(userUUID)
	}
	return nil, nil
}

func (m *mockEnvelopeRepo) UpdateEnvelope(env *core.Envelope) error {
	if m.updateFn != nil {
		return m.updateFn(env)
	}
	return nil
}

func (m *mockEnvelopeRepo) DeleteEnvelope(id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

func (m *mockEnvelopeRepo) GetEnvelopeIdByName(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
	if m.getByNameFn != nil {
		return m.getByNameFn(envlopeName, userUUID, tx)
	}
	return uuid.Nil, nil
}

type mockAllocationRepo struct {
	createFn          func(alloc *core.Allocation) (uuid.UUID, error)
	getByIDFn         func(id uuid.UUID) (*core.Allocation, error)
	getByEnvelopeFn   func(envelopeID uuid.UUID) ([]*core.Allocation, error)
	getActiveByUserFn func(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error)
	updateFn          func(alloc *core.Allocation) error
	deleteFn          func(id uuid.UUID) error
}

func (m *mockAllocationRepo) CreateAllocation(alloc *core.Allocation, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(alloc)
	}
	return uuid.Nil, nil
}

func (m *mockAllocationRepo) GetAllocationByID(id uuid.UUID) (*core.Allocation, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}

func (m *mockAllocationRepo) GetAllocationsByEnvelopeID(envelopeID uuid.UUID) ([]*core.Allocation, error) {
	if m.getByEnvelopeFn != nil {
		return m.getByEnvelopeFn(envelopeID)
	}
	return nil, nil
}

func (m *mockAllocationRepo) GetActiveAllocationsByUserUUID(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error) {
	if m.getActiveByUserFn != nil {
		return m.getActiveByUserFn(userUUID, targetDate, Tx)
	}
	return nil, nil
}

func (m *mockAllocationRepo) UpdateAllocation(alloc *core.Allocation) error {
	if m.updateFn != nil {
		return m.updateFn(alloc)
	}
	return nil
}

func (m *mockAllocationRepo) DeleteAllocation(id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

type mockShortcutIntentRepo struct {
	createFn    func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error)
	getByIDFn   func(id uuid.UUID) (*core.ShortcutIntent, error)
	getByUserFn func(userUUID uuid.UUID) ([]*core.ShortcutIntent, error)
	updateFn    func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error
	deleteFn    func(id uuid.UUID) error
}

func (m *mockShortcutIntentRepo) CreateShortcutIntent(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(shortcutIntent, Tx)
	}
	return uuid.Nil, nil
}
func (m *mockShortcutIntentRepo) GetShortcutIntentByID(id uuid.UUID) (*core.ShortcutIntent, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}
func (m *mockShortcutIntentRepo) GetShortcutIntentsByUserUUID(userUUID uuid.UUID) ([]*core.ShortcutIntent, error) {
	if m.getByUserFn != nil {
		return m.getByUserFn(userUUID)
	}
	return nil, nil
}
func (m *mockShortcutIntentRepo) UpdateShortcutIntent(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
	if m.updateFn != nil {
		return m.updateFn(shortcutIntent, Tx)
	}
	return nil
}
func (m *mockShortcutIntentRepo) DeleteShortcutIntent(id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

func TestBudgetingServiceHandler_EnvelopeGroup(t *testing.T) {
	logger := zap.NewNop()
	groupRepo := &mockEnvelopeGroupRepo{}
	envRepo := &mockEnvelopeRepo{}
	allocRepo := &mockAllocationRepo{}
	txnRepo := &mockTxnRepo{}
	shortcutIntentRepo := &mockShortcutIntentRepo{}
	mockDB, _, _ := sqlmock.New()
	defer mockDB.Close()

	handler := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, txnRepo, shortcutIntentRepo, logger, mockDB)
	validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("CreateEnvelopeGroup - Success", func(t *testing.T) {
		groupRepo.createFn = func(group *core.EnvelopeGroup) (uuid.UUID, error) {
			return uuid.New(), nil
		}
		body, _ := json.Marshal(core.EnvelopeGroup{Name: "Bills"})
		req := httptest.NewRequest("POST", "/envelope-group", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.CreateEnvelopeGroup(rr, req.WithContext(ctx))
		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})

	t.Run("CreateEnvelopeGroup - Missing Context", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/envelope-group", nil)
		rr := httptest.NewRecorder()

		handler.CreateEnvelopeGroup(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateEnvelopeGroup - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/envelope-group", bytes.NewBufferString("{invalid"))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.CreateEnvelopeGroup(rr, req.WithContext(ctx))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateEnvelopeGroup - Repo Error", func(t *testing.T) {
		groupRepo.createFn = func(group *core.EnvelopeGroup) (uuid.UUID, error) {
			return uuid.Nil, errors.New("db error")
		}
		body, _ := json.Marshal(core.EnvelopeGroup{Name: "Bills"})
		req := httptest.NewRequest("POST", "/envelope-group", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.CreateEnvelopeGroup(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetEnvelopeGroupByID - Success", func(t *testing.T) {
		groupID := uuid.New()
		groupRepo.getByIDFn = func(id uuid.UUID) (*core.EnvelopeGroup, error) {
			return &core.EnvelopeGroup{ID: groupID, Name: "Bills"}, nil
		}

		req := httptest.NewRequest("GET", "/envelope-group?id="+groupID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetEnvelopeGroupByID(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetEnvelopeGroupByID - Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/envelope-group?id=invalid-uuid", nil)
		rr := httptest.NewRecorder()

		handler.GetEnvelopeGroupByID(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetEnvelopeGroupByID - Repo Error", func(t *testing.T) {
		groupID := uuid.New()
		groupRepo.getByIDFn = func(id uuid.UUID) (*core.EnvelopeGroup, error) {
			return nil, errors.New("not found")
		}

		req := httptest.NewRequest("GET", "/envelope-group?id="+groupID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetEnvelopeGroupByID(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("GetEnvelopeGroupsByUserUUID - Success", func(t *testing.T) {
		groupRepo.getByUserFn = func(userUUID uuid.UUID) ([]*core.EnvelopeGroup, error) {
			return []*core.EnvelopeGroup{{Name: "Bills"}}, nil
		}

		req := httptest.NewRequest("GET", "/envelope-groups", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.GetEnvelopeGroupsByUserUUID(rr, req.WithContext(ctx))
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetEnvelopeGroupsByUserUUID - Query Param Fallback", func(t *testing.T) {
		groupRepo.getByUserFn = func(userUUID uuid.UUID) ([]*core.EnvelopeGroup, error) {
			return []*core.EnvelopeGroup{{Name: "Bills"}}, nil
		}

		req := httptest.NewRequest("GET", "/envelope-groups?user_uuid="+validUserUUID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetEnvelopeGroupsByUserUUID(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetEnvelopeGroupsByUserUUID - Missing User UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/envelope-groups", nil)
		rr := httptest.NewRecorder()

		handler.GetEnvelopeGroupsByUserUUID(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetEnvelopeGroupsByUserUUID - Repo Error", func(t *testing.T) {
		groupRepo.getByUserFn = func(userUUID uuid.UUID) ([]*core.EnvelopeGroup, error) {
			return nil, errors.New("db error")
		}

		req := httptest.NewRequest("GET", "/envelope-groups", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.GetEnvelopeGroupsByUserUUID(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("UpdateEnvelopeGroup - Success", func(t *testing.T) {
		groupRepo.updateFn = func(group *core.EnvelopeGroup) error {
			return nil
		}

		body, _ := json.Marshal(core.EnvelopeGroup{ID: uuid.New(), Name: "Updated Bills"})
		req := httptest.NewRequest("PUT", "/envelope-group", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.UpdateEnvelopeGroup(rr, req.WithContext(ctx))
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("UpdateEnvelopeGroup - Missing Context", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/envelope-group", nil)
		rr := httptest.NewRecorder()

		handler.UpdateEnvelopeGroup(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("UpdateEnvelopeGroup - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/envelope-group", bytes.NewBufferString("{invalid"))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.UpdateEnvelopeGroup(rr, req.WithContext(ctx))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("UpdateEnvelopeGroup - Repo Error", func(t *testing.T) {
		groupRepo.updateFn = func(group *core.EnvelopeGroup) error {
			return errors.New("update error")
		}

		body, _ := json.Marshal(core.EnvelopeGroup{ID: uuid.New(), Name: "Updated Bills"})
		req := httptest.NewRequest("PUT", "/envelope-group", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.UpdateEnvelopeGroup(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("DeleteEnvelopeGroup - Success", func(t *testing.T) {
		groupID := uuid.New()
		groupRepo.deleteFn = func(id uuid.UUID) error {
			return nil
		}

		req := httptest.NewRequest("DELETE", "/envelope-group?id="+groupID.String(), nil)
		rr := httptest.NewRecorder()

		handler.DeleteEnvelopeGroup(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, rr.Code)
		}
	})

	t.Run("DeleteEnvelopeGroup - Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/envelope-group?id=invalid-uuid", nil)
		rr := httptest.NewRecorder()

		handler.DeleteEnvelopeGroup(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("DeleteEnvelopeGroup - Repo Error", func(t *testing.T) {
		groupID := uuid.New()
		groupRepo.deleteFn = func(id uuid.UUID) error {
			return errors.New("delete error")
		}

		req := httptest.NewRequest("DELETE", "/envelope-group?id="+groupID.String(), nil)
		rr := httptest.NewRecorder()

		handler.DeleteEnvelopeGroup(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})
}

func TestBudgetingServiceHandler_Envelope(t *testing.T) {
	logger := zap.NewNop()
	groupRepo := &mockEnvelopeGroupRepo{}
	envRepo := &mockEnvelopeRepo{}
	allocRepo := &mockAllocationRepo{}
	txnRepo := &mockTxnRepo{}
	shortcutIntentRepo := &mockShortcutIntentRepo{}
	mockDB, _, _ := sqlmock.New()
	defer mockDB.Close()

	handler := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, txnRepo, shortcutIntentRepo, logger, mockDB)
	validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("CreateEnvelope - Success", func(t *testing.T) {
		envRepo.createFn = func(env *core.Envelope) (uuid.UUID, error) {
			return uuid.New(), nil
		}

		body, _ := json.Marshal(core.Envelope{TargetAmountE5: 10000})
		req := httptest.NewRequest("POST", "/envelope", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.CreateEnvelope(rr, req.WithContext(ctx))
		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})

	t.Run("CreateEnvelope - Missing Context", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/envelope", nil)
		rr := httptest.NewRecorder()

		handler.CreateEnvelope(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateEnvelope - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/envelope", bytes.NewBufferString("{invalid"))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.CreateEnvelope(rr, req.WithContext(ctx))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateEnvelope - Repo Error", func(t *testing.T) {
		envRepo.createFn = func(env *core.Envelope) (uuid.UUID, error) {
			return uuid.Nil, errors.New("create error")
		}

		body, _ := json.Marshal(core.Envelope{TargetAmountE5: 10000})
		req := httptest.NewRequest("POST", "/envelope", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.CreateEnvelope(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetEnvelopeByID - Success", func(t *testing.T) {
		envID := uuid.New()
		envRepo.getByIDFn = func(id uuid.UUID) (*core.Envelope, error) {
			return &core.Envelope{ID: envID}, nil
		}

		req := httptest.NewRequest("GET", "/envelope?id="+envID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetEnvelopeByID(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetEnvelopeByID - Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/envelope?id=invalid-uuid", nil)
		rr := httptest.NewRecorder()

		handler.GetEnvelopeByID(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetEnvelopeByID - Repo Error", func(t *testing.T) {
		envID := uuid.New()
		envRepo.getByIDFn = func(id uuid.UUID) (*core.Envelope, error) {
			return nil, errors.New("not found")
		}

		req := httptest.NewRequest("GET", "/envelope?id="+envID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetEnvelopeByID(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("GetEnvelopesByUserUUID - Success", func(t *testing.T) {
		envRepo.getByUserFn = func(userUUID uuid.UUID) ([]*core.Envelope, error) {
			return []*core.Envelope{{UserUUID: userUUID}}, nil
		}

		req := httptest.NewRequest("GET", "/envelopes", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.GetEnvelopesByUserUUID(rr, req.WithContext(ctx))
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetEnvelopesByUserUUID - Query Param Fallback", func(t *testing.T) {
		envRepo.getByUserFn = func(userUUID uuid.UUID) ([]*core.Envelope, error) {
			return []*core.Envelope{{UserUUID: userUUID}}, nil
		}

		req := httptest.NewRequest("GET", "/envelopes?user_uuid="+validUserUUID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetEnvelopesByUserUUID(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetEnvelopesByUserUUID - Missing User UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/envelopes", nil)
		rr := httptest.NewRecorder()

		handler.GetEnvelopesByUserUUID(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetEnvelopesByUserUUID - Repo Error", func(t *testing.T) {
		envRepo.getByUserFn = func(userUUID uuid.UUID) ([]*core.Envelope, error) {
			return nil, errors.New("db error")
		}

		req := httptest.NewRequest("GET", "/envelopes", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.GetEnvelopesByUserUUID(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("UpdateEnvelope - Success", func(t *testing.T) {
		envRepo.updateFn = func(env *core.Envelope) error {
			return nil
		}

		body, _ := json.Marshal(core.Envelope{ID: uuid.New(), TargetAmountE5: 20000})
		req := httptest.NewRequest("PUT", "/envelope", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.UpdateEnvelope(rr, req.WithContext(ctx))
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("UpdateEnvelope - Missing Context", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/envelope", nil)
		rr := httptest.NewRecorder()

		handler.UpdateEnvelope(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("UpdateEnvelope - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/envelope", bytes.NewBufferString("{invalid"))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.UpdateEnvelope(rr, req.WithContext(ctx))
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("UpdateEnvelope - Repo Error", func(t *testing.T) {
		envRepo.updateFn = func(env *core.Envelope) error {
			return errors.New("update error")
		}

		body, _ := json.Marshal(core.Envelope{ID: uuid.New(), TargetAmountE5: 20000})
		req := httptest.NewRequest("PUT", "/envelope", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.UpdateEnvelope(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("DeleteEnvelope - Success", func(t *testing.T) {
		envID := uuid.New()
		envRepo.deleteFn = func(id uuid.UUID) error {
			return nil
		}

		req := httptest.NewRequest("DELETE", "/envelope?id="+envID.String(), nil)
		rr := httptest.NewRecorder()

		handler.DeleteEnvelope(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, rr.Code)
		}
	})

	t.Run("DeleteEnvelope - Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/envelope?id=invalid-uuid", nil)
		rr := httptest.NewRecorder()

		handler.DeleteEnvelope(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("DeleteEnvelope - Repo Error", func(t *testing.T) {
		envID := uuid.New()
		envRepo.deleteFn = func(id uuid.UUID) error {
			return errors.New("delete error")
		}

		req := httptest.NewRequest("DELETE", "/envelope?id="+envID.String(), nil)
		rr := httptest.NewRecorder()

		handler.DeleteEnvelope(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})
}

func TestBudgetingServiceHandler_Allocation(t *testing.T) {
	logger := zap.NewNop()
	groupRepo := &mockEnvelopeGroupRepo{}
	envRepo := &mockEnvelopeRepo{}
	allocRepo := &mockAllocationRepo{}
	txnRepo := &mockTxnRepo{}
	shortcutIntentRepo := &mockShortcutIntentRepo{}
	mockDB, _, _ := sqlmock.New()
	defer mockDB.Close()

	handler := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, txnRepo, shortcutIntentRepo, logger, mockDB)

	t.Run("CreateAllocation - Success", func(t *testing.T) {
		allocRepo.createFn = func(alloc *core.Allocation) (uuid.UUID, error) {
			return uuid.New(), nil
		}

		body, _ := json.Marshal(core.Allocation{AllocatedAmountE5: 50000})
		req := httptest.NewRequest("POST", "/allocation", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.CreateAllocation(rr, req)
		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})

	t.Run("CreateAllocation - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/allocation", bytes.NewBufferString("{invalid"))
		rr := httptest.NewRecorder()

		handler.CreateAllocation(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateAllocation - Repo Error", func(t *testing.T) {
		allocRepo.createFn = func(alloc *core.Allocation) (uuid.UUID, error) {
			return uuid.Nil, errors.New("create error")
		}

		body, _ := json.Marshal(core.Allocation{AllocatedAmountE5: 50000})
		req := httptest.NewRequest("POST", "/allocation", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.CreateAllocation(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetAllocationByID - Success", func(t *testing.T) {
		allocID := uuid.New()
		allocRepo.getByIDFn = func(id uuid.UUID) (*core.Allocation, error) {
			return &core.Allocation{ID: allocID}, nil
		}

		req := httptest.NewRequest("GET", "/allocation?id="+allocID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetAllocationByID(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetAllocationByID - Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/allocation?id=invalid-uuid", nil)
		rr := httptest.NewRecorder()

		handler.GetAllocationByID(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetAllocationByID - Repo Error", func(t *testing.T) {
		allocID := uuid.New()
		allocRepo.getByIDFn = func(id uuid.UUID) (*core.Allocation, error) {
			return nil, errors.New("not found")
		}

		req := httptest.NewRequest("GET", "/allocation?id="+allocID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetAllocationByID(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("GetAllocationsByEnvelopeID - Success", func(t *testing.T) {
		envID := uuid.New()
		allocRepo.getByEnvelopeFn = func(envelopeID uuid.UUID) ([]*core.Allocation, error) {
			return []*core.Allocation{{EnvelopeID: envelopeID}}, nil
		}

		req := httptest.NewRequest("GET", "/allocations?envelope_id="+envID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetAllocationsByEnvelopeID(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetAllocationsByEnvelopeID - Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/allocations?envelope_id=invalid-uuid", nil)
		rr := httptest.NewRecorder()

		handler.GetAllocationsByEnvelopeID(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetAllocationsByEnvelopeID - Repo Error", func(t *testing.T) {
		envID := uuid.New()
		allocRepo.getByEnvelopeFn = func(envelopeID uuid.UUID) ([]*core.Allocation, error) {
			return nil, errors.New("db error")
		}

		req := httptest.NewRequest("GET", "/allocations?envelope_id="+envID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetAllocationsByEnvelopeID(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("UpdateAllocation - Success", func(t *testing.T) {
		allocRepo.updateFn = func(alloc *core.Allocation) error {
			return nil
		}

		body, _ := json.Marshal(core.Allocation{ID: uuid.New(), AllocatedAmountE5: 75000})
		req := httptest.NewRequest("PUT", "/allocation", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.UpdateAllocation(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("UpdateAllocation - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/allocation", bytes.NewBufferString("{invalid"))
		rr := httptest.NewRecorder()

		handler.UpdateAllocation(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("UpdateAllocation - Repo Error", func(t *testing.T) {
		allocRepo.updateFn = func(alloc *core.Allocation) error {
			return errors.New("update error")
		}

		body, _ := json.Marshal(core.Allocation{ID: uuid.New(), AllocatedAmountE5: 75000})
		req := httptest.NewRequest("PUT", "/allocation", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.UpdateAllocation(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("DeleteAllocation - Success", func(t *testing.T) {
		allocID := uuid.New()
		allocRepo.deleteFn = func(id uuid.UUID) error {
			return nil
		}

		req := httptest.NewRequest("DELETE", "/allocation?id="+allocID.String(), nil)
		rr := httptest.NewRecorder()

		handler.DeleteAllocation(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, rr.Code)
		}
	})

	t.Run("DeleteAllocation - Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/allocation?id=invalid-uuid", nil)
		rr := httptest.NewRecorder()

		handler.DeleteAllocation(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("DeleteAllocation - Repo Error", func(t *testing.T) {
		allocID := uuid.New()
		allocRepo.deleteFn = func(id uuid.UUID) error {
			return errors.New("delete error")
		}

		req := httptest.NewRequest("DELETE", "/allocation?id="+allocID.String(), nil)
		rr := httptest.NewRecorder()

		handler.DeleteAllocation(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetActiveAllocationsByUserUUID - Success", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		allocRepo.getActiveByUserFn = func(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error) {
			return []*core.Allocation{{AllocatedAmountE5: 100000}}, nil
		}

		req := httptest.NewRequest("GET", "/allocations/active?date=2026-08-08", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.GetActiveAllocationsByUserUUID(rr, req.WithContext(ctx))
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetActiveAllocationsByUserUUID - Missing User UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/allocations/active", nil)
		rr := httptest.NewRecorder()

		handler.GetActiveAllocationsByUserUUID(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetActiveAllocationsByUserUUID - Query Param Fallback", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		allocRepo.getActiveByUserFn = func(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error) {
			return []*core.Allocation{{AllocatedAmountE5: 100000}}, nil
		}

		req := httptest.NewRequest("GET", "/allocations/active?user_uuid="+validUserUUID.String(), nil)
		rr := httptest.NewRecorder()

		handler.GetActiveAllocationsByUserUUID(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetActiveAllocationsByUserUUID - Repo Error", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		allocRepo.getActiveByUserFn = func(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error) {
			return nil, errors.New("db error")
		}

		req := httptest.NewRequest("GET", "/allocations/active", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		handler.GetActiveAllocationsByUserUUID(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetActiveCategoriesByUserUUID - Missing User UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/get-active-categories", nil)
		rr := httptest.NewRecorder()

		handler.GetActiveCategoriesByUserUUID(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetActiveCategoriesByUserUUID - BeginTx Error", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		dbClosed, _, _ := sqlmock.New()
		dbClosed.Close() // closed DB will fail BeginTx
		hClosed := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, txnRepo, shortcutIntentRepo, logger, dbClosed)

		req := httptest.NewRequest("GET", "/api/get-active-categories", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		hClosed.GetActiveCategoriesByUserUUID(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetActiveCategoriesByUserUUID - GetActiveAllocations Error", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		allocRepo.getActiveByUserFn = func(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error) {
			return nil, errors.New("alloc error")
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, txnRepo, shortcutIntentRepo, logger, dbMock)
		req := httptest.NewRequest("GET", "/api/get-active-categories", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.GetActiveCategoriesByUserUUID(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetActiveCategoriesByUserUUID - GetEnvelopeByID Error", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		envID := uuid.New()
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		allocRepo.getActiveByUserFn = func(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error) {
			return []*core.Allocation{{EnvelopeID: envID, AllocatedAmountE5: 5000}}, nil
		}
		envRepo.getByIDFn = func(id uuid.UUID) (*core.Envelope, error) {
			return nil, errors.New("env error")
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, txnRepo, shortcutIntentRepo, logger, dbMock)
		req := httptest.NewRequest("GET", "/api/get-active-categories", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.GetActiveCategoriesByUserUUID(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("GetActiveCategoriesByUserUUID - Success", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		envID := uuid.New()
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		allocRepo.getActiveByUserFn = func(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error) {
			return []*core.Allocation{{EnvelopeID: envID, AllocatedAmountE5: 5000}}, nil
		}
		envRepo.getByIDFn = func(id uuid.UUID) (*core.Envelope, error) {
			return &core.Envelope{ID: envID, Name: "Groceries", CountryISO: "US", Cadence: core.MonthlyCadence}, nil
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, txnRepo, shortcutIntentRepo, logger, dbMock)
		req := httptest.NewRequest("GET", "/api/get-active-categories", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.GetActiveCategoriesByUserUUID(rr, req.WithContext(ctx))
		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBufferString("invalid json"))
		rr := httptest.NewRecorder()

		handler.CreateNewShortcutIntent(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - Missing User UUID", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{"name": "Groceries"})
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		handler.CreateNewShortcutIntent(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - BeginTx Error", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		dbClosed, _, _ := sqlmock.New()
		dbClosed.Close()
		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, txnRepo, shortcutIntentRepo, logger, dbClosed)

		body, _ := json.Marshal(map[string]interface{}{"name": "Groceries"})
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.CreateNewShortcutIntent(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - GetEnvelopeIdByName Error", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		envRepo.getByNameFn = func(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
			return uuid.Nil, errors.New("env name error")
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, txnRepo, shortcutIntentRepo, logger, dbMock)
		body, _ := json.Marshal(map[string]interface{}{"name": "Groceries"})
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.CreateNewShortcutIntent(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - CreateShortcutIntent Error", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		envID := uuid.New()
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		envRepo.getByNameFn = func(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
			return envID, nil
		}
		localShortcutRepo := &mockShortcutIntentRepo{
			createFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("create intent error")
			},
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, txnRepo, localShortcutRepo, logger, dbMock)
		body, _ := json.Marshal(map[string]interface{}{"name": "Groceries"})
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.CreateNewShortcutIntent(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - Success (No Transaction Match)", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		envID := uuid.New()
		intentID := uuid.New()
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectCommit()

		envRepo.getByNameFn = func(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
			return envID, nil
		}
		localShortcutRepo := &mockShortcutIntentRepo{
			createFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
				return intentID, nil
			},
		}
		localTxnRepo := &mockTxnRepo{
			getTransactionByTimeFn: func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
				return nil, sql.ErrNoRows
			},
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, localTxnRepo, localShortcutRepo, logger, dbMock)
		body, _ := json.Marshal(map[string]interface{}{"name": "Groceries", "latitude": 12.97, "longitude": 77.59})
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.CreateNewShortcutIntent(rr, req.WithContext(ctx))
		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - Success (Matched Transaction)", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		envID := uuid.New()
		intentID := uuid.New()
		txnID := uuid.New()
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectCommit()

		envRepo.getByNameFn = func(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
			return envID, nil
		}
		localShortcutRepo := &mockShortcutIntentRepo{
			createFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
				return intentID, nil
			},
			updateFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
				return nil
			},
		}
		localTxnRepo := &mockTxnRepo{
			getTransactionByTimeFn: func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
				return &core.Transaction{ID: txnID}, nil
			},
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, localTxnRepo, localShortcutRepo, logger, dbMock)
		body, _ := json.Marshal(map[string]interface{}{"name": "Groceries", "latitude": 12.97, "longitude": 77.59})
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.CreateNewShortcutIntent(rr, req.WithContext(ctx))
		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - Commit Error (Pending Path)", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		envID := uuid.New()
		intentID := uuid.New()
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectCommit().WillReturnError(errors.New("commit error"))

		envRepo.getByNameFn = func(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
			return envID, nil
		}
		localShortcutRepo := &mockShortcutIntentRepo{
			createFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
				return intentID, nil
			},
		}
		localTxnRepo := &mockTxnRepo{
			getTransactionByTimeFn: func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
				return nil, sql.ErrNoRows
			},
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, localTxnRepo, localShortcutRepo, logger, dbMock)
		body, _ := json.Marshal(map[string]interface{}{"name": "Groceries"})
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.CreateNewShortcutIntent(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - Commit Error (Settled Path)", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		envID := uuid.New()
		intentID := uuid.New()
		txnID := uuid.New()
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectCommit().WillReturnError(errors.New("commit error"))

		envRepo.getByNameFn = func(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
			return envID, nil
		}
		localShortcutRepo := &mockShortcutIntentRepo{
			createFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
				return intentID, nil
			},
			updateFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
				return nil
			},
		}
		localTxnRepo := &mockTxnRepo{
			getTransactionByTimeFn: func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
				return &core.Transaction{ID: txnID}, nil
			},
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, localTxnRepo, localShortcutRepo, logger, dbMock)
		body, _ := json.Marshal(map[string]interface{}{"name": "Groceries"})
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.CreateNewShortcutIntent(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - UpdateTransaction Error", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		envID := uuid.New()
		intentID := uuid.New()
		txnID := uuid.New()
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		envRepo.getByNameFn = func(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
			return envID, nil
		}
		localShortcutRepo := &mockShortcutIntentRepo{
			createFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
				return intentID, nil
			},
		}
		localTxnRepo := &mockTxnRepo{
			getTransactionByTimeFn: func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
				return &core.Transaction{ID: txnID}, nil
			},
			updateTransactionFn: func(txn *core.Transaction) error {
				return errors.New("update txn error")
			},
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, localTxnRepo, localShortcutRepo, logger, dbMock)
		body, _ := json.Marshal(map[string]interface{}{"name": "Groceries"})
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.CreateNewShortcutIntent(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateNewShortcutIntent - UpdateShortcutIntent Error", func(t *testing.T) {
		validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
		envID := uuid.New()
		intentID := uuid.New()
		txnID := uuid.New()
		dbMock, mock, _ := sqlmock.New()
		defer dbMock.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		envRepo.getByNameFn = func(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
			return envID, nil
		}
		localShortcutRepo := &mockShortcutIntentRepo{
			createFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
				return intentID, nil
			},
			updateFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
				return errors.New("update intent error")
			},
		}
		localTxnRepo := &mockTxnRepo{
			getTransactionByTimeFn: func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
				return &core.Transaction{ID: txnID}, nil
			},
		}

		h := NewBudgetingServiceHandler(groupRepo, envRepo, allocRepo, localTxnRepo, localShortcutRepo, logger, dbMock)
		body, _ := json.Marshal(map[string]interface{}{"name": "Groceries"})
		req := httptest.NewRequest("POST", "/api/create-new-intent", bytes.NewBuffer(body))
		ctx := context.WithValue(req.Context(), "user_uuid", validUserUUID)
		rr := httptest.NewRecorder()

		h.CreateNewShortcutIntent(rr, req.WithContext(ctx))
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})
}
