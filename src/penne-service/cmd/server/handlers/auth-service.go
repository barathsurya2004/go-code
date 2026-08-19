package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuthServiceHandler struct {
	log               *zap.Logger
	userRepo          core.UserRepository
	tokenRepo         core.TokenRepository
	envelopeGroupRepo core.EnvelopeGroupRepository
	envelopeRepo      core.EnvelopeRepository
	allocationRepo    core.AllocationRepository
	db                *sql.DB
}

func NewAuthServiceHandler(userRepo core.UserRepository, tokenRepo core.TokenRepository, envelopeGroupRepo core.EnvelopeGroupRepository, envelopeRepo core.EnvelopeRepository, allocationRepo core.AllocationRepository, logger *zap.Logger, db *sql.DB) *AuthServiceHandler {
	return &AuthServiceHandler{
		userRepo:          userRepo,
		tokenRepo:         tokenRepo,
		envelopeGroupRepo: envelopeGroupRepo,
		envelopeRepo:      envelopeRepo,
		allocationRepo:    allocationRepo,
		log:               logger,
		db:                db,
	}
}

func (h *AuthServiceHandler) Login(w http.ResponseWriter, r *http.Request) {
	// so first we will get the user using getUsersByEmail
	// and compare the password with the hashed password
	// if correct then we will do a token get or create token
	// if token already exists and not expired then we will return the token
	// else we will create a new token and return it
	type loginRequest struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		RememberMe *bool  `json:"remember_me"`
	}

	var loginReq loginRequest

	// decode the request body
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.log.Error("Failed to decode login request payload", zap.Error(err))
		return
	}

	// check for email in db
	user, err := h.userRepo.GetUserByEmail(loginReq.Email)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.log.Error("Failed to get user by email", zap.Error(err))
		return
	}

	// check for password
	if ok, err := utils.ComparePasswords(loginReq.Password, user.PasswordHash); !ok || err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.log.Error("Invalid password", zap.Error(err))
		return
	}

	token, err := h.tokenRepo.GetActiveTokenWithUserUUID(user.UUID)
	if token == nil || err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.log.Error("Failed to get token by user UUID", zap.Error(err))
		return
	}
	// for now skipping the expiry check since all tokens are permanent
	json.NewEncoder(w).Encode(struct {
		Token string `json:"token"`
	}{Token: token.Token.String()})

}

func (h *AuthServiceHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	type signUpRequest struct {
		Email       string `json:"email"`
		Name        string `json:"name"`
		Password    string `json:"password"`
		CountryIso2 string `json:"country_iso2"`
	}

	var signUpReq signUpRequest
	if err := json.NewDecoder(r.Body).Decode(&signUpReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.log.Error("Failed to decode sign up request payload", zap.Error(err))
		return
	}

	//check if user already present
	user, err := h.userRepo.GetUserByEmail(signUpReq.Email)

	var authToken uuid.UUID

	if err != nil {
		//create a new user
		passwordHash, err := utils.CreatePassword(signUpReq.Password)
		if err != nil {
			http.Error(w, "Invalid request payload", http.StatusInternalServerError)
			h.log.Error("Failed to create password", zap.Error(err))
			return
		}
		usr := core.User{
			Name:         signUpReq.Name,
			Email:        signUpReq.Email,
			PasswordHash: passwordHash,
		}
		user = &usr
		tx, err := h.db.BeginTx(r.Context(), nil)
		if err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			h.log.Error("Failed to begin transaction", zap.Error(err))
			return
		}
		defer tx.Rollback()
		userUUID, err := h.userRepo.CreateUser(&usr, tx)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			h.log.Error("Failed to create user", zap.Error(err))
			return
		}
		user.UUID = userUUID
		now := utils.NowUTC()
		// create envelope group
		envelopeGroup := core.EnvelopeGroup{
			UserUUID:  userUUID,
			Name:      core.DefaultName,
			IsSystem:  true,
			CreatedAt: now,
			UpdatedAt: now,
		}
		envGroupID, err := h.envelopeGroupRepo.CreateEnvelopeGroup(&envelopeGroup, tx)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			h.log.Error("Failed to create envelope group", zap.Error(err))
			return
		}
		// create default envelopes
		envelope := core.Envelope{
			UserUUID:        user.UUID,
			EnvelopeGroupID: envGroupID,
			TargetAmountE5:  0,
			Name:            core.DefaultName,
			Cadence:         "forever",
			CountryISO:      "IN",
			CreatedAt:       now,
			UpdatedAt:       now,
			IsSystem:        true,
		}

		envID, err := h.envelopeRepo.CreateEnvelope(&envelope, tx)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			h.log.Error("Failed to create envelope", zap.Error(err))
			return
		}
		// create default allocation
		today := now
		oneHundredYearFromNow := today.AddDate(100, 0, 0)
		allocation := core.Allocation{
			EnvelopeID:        envID,
			AllocatedAmountE5: 0,
			StartDate:         &today,
			EndDate:           &oneHundredYearFromNow,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		if _, err := h.allocationRepo.CreateAllocation(&allocation, tx); err != nil {
			tx.Rollback()
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			h.log.Error("Failed to create allocation", zap.Error(err))
			return
		}

		// create token
		lastUsedAt := now
		token := core.Token{
			UserUUID:   userUUID,
			Prefix:     core.AuthToken,
			Name:       core.DefaultName,
			Scope:      []string{"read", "write"},
			ExpiresAt:  nil,
			LastUsedAt: &lastUsedAt,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if authToken, err = h.tokenRepo.CreateToken(&token, tx); err != nil {
			tx.Rollback()
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			h.log.Error("Failed to create token", zap.Error(err))
			return
		}

		if err = tx.Commit(); err != nil {
			http.Error(w, "Invalid request payload", http.StatusInternalServerError)
			h.log.Error("Failed to commit transaction", zap.Error(err))
			return
		}
	} else {
		// check passwords

		if ok, err := utils.ComparePasswords(signUpReq.Password, user.PasswordHash); !ok || err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Incorrect email or password",
			})
			return
		}

		Token, err := h.tokenRepo.GetActiveTokenWithUserUUID(user.UUID)
		if err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			h.log.Error("Failed to get token by user UUID", zap.Error(err))
			return
		}
		authToken = Token.Token
	}
	json.NewEncoder(w).Encode(struct {
		Token string `json:"token"`
	}{Token: authToken.String()})

}
