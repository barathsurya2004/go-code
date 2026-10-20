package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/pulse-service/internal/core"
	"github.com/barathsurya2004/go-code/pulse-service/internal/db"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockRepo struct {
	saveFn         func(payload *core.SyncPayload) (*core.SyncResponse, error)
	statsFn        func(userUUID string) (*core.PulseUserStats, error)
	deleteFn       func(userUUID string) error
	createUserFn   func(req *core.CreateUserRequest) (*core.PulseUser, error)
	getUserEmailFn func(email string) (*core.PulseUser, error)
	getUserUUIDFn  func(uuid string) (*core.PulseUser, error)
	authFn         func(email, password string) (*core.PulseUser, error)
}

func (m *mockRepo) SaveSyncPayload(ctx context.Context, payload *core.SyncPayload) (*core.SyncResponse, error) {
	if m.saveFn != nil {
		return m.saveFn(payload)
	}
	return &core.SyncResponse{Status: "ok"}, nil
}

func (m *mockRepo) GetUserStats(ctx context.Context, userUUID string) (*core.PulseUserStats, error) {
	if m.statsFn != nil {
		return m.statsFn(userUUID)
	}
	return &core.PulseUserStats{UserUUID: userUUID}, nil
}

func (m *mockRepo) DeleteUserData(ctx context.Context, userUUID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(userUUID)
	}
	return nil
}

func (m *mockRepo) CreateUser(ctx context.Context, req *core.CreateUserRequest) (*core.PulseUser, error) {
	if m.createUserFn != nil {
		return m.createUserFn(req)
	}
	return &core.PulseUser{
		ID:        "u-123",
		UserUUID:  "u-123",
		Email:     req.Email,
		Name:      req.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockRepo) GetUserByEmail(ctx context.Context, email string) (*core.PulseUser, error) {
	if m.getUserEmailFn != nil {
		return m.getUserEmailFn(email)
	}
	return &core.PulseUser{ID: "u-123", UserUUID: "u-123", Email: email}, nil
}

func (m *mockRepo) GetUserByUUID(ctx context.Context, userUUID string) (*core.PulseUser, error) {
	if m.getUserUUIDFn != nil {
		return m.getUserUUIDFn(userUUID)
	}
	return &core.PulseUser{ID: userUUID, UserUUID: userUUID, Name: "Athlete"}, nil
}

func (m *mockRepo) AuthenticateUser(ctx context.Context, email, password string) (*core.PulseUser, error) {
	if m.authFn != nil {
		return m.authFn(email, password)
	}
	return &core.PulseUser{ID: "u-123", UserUUID: "u-123", Email: email}, nil
}

func (m *mockRepo) Close() error {
	return nil
}

func TestPulseHandler(t *testing.T) {
	log := zap.NewNop()
	repo := db.NewMemoryPulseRepository(log)
	handler := NewPulseHandler(repo, log)
	router := NewRouter(handler)

	t.Run("HandleHealth returns 200 UP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]string
		assert.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
		assert.Equal(t, "UP", body["status"])
		assert.Equal(t, "pulse-service", body["service"])
	})

	t.Run("HandleSync invalid method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/pulse/sync", nil)
		rec := httptest.NewRecorder()
		handler.HandleSync(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})

	t.Run("HandleSync invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/sync", bytes.NewBufferString("invalid json"))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("HandleSync missing auth and user_uuid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/sync", bytes.NewBufferString("{}"))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("HandleSync with header x-user-uuid", func(t *testing.T) {
		payload := core.SyncPayload{
			ClientTime: 12345,
			Settings: &core.Settings{
				Unit: "kg",
			},
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/sync", bytes.NewReader(data))
		req.Header.Set("x-user-uuid", "u-header-1")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp core.SyncResponse
		assert.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("HandleSync with Authorization Bearer header", func(t *testing.T) {
		payload := core.SyncPayload{
			ClientTime: 12345,
		}
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer u-bearer-1")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("HandleSync repo error", func(t *testing.T) {
		errRepo := &mockRepo{
			saveFn: func(payload *core.SyncPayload) (*core.SyncResponse, error) {
				return nil, errors.New("database locked")
			},
		}
		errHandler := NewPulseHandler(errRepo, log)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/sync", bytes.NewBufferString(`{"user_uuid":"u-1"}`))
		rec := httptest.NewRecorder()
		errHandler.HandleSync(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("HandleGetStats missing user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/pulse/stats", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("HandleGetStats with query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/pulse/stats?user_uuid=u-header-1", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var stats core.PulseUserStats
		assert.NoError(t, json.NewDecoder(rec.Body).Decode(&stats))
		assert.Equal(t, "u-header-1", stats.UserUUID)
	})

	t.Run("HandleGetStats with Authorization header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/stats", nil)
		req.Header.Set("Authorization", "Bearer u-header-1")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("HandleGetStats repo error", func(t *testing.T) {
		errRepo := &mockRepo{
			statsFn: func(userUUID string) (*core.PulseUserStats, error) {
				return nil, errors.New("db error")
			},
		}
		errHandler := NewPulseHandler(errRepo, log)

		req := httptest.NewRequest(http.MethodGet, "/stats?user_uuid=u-1", nil)
		rec := httptest.NewRecorder()
		errHandler.HandleGetStats(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("CORSMiddleware preflight OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/pulse/sync", nil)
		req.Header.Set("Origin", "http://localhost:5175")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "http://localhost:5175", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	// ── User Management & Auth Tests ──
	t.Run("HandleCreateUser success", func(t *testing.T) {
		reqBody := core.CreateUserRequest{
			Name:     "Test Athlete",
			Email:    "athlete@pulse.test",
			Password: "password123",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/users", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		var resp core.AuthResponse
		assert.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		assert.NotEmpty(t, resp.Token)
		assert.Equal(t, "athlete@pulse.test", resp.User.Email)
	})

	t.Run("HandleCreateUser missing email", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/users", bytes.NewBufferString(`{"name":"No Email"}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("HandleCreateUser already exists", func(t *testing.T) {
		// First create
		reqBody := core.CreateUserRequest{
			Name:  "Dup Athlete",
			Email: "dup@pulse.test",
		}
		body, _ := json.Marshal(reqBody)
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/users", bytes.NewReader(body))
		rec1 := httptest.NewRecorder()
		router.ServeHTTP(rec1, req1)
		assert.Equal(t, http.StatusCreated, rec1.Code)

		// Second create with same email
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/users", bytes.NewReader(body))
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)
		assert.Equal(t, http.StatusConflict, rec2.Code)
	})

	t.Run("HandleSignup success and duplicate", func(t *testing.T) {
		reqBody := core.CreateUserRequest{
			Name:     "New Runner",
			Email:    "runner@pulse.test",
			Password: "securepass",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/auth/signup", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		// Duplicate signup
		reqDup := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/auth/signup", bytes.NewReader(body))
		recDup := httptest.NewRecorder()
		router.ServeHTTP(recDup, reqDup)
		assert.Equal(t, http.StatusConflict, recDup.Code)
	})

	t.Run("HandleSignup missing fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/auth/signup", bytes.NewBufferString(`{"email":""}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("HandleLogin success and failure", func(t *testing.T) {
		// First register via signup
		reqBody := core.CreateUserRequest{
			Name:     "Login Test",
			Email:    "logintest@pulse.test",
			Password: "mypassword",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/auth/signup", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code)

		// Login success
		authReq := core.AuthRequest{
			Email:    "logintest@pulse.test",
			Password: "mypassword",
		}
		authBody, _ := json.Marshal(authReq)
		reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/auth/login", bytes.NewReader(authBody))
		recLogin := httptest.NewRecorder()
		router.ServeHTTP(recLogin, reqLogin)
		assert.Equal(t, http.StatusOK, recLogin.Code)

		// Login bad password
		badReq := core.AuthRequest{
			Email:    "logintest@pulse.test",
			Password: "wrongpassword",
		}
		badBody, _ := json.Marshal(badReq)
		reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/pulse/auth/login", bytes.NewReader(badBody))
		recBad := httptest.NewRecorder()
		router.ServeHTTP(recBad, reqBad)
		assert.Equal(t, http.StatusUnauthorized, recBad.Code)
	})

	t.Run("HandleGetUser by UUID and fallback", func(t *testing.T) {
		// Existing user
		req := httptest.NewRequest(http.MethodGet, "/api/v1/pulse/user", nil)
		req.Header.Set("x-user-uuid", "u-123")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Unknown token fallback profile
		reqUnknown := httptest.NewRequest(http.MethodGet, "/api/v1/pulse/user", nil)
		reqUnknown.Header.Set("Authorization", "Bearer unknown-token-456")
		recUnknown := httptest.NewRecorder()
		router.ServeHTTP(recUnknown, reqUnknown)

		assert.Equal(t, http.StatusOK, recUnknown.Code)
		var u core.PulseUser
		assert.NoError(t, json.NewDecoder(recUnknown.Body).Decode(&u))
		assert.Equal(t, "unknown-token-456", u.UserUUID)
	})

	t.Run("HandleDeleteUserData purge", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/pulse/sync", nil)
		req.Header.Set("x-user-uuid", "athlete@pulse.test")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
