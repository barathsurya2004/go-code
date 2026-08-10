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
	"go.uber.org/zap"
)

type mockUserRepo struct {
	createUserFn     func(user *core.User) (uuid.UUID, error)
	getUserByUUIDFn  func(id uuid.UUID) (*core.User, error)
	getUserByEmailFn func(email string) (*core.User, error)
}

func (m *mockUserRepo) CreateUser(user *core.User, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createUserFn != nil {
		return m.createUserFn(user)
	}
	return uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), nil
}

func (m *mockUserRepo) GetUserByUUID(id uuid.UUID) (*core.User, error) {
	if m.getUserByUUIDFn != nil {
		return m.getUserByUUIDFn(id)
	}
	return nil, nil
}

func (m *mockUserRepo) GetUserByEmail(email string) (*core.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(email)
	}
	return nil, nil
}

type mockTokenRepo struct {
	createTokenFn                func(token *core.Token) (uuid.UUID, error)
	deleteTokenFn                func(userUUID uuid.UUID) error
	getTokenFn                   func(token uuid.UUID) (*core.Token, error)
	getActiveTokenWithUserUUIDFn func(userUUID uuid.UUID) (*core.Token, error)
	updateTokenFn                func(token *core.Token) error
}

func (m *mockTokenRepo) CreateToken(token *core.Token, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createTokenFn != nil {
		return m.createTokenFn(token)
	}
	return uuid.Nil, nil
}

func (m *mockTokenRepo) DeleteToken(userUUID uuid.UUID) error {
	if m.deleteTokenFn != nil {
		return m.deleteTokenFn(userUUID)
	}
	return nil
}

func (m *mockTokenRepo) GetToken(token uuid.UUID) (*core.Token, error) {
	if m.getTokenFn != nil {
		return m.getTokenFn(token)
	}
	return nil, nil
}

func (m *mockTokenRepo) GetActiveTokenWithUserUUID(userUUID uuid.UUID) (*core.Token, error) {
	if m.getActiveTokenWithUserUUIDFn != nil {
		return m.getActiveTokenWithUserUUIDFn(userUUID)
	}
	return nil, nil
}

func (m *mockTokenRepo) UpdateToken(token *core.Token) error {
	if m.updateTokenFn != nil {
		return m.updateTokenFn(token)
	}
	return nil
}

type dummyEnvelopeGroupRepo struct {
	createFn func(envelopeGroup *core.EnvelopeGroup) (uuid.UUID, error)
}

func (d *dummyEnvelopeGroupRepo) CreateEnvelopeGroup(envelopeGroup *core.EnvelopeGroup, Tx *sql.Tx) (uuid.UUID, error) {
	if d.createFn != nil {
		return d.createFn(envelopeGroup)
	}
	return envelopeGroup.ID, nil
}
func (d *dummyEnvelopeGroupRepo) GetEnvelopeGroupByID(id uuid.UUID) (*core.EnvelopeGroup, error) {
	return nil, nil
}
func (d *dummyEnvelopeGroupRepo) GetEnvelopeGroupsByUserUUID(userUUID uuid.UUID) ([]*core.EnvelopeGroup, error) {
	return nil, nil
}
func (d *dummyEnvelopeGroupRepo) UpdateEnvelopeGroup(envelopeGroup *core.EnvelopeGroup) error {
	return nil
}
func (d *dummyEnvelopeGroupRepo) DeleteEnvelopeGroup(id uuid.UUID) error { return nil }

type dummyEnvelopeRepo struct {
	createFn func(envelope *core.Envelope) (uuid.UUID, error)
}

func (d *dummyEnvelopeRepo) CreateEnvelope(envelope *core.Envelope, Tx *sql.Tx) (uuid.UUID, error) {
	if d.createFn != nil {
		return d.createFn(envelope)
	}
	return envelope.ID, nil
}
func (d *dummyEnvelopeRepo) GetEnvelopeByID(id uuid.UUID) (*core.Envelope, error) { return nil, nil }
func (d *dummyEnvelopeRepo) GetEnvelopesByUserUUID(userUUID uuid.UUID) ([]*core.Envelope, error) {
	return nil, nil
}
func (d *dummyEnvelopeRepo) UpdateEnvelope(envelope *core.Envelope) error { return nil }
func (d *dummyEnvelopeRepo) DeleteEnvelope(id uuid.UUID) error            { return nil }

type dummyAllocationRepo struct {
	createFn func(allocation *core.Allocation) (uuid.UUID, error)
}

func (d *dummyAllocationRepo) CreateAllocation(allocation *core.Allocation, Tx *sql.Tx) (uuid.UUID, error) {
	if d.createFn != nil {
		return d.createFn(allocation)
	}
	return allocation.ID, nil
}
func (d *dummyAllocationRepo) GetAllocationByID(id uuid.UUID) (*core.Allocation, error) {
	return nil, nil
}
func (d *dummyAllocationRepo) GetAllocationsByEnvelopeID(envelopeID uuid.UUID) ([]*core.Allocation, error) {
	return nil, nil
}
func (d *dummyAllocationRepo) GetActiveAllocationsByUserUUID(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error) {
	return nil, nil
}
func (d *dummyAllocationRepo) UpdateAllocation(allocation *core.Allocation) error { return nil }
func (d *dummyAllocationRepo) DeleteAllocation(id uuid.UUID) error                { return nil }

func TestUserServiceHandler(t *testing.T) {
	logger := zap.NewNop()
	userRepo := &mockUserRepo{}
	tokenRepo := &mockTokenRepo{}
	envGroupRepo := &dummyEnvelopeGroupRepo{}
	envRepo := &dummyEnvelopeRepo{}
	allocRepo := &dummyAllocationRepo{}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewUserServiceHandler(userRepo, tokenRepo, logger, envGroupRepo, envRepo, allocRepo, db)
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("GetUserByUUID - Missing UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/user", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", uuid.Nil)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.GetUserByUUID(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetUserByUUID - Repo Error", func(t *testing.T) {
		userRepo.getUserByUUIDFn = func(id uuid.UUID) (*core.User, error) {
			return nil, errors.New("user not found")
		}
		req := httptest.NewRequest("GET", "/user", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUUID)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.GetUserByUUID(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("GetUserByUUID - Success", func(t *testing.T) {
		userRepo.getUserByUUIDFn = func(id uuid.UUID) (*core.User, error) {
			return &core.User{UUID: id, Name: "Alice"}, nil
		}
		req := httptest.NewRequest("GET", "/user", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUUID)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.GetUserByUUID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetUserByUUID - String Context UUID", func(t *testing.T) {
		userRepo.getUserByUUIDFn = func(id uuid.UUID) (*core.User, error) {
			return &core.User{UUID: id, Name: "Alice"}, nil
		}
		req := httptest.NewRequest("GET", "/user", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUUID.String())
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.GetUserByUUID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GetUserByUUID - Success", func(t *testing.T) {
		expectedUser := &core.User{
			UUID: validUUID,
			Name: "John Doe",
		}
		userRepo.getUserByUUIDFn = func(id uuid.UUID) (*core.User, error) {
			return expectedUser, nil
		}
		req := httptest.NewRequest("GET", "/user", nil)
		ctx := context.WithValue(req.Context(), "user_uuid", validUUID)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.GetUserByUUID(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("CreateUser - Invalid Payload", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString("invalid json"))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("CreateUser - BeginTx Error", func(t *testing.T) {
		mock.ExpectBegin().WillReturnError(errors.New("tx begin error"))
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString(`{"name":"Jane Doe"}`))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateUser - User Repo Error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()
		userRepo.createUserFn = func(user *core.User) (uuid.UUID, error) {
			return uuid.Nil, errors.New("db error")
		}
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString(`{"name":"Jane Doe"}`))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateUser - EnvelopeGroup Repo Error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()
		userRepo.createUserFn = func(user *core.User) (uuid.UUID, error) {
			return validUUID, nil
		}
		envGroupRepo.createFn = func(g *core.EnvelopeGroup) (uuid.UUID, error) {
			return uuid.Nil, errors.New("env group error")
		}
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString(`{"name":"Jane Doe"}`))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
		envGroupRepo.createFn = nil
	})

	t.Run("CreateUser - Envelope Repo Error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()
		userRepo.createUserFn = func(user *core.User) (uuid.UUID, error) {
			return validUUID, nil
		}
		envRepo.createFn = func(env *core.Envelope) (uuid.UUID, error) {
			return uuid.Nil, errors.New("env error")
		}
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString(`{"name":"Jane Doe"}`))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
		envRepo.createFn = nil
	})

	t.Run("CreateUser - Allocation Repo Error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()
		userRepo.createUserFn = func(user *core.User) (uuid.UUID, error) {
			return validUUID, nil
		}
		allocRepo.createFn = func(alloc *core.Allocation) (uuid.UUID, error) {
			return uuid.Nil, errors.New("alloc error")
		}
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString(`{"name":"Jane Doe"}`))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
		allocRepo.createFn = nil
	})

	t.Run("CreateUser - Commit Error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit().WillReturnError(errors.New("commit error"))
		userRepo.createUserFn = func(user *core.User) (uuid.UUID, error) {
			return validUUID, nil
		}
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString(`{"name":"Jane Doe"}`))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateUser - Token Repo Error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit()
		userRepo.createUserFn = func(user *core.User) (uuid.UUID, error) {
			return validUUID, nil
		}
		tokenRepo.createTokenFn = func(token *core.Token) (uuid.UUID, error) {
			return uuid.Nil, errors.New("failed to create token")
		}
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString(`{"name":"Jane Doe"}`))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateUser - Success", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit()
		userRepo.createUserFn = func(user *core.User) (uuid.UUID, error) {
			return validUUID, nil
		}
		tokenRepo.createTokenFn = func(token *core.Token) (uuid.UUID, error) {
			return uuid.MustParse("87654321-e89b-12d3-a456-426614174000"), nil
		}
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString(`{"name":"Jane Doe"}`))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})
}
