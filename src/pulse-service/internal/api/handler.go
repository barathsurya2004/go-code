package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/barathsurya2004/go-code/pulse-service/internal/core"
	"github.com/barathsurya2004/go-code/pulse-service/internal/db"
	"go.uber.org/zap"
)

type PulseHandler struct {
	repo db.PulseRepository
	log  *zap.Logger
}

func NewPulseHandler(repo db.PulseRepository, log *zap.Logger) *PulseHandler {
	return &PulseHandler{
		repo: repo,
		log:  log,
	}
}

func (h *PulseHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "UP",
		"service": "pulse-service",
	})
}

func (h *PulseHandler) HandleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var payload core.SyncPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.log.Warn("Failed to decode sync payload", zap.Error(err))
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Extract user UUID from headers if not specified in body
	userUUID := payload.UserUUID
	if userUUID == "" {
		userUUID = r.Header.Get("x-user-uuid")
	}
	if userUUID == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			userUUID = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if userUUID == "" {
		writeError(w, http.StatusUnauthorized, "user_uuid or Authorization header is required")
		return
	}

	payload.UserUUID = userUUID

	resp, err := h.repo.SaveSyncPayload(r.Context(), &payload)
	if err != nil {
		h.log.Error("Failed to save sync payload", zap.Error(err), zap.String("user_uuid", userUUID))
		writeError(w, http.StatusInternalServerError, "failed to persist sync payload: "+err.Error())
		return
	}

	h.log.Info("Successfully synced pulse data",
		zap.String("user_uuid", userUUID),
		zap.Int("sessions", len(payload.Sessions)),
		zap.Int("routines", len(payload.Routines)),
	)

	writeJSON(w, http.StatusOK, resp)
}

func (h *PulseHandler) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	userUUID := r.URL.Query().Get("user_uuid")
	if userUUID == "" {
		userUUID = r.Header.Get("x-user-uuid")
	}
	if userUUID == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			userUUID = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if userUUID == "" {
		writeError(w, http.StatusBadRequest, "user_uuid query parameter or header is required")
		return
	}

	stats, err := h.repo.GetUserStats(r.Context(), userUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user stats: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
