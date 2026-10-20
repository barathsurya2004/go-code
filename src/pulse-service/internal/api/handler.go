package api

import (
	"encoding/json"
	"errors"
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

// HandleCreateUser creates or provisions a Pulse user
func (h *PulseHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req core.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	user, err := h.repo.CreateUser(r.Context(), &req)
	if err != nil {
		if errors.Is(err, db.ErrUserAlreadyExists) {
			writeError(w, http.StatusConflict, "user with this email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user: "+err.Error())
		return
	}

	resp := core.AuthResponse{
		Token:    user.UserUUID,
		UserUUID: user.UserUUID,
		User:     user,
		Message:  "User created successfully",
	}
	writeJSON(w, http.StatusCreated, resp)
}

// HandleSignup handles user registration for Pulse
func (h *PulseHandler) HandleSignup(w http.ResponseWriter, r *http.Request) {
	var req core.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Email == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "name and email are required")
		return
	}

	user, err := h.repo.CreateUser(r.Context(), &req)
	if err != nil {
		if errors.Is(err, db.ErrUserAlreadyExists) {
			writeError(w, http.StatusConflict, "user with this email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to register user: "+err.Error())
		return
	}

	resp := core.AuthResponse{
		Token:    user.UserUUID,
		UserUUID: user.UserUUID,
		User:     user,
		Message:  "Registration successful",
	}
	writeJSON(w, http.StatusCreated, resp)
}

// HandleLogin handles user login for Pulse
func (h *PulseHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req core.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	user, err := h.repo.AuthenticateUser(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	resp := core.AuthResponse{
		Token:    user.UserUUID,
		UserUUID: user.UserUUID,
		User:     user,
		Message:  "Login successful",
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleGetUser returns the profile for the authenticated Pulse user
func (h *PulseHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	userUUID := r.Header.Get("x-user-uuid")
	if userUUID == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			userUUID = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if userUUID == "" {
		userUUID = r.URL.Query().Get("user_uuid")
	}

	if userUUID == "" {
		writeError(w, http.StatusUnauthorized, "missing authentication token or x-user-uuid")
		return
	}

	user, err := h.repo.GetUserByUUID(r.Context(), userUUID)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			// If not found in pulse_users table, return a basic user profile from the token
			writeJSON(w, http.StatusOK, &core.PulseUser{
				ID:       userUUID,
				UserUUID: userUUID,
				Name:     "Athlete",
				Email:    "",
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, user)
}

// HandleDeleteUserData wipes data for testing/reset
func (h *PulseHandler) HandleDeleteUserData(w http.ResponseWriter, r *http.Request) {
	userUUID := r.Header.Get("x-user-uuid")
	if userUUID == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			userUUID = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if userUUID == "" {
		userUUID = r.URL.Query().Get("user_uuid")
	}

	if userUUID == "" {
		writeError(w, http.StatusBadRequest, "user_uuid is required")
		return
	}

	if err := h.repo.DeleteUserData(r.Context(), userUUID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete user data: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"message":   "User data successfully purged",
		"user_uuid": userUUID,
	})
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
