package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserServiceHandler struct {
	userRepo       core.UserRepository
	userTokenRepo  core.TokenRepository
	envGroupRepo   core.EnvelopeGroupRepository
	envelopeRepo   core.EnvelopeRepository
	allocationRepo core.AllocationRepository
	Logger         *zap.Logger
	db             *sql.DB
}

func NewUserServiceHandler(userRepo core.UserRepository, userTokenRepo core.TokenRepository, logger *zap.Logger, envGroupRepo core.EnvelopeGroupRepository, envelopeRepo core.EnvelopeRepository, allocationRepo core.AllocationRepository, db *sql.DB) *UserServiceHandler {
	return &UserServiceHandler{
		userRepo:       userRepo,
		userTokenRepo:  userTokenRepo,
		envGroupRepo:   envGroupRepo,
		envelopeRepo:   envelopeRepo,
		allocationRepo: allocationRepo,
		Logger:         logger,
		db:             db,
	}
}

func (h *UserServiceHandler) GetUserByUUID(w http.ResponseWriter, r *http.Request) {
	var userUUID uuid.UUID
	if val := r.Context().Value("user_uuid"); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			userUUID = id
		} else if idStr, ok := val.(string); ok && idStr != "" {
			userUUID, _ = uuid.Parse(idStr)
		}
	}
	if userUUID == uuid.Nil {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.Logger.Error("Missing user UUID")
		return
	}

	user, err := h.userRepo.GetUserByUUID(userUUID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		h.Logger.Error("User not found", zap.String("user_uuid", userUUID.String()), zap.Error(err))
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

	// create a transaction

	tx, err := h.db.BeginTx(context.Background(), nil)
	if err != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		h.Logger.Error("Failed to begin transaction", zap.Error(err))
		return
	}

	userUUID, err := h.userRepo.CreateUser(&user, tx)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		h.Logger.Error("Failed to create user", zap.Error(err))
		tx.Rollback()
		return
	}

	now := time.Now()

	//create a new envGroup for this user with is_system = true (unallocated budget)

	envGroup := core.EnvelopeGroup{
		ID:        uuid.New(),
		UserUUID:  userUUID,
		Name:      "Unallocated Budget",
		IsSystem:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	envGroupUUID, err := h.envGroupRepo.CreateEnvelopeGroup(&envGroup, tx)
	if err != nil {
		h.Logger.Error("Failed to create envelope group", zap.Error(err))
		tx.Rollback()
		http.Error(w, "Failed to create envelope group", http.StatusInternalServerError)
		return
	}

	// create a new env for this user with unallocated budget group

	env := core.Envelope{
		UserUUID:        userUUID,
		EnvelopeGroupID: envGroupUUID,
		TargetAmountE5:  0,
		Cadence:         "monthly",
		CountryISO:      "IN",
		IsSystem:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	envUUID, err := h.envelopeRepo.CreateEnvelope(&env, tx)
	if err != nil {
		http.Error(w, "Failed to create envelope ", http.StatusInternalServerError)
		h.Logger.Error("Failed to create envelope", zap.Error(err))
		tx.Rollback()
		return
	}

	// create a new allocation from now to 100 years from now with 0 amount

	endDate := now.AddDate(100, 0, 0)
	alloc := core.Allocation{
		EnvelopeID:        envUUID,
		AllocatedAmountE5: 0,
		StartDate:         &now,
		EndDate:           &endDate,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if _, err = h.allocationRepo.CreateAllocation(&alloc, tx); err != nil {
		http.Error(w, "Failed to create allocation", http.StatusInternalServerError)
		h.Logger.Error("Failed to create allocation", zap.Error(err))
		tx.Rollback()
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		h.Logger.Error("Failed to commit transaction", zap.Error(err))
		return
	}
	var token core.Token
	token.UserUUID = userUUID
	token.Prefix = core.AuthToken
	token.Name = core.DefaultName
	token.Scope = []string{"all"}
	token.ExpiresAt = nil
	token.LastUsedAt = nil
	token.CreatedAt = now
	token.UpdatedAt = now

	userAuthToken, err := h.userTokenRepo.CreateToken(&token, nil)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		h.Logger.Error("Failed to create token", zap.Error(err))
		return
	}

	type res struct {
		UserAuthToken string `json:"user_auth_token"`
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res{UserAuthToken: userAuthToken.String()})
}
