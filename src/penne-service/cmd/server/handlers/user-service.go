package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/zap"
)

type UserServiceHandler struct {
	userRepo core.UserRepository
	Logger   *zap.Logger
}

func NewUserServiceHandler(userRepo core.UserRepository, logger *zap.Logger) *UserServiceHandler {
	return &UserServiceHandler{
		userRepo: userRepo,
		Logger:   logger,
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
	json.NewEncoder(w).Encode(user)
	w.WriteHeader(http.StatusOK)

}

func (h *UserServiceHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user core.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.Logger.Error("Failed to decode user payload", zap.Error(err))
		return
	}

	if err := h.userRepo.CreateUser(&user); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		h.Logger.Error("Failed to create user", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
	w.WriteHeader(http.StatusCreated)
}
