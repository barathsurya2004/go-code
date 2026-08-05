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

type mockUserRepo struct {
	createUserFn    func(user *core.User) error
	getUserByUUIDFn func(uuid string) (*core.User, error)
}

func (m *mockUserRepo) CreateUser(user *core.User) error {
	if m.createUserFn != nil {
		return m.createUserFn(user)
	}
	return nil
}

func (m *mockUserRepo) GetUserByUUID(uuid string) (*core.User, error) {
	if m.getUserByUUIDFn != nil {
		return m.getUserByUUIDFn(uuid)
	}
	return nil, nil
}

func TestUserServiceHandler(t *testing.T) {
	logger := zap.NewNop()
	repo := &mockUserRepo{}
	handler := NewUserServiceHandler(repo, logger)

	t.Run("GetUserByUUID - Missing UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/user", nil)
		rr := httptest.NewRecorder()

		handler.GetUserByUUID(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("GetUserByUUID - Repo Error", func(t *testing.T) {
		repo.getUserByUUIDFn = func(uuid string) (*core.User, error) {
			return nil, errors.New("user not found")
		}
		req := httptest.NewRequest("GET", "/user?user_uuid=123e4567-e89b-12d3-a456-426614174000", nil)
		rr := httptest.NewRecorder()

		handler.GetUserByUUID(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("GetUserByUUID - Success", func(t *testing.T) {
		expectedUser := &core.User{
			UUID: "123e4567-e89b-12d3-a456-426614174000",
			Name: "John Doe",
		}
		repo.getUserByUUIDFn = func(uuid string) (*core.User, error) {
			return expectedUser, nil
		}
		req := httptest.NewRequest("GET", "/user?user_uuid=123e4567-e89b-12d3-a456-426614174000", nil)
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

	t.Run("CreateUser - Repo Error", func(t *testing.T) {
		repo.createUserFn = func(user *core.User) error {
			return errors.New("db error")
		}
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString(`{"name":"Jane Doe"}`))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("CreateUser - Success", func(t *testing.T) {
		repo.createUserFn = func(user *core.User) error {
			return nil
		}
		req := httptest.NewRequest("POST", "/user", bytes.NewBufferString(`{"name":"Jane Doe"}`))
		rr := httptest.NewRecorder()

		handler.CreateUser(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
		}
	})
}
