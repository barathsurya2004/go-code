package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserServiceHandler struct {
	userRepo      core.UserRepository
	userTokenRepo core.TokenRepository
	Logger        *zap.Logger
}

func NewUserServiceHandler(userRepo core.UserRepository, userTokenRepo core.TokenRepository, logger *zap.Logger) *UserServiceHandler {
	return &UserServiceHandler{
		userRepo:      userRepo,
		userTokenRepo: userTokenRepo,
		Logger:        logger,
	}
}

func (h *UserServiceHandler) GetUserByUUID(w http.ResponseWriter, r *http.Request) {
	userUUID := r.URL.Query().Get("user_uuid")
	if userUUID == "" {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.Logger.Error("Missing user UUID")
		return
	}

	user, err := h.userRepo.GetUserByUUID(userUUID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		h.Logger.Error("User not found", zap.String("user_uuid", userUUID), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

func (h *UserServiceHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user core.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.Logger.Error("Failed to decode user payload", zap.Error(err))
		return
	}

	if user.UUID == "" {
		user.UUID = uuid.NewString()
	}

	if err := h.userRepo.CreateUser(&user); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		h.Logger.Error("Failed to create user", zap.Error(err))
		return
	}

	now := time.Now()
	var token core.Token
	token.UserUUID = user.UUID
	token.Prefix = "mcp_"
	token.Name = "default"
	token.Scope = []string{"all"}
	token.ExpiresAt = nil
	token.LastUsedAt = nil
	token.CreatedAt = now
	token.UpdatedAt = now

	if err := h.userTokenRepo.CreateToken(&token); err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		h.Logger.Error("Failed to create token", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}
