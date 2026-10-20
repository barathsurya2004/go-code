package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/barathsurya2004/go-code/pulse-service/internal/core"
	"github.com/barathsurya2004/go-code/pulse-service/internal/db"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockRepo struct {
	saveFn  func(payload *core.SyncPayload) (*core.SyncResponse, error)
	statsFn func(userUUID string) (*core.PulseUserStats, error)
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
}
