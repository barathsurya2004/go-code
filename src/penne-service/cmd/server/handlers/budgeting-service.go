package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type BudgetingServiceHandler struct {
	envelopeGroupRepo  core.EnvelopeGroupRepository
	envelopeRepo       core.EnvelopeRepository
	allocationRepo     core.AllocationRepository
	ShortcutIntentRepo core.ShortcutIntentRepository
	TransactionRepo    core.TransactionRepository
	logger             *zap.Logger
	db                 *sql.DB
}

func NewBudgetingServiceHandler(
	envelopeGroupRepo core.EnvelopeGroupRepository,
	envelopeRepo core.EnvelopeRepository,
	allocationRepo core.AllocationRepository,
	TransactionRepo core.TransactionRepository,
	shortcutIntentRepo core.ShortcutIntentRepository,
	logger *zap.Logger,
	db *sql.DB,
) *BudgetingServiceHandler {
	return &BudgetingServiceHandler{
		envelopeGroupRepo:  envelopeGroupRepo,
		envelopeRepo:       envelopeRepo,
		allocationRepo:     allocationRepo,
		ShortcutIntentRepo: shortcutIntentRepo,
		TransactionRepo:    TransactionRepo,
		logger:             logger,
		db:                 db,
	}
}

// Envelope Group Handlers

func getUserUUIDFromContextOrQuery(r *http.Request) (uuid.UUID, bool) {
	if val := r.Context().Value("user_uuid"); val != nil {
		if id, ok := val.(uuid.UUID); ok && id != uuid.Nil {
			return id, true
		}
		if idStr, ok := val.(string); ok && idStr != "" {
			if id, err := uuid.Parse(idStr); err == nil && id != uuid.Nil {
				return id, true
			}
		}
	}
	if queryStr := r.URL.Query().Get("user_uuid"); queryStr != "" {
		if id, err := uuid.Parse(queryStr); err == nil && id != uuid.Nil {
			return id, true
		}
	}
	return uuid.Nil, false
}

func (h *BudgetingServiceHandler) CreateEnvelopeGroup(w http.ResponseWriter, r *http.Request) {
	var group core.EnvelopeGroup
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID in context", http.StatusBadRequest)
		h.logger.Error("No user UUID found in request context")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode envelope group payload", zap.Error(err))
		return
	}

	group.UserUUID = userUUID

	if _, err := h.envelopeGroupRepo.CreateEnvelopeGroup(&group, nil); err != nil {
		http.Error(w, "Failed to create envelope group", http.StatusInternalServerError)
		h.logger.Error("Failed to create envelope group", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

func (h *BudgetingServiceHandler) GetEnvelopeGroupByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid envelope group ID", http.StatusBadRequest)
		h.logger.Error("Invalid envelope group ID", zap.String("id", idStr), zap.Error(err))
		return
	}

	group, err := h.envelopeGroupRepo.GetEnvelopeGroupByID(id)
	if err != nil {
		http.Error(w, "Envelope group not found", http.StatusNotFound)
		h.logger.Error("Envelope group not found", zap.String("id", idStr), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(group)
}

func (h *BudgetingServiceHandler) GetEnvelopeGroupsByUserUUID(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID")
		return
	}

	groups, err := h.envelopeGroupRepo.GetEnvelopeGroupsByUserUUID(userUUID)
	if err != nil {
		http.Error(w, "Failed to get envelope groups", http.StatusInternalServerError)
		h.logger.Error("Failed to get envelope groups", zap.String("user_uuid", userUUID.String()), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(groups)
}

func (h *BudgetingServiceHandler) UpdateEnvelopeGroup(w http.ResponseWriter, r *http.Request) {
	var group core.EnvelopeGroup
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID in context", http.StatusBadRequest)
		h.logger.Error("No user UUID found in request context")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode envelope group payload", zap.Error(err))
		return
	}

	group.UserUUID = userUUID

	if err := h.envelopeGroupRepo.UpdateEnvelopeGroup(&group); err != nil {
		http.Error(w, "Failed to update envelope group", http.StatusInternalServerError)
		h.logger.Error("Failed to update envelope group", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(group)
}

func (h *BudgetingServiceHandler) DeleteEnvelopeGroup(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid envelope group ID", http.StatusBadRequest)
		h.logger.Error("Invalid envelope group ID", zap.String("id", idStr), zap.Error(err))
		return
	}

	if err := h.envelopeGroupRepo.DeleteEnvelopeGroup(id); err != nil {
		http.Error(w, "Failed to delete envelope group", http.StatusInternalServerError)
		h.logger.Error("Failed to delete envelope group", zap.String("id", idStr), zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Envelope Handlers

func (h *BudgetingServiceHandler) CreateEnvelope(w http.ResponseWriter, r *http.Request) {
	var env core.Envelope
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID in context", http.StatusBadRequest)
		h.logger.Error("No user UUID found in request context")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode envelope payload", zap.Error(err))
		return
	}

	env.UserUUID = userUUID

	envID, _ := h.envelopeRepo.GetEnvelopeIdByName(env.Name, userUUID, nil)
	if envID != uuid.Nil {
		env, err := h.envelopeRepo.GetEnvelopeByID(envID)
		if err != nil {
			http.Error(w, "Failed to get envelope", http.StatusInternalServerError)
			h.logger.Error("Failed to get envelope", zap.Error(err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(env)
		return
	}

	if _, err := h.envelopeRepo.CreateEnvelope(&env, nil); err != nil {
		http.Error(w, "Failed to create envelope", http.StatusInternalServerError)
		h.logger.Error("Failed to create envelope", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(env)
}

func (h *BudgetingServiceHandler) GetEnvelopeByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid envelope ID", http.StatusBadRequest)
		h.logger.Error("Invalid envelope ID", zap.String("id", idStr), zap.Error(err))
		return
	}

	env, err := h.envelopeRepo.GetEnvelopeByID(id)
	if err != nil {
		http.Error(w, "Envelope not found", http.StatusNotFound)
		h.logger.Error("Envelope not found", zap.String("id", idStr), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(env)
}

func (h *BudgetingServiceHandler) GetEnvelopesByUserUUID(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID")
		return
	}

	envelopes, err := h.envelopeRepo.GetEnvelopesByUserUUID(userUUID)
	if err != nil {
		http.Error(w, "Failed to get envelopes", http.StatusInternalServerError)
		h.logger.Error("Failed to get envelopes", zap.String("user_uuid", userUUID.String()), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(envelopes)
}

func (h *BudgetingServiceHandler) UpdateEnvelope(w http.ResponseWriter, r *http.Request) {
	var env core.Envelope
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID in context", http.StatusBadRequest)
		h.logger.Error("No user UUID found in request context")
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode envelope payload", zap.Error(err))
		return
	}

	env.UserUUID = userUUID

	if err := h.envelopeRepo.UpdateEnvelope(&env); err != nil {
		http.Error(w, "Failed to update envelope", http.StatusInternalServerError)
		h.logger.Error("Failed to update envelope", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(env)
}

func (h *BudgetingServiceHandler) DeleteEnvelope(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid envelope ID", http.StatusBadRequest)
		h.logger.Error("Invalid envelope ID", zap.String("id", idStr), zap.Error(err))
		return
	}

	if err := h.envelopeRepo.DeleteEnvelope(id); err != nil {
		http.Error(w, "Failed to delete envelope", http.StatusInternalServerError)
		h.logger.Error("Failed to delete envelope", zap.String("id", idStr), zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Allocation Handlers

func (h *BudgetingServiceHandler) CreateAllocation(w http.ResponseWriter, r *http.Request) {
	var alloc core.Allocation
	if err := json.NewDecoder(r.Body).Decode(&alloc); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode allocation payload", zap.Error(err))
		return
	}

	if _, err := h.allocationRepo.CreateAllocation(&alloc, nil); err != nil {
		http.Error(w, "Failed to create allocation", http.StatusInternalServerError)
		h.logger.Error("Failed to create allocation", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(alloc)
}

func (h *BudgetingServiceHandler) GetAllocationByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid allocation ID", http.StatusBadRequest)
		h.logger.Error("Invalid allocation ID", zap.String("id", idStr), zap.Error(err))
		return
	}

	alloc, err := h.allocationRepo.GetAllocationByID(id)
	if err != nil {
		http.Error(w, "Allocation not found", http.StatusNotFound)
		h.logger.Error("Allocation not found", zap.String("id", idStr), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(alloc)
}

func (h *BudgetingServiceHandler) GetAllocationsByEnvelopeID(w http.ResponseWriter, r *http.Request) {
	envIDStr := r.URL.Query().Get("envelope_id")
	envID, err := uuid.Parse(envIDStr)
	if err != nil {
		http.Error(w, "Invalid envelope ID", http.StatusBadRequest)
		h.logger.Error("Invalid envelope ID", zap.String("envelope_id", envIDStr), zap.Error(err))
		return
	}

	allocations, err := h.allocationRepo.GetAllocationsByEnvelopeID(envID)
	if err != nil {
		http.Error(w, "Failed to get allocations", http.StatusInternalServerError)
		h.logger.Error("Failed to get allocations", zap.String("envelope_id", envIDStr), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(allocations)
}

func (h *BudgetingServiceHandler) UpdateAllocation(w http.ResponseWriter, r *http.Request) {
	var alloc core.Allocation
	if err := json.NewDecoder(r.Body).Decode(&alloc); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode allocation payload", zap.Error(err))
		return
	}

	if err := h.allocationRepo.UpdateAllocation(&alloc); err != nil {
		http.Error(w, "Failed to update allocation", http.StatusInternalServerError)
		h.logger.Error("Failed to update allocation", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(alloc)
}

func (h *BudgetingServiceHandler) DeleteAllocation(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid allocation ID", http.StatusBadRequest)
		h.logger.Error("Invalid allocation ID", zap.String("id", idStr), zap.Error(err))
		return
	}

	if err := h.allocationRepo.DeleteAllocation(id); err != nil {
		http.Error(w, "Failed to delete allocation", http.StatusInternalServerError)
		h.logger.Error("Failed to delete allocation", zap.String("id", idStr), zap.Error(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BudgetingServiceHandler) GetActiveAllocationsByUserUUID(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID")
		return
	}

	targetDate := time.Now()
	if dateStr := r.URL.Query().Get("date"); dateStr != "" {
		if parsedDate, err := time.Parse("2006-01-02", dateStr); err == nil {
			targetDate = parsedDate
		}
	}

	allocations, err := h.allocationRepo.GetActiveAllocationsByUserUUID(userUUID, targetDate, nil)
	if err != nil {
		http.Error(w, "Failed to get active allocations", http.StatusInternalServerError)
		h.logger.Error("Failed to get active allocations", zap.String("user_uuid", userUUID.String()), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(allocations)
}

func (h *BudgetingServiceHandler) GetActiveCategoriesByUserUUID(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		h.logger.Error("Failed to begin transaction", zap.Error(err))
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	allocations, err := h.allocationRepo.GetActiveAllocationsByUserUUID(userUUID, utils.NowUTC(), tx)
	if err != nil {
		h.logger.Error("Failed to get active allocations", zap.String("user_uuid", userUUID.String()), zap.Error(err))
		http.Error(w, "Failed to get active allocations", http.StatusInternalServerError)
		return
	}

	type budgetCategory struct {
		Name            string       `json:"name"`
		AllocatedAmount float64      `json:"allocated_amount_e5"`
		IsSystem        bool         `json:"is_system"`
		Currency        string       `json:"currency"`
		Cadence         core.Cadence `json:"cadence"`
		EnvelopeID      uuid.UUID    `json:"envelope_id"`
	}

	budgetCategories := make([]budgetCategory, 0)
	for _, allocation := range allocations {
		env, err := h.envelopeRepo.GetEnvelopeByID(allocation.EnvelopeID)
		if err != nil {
			h.logger.Error("Failed to get envelope", zap.String("envelope_id", allocation.EnvelopeID.String()), zap.Error(err))
			http.Error(w, "Failed to get envelope", http.StatusInternalServerError)
			return
		}

		budgetCategories = append(budgetCategories, budgetCategory{
			Name:            env.Name,
			AllocatedAmount: allocation.AllocatedAmountE5,
			IsSystem:        env.IsSystem,
			Currency:        env.CountryISO,
			Cadence:         env.Cadence,
			EnvelopeID:      env.ID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(budgetCategories)
}

func (h *BudgetingServiceHandler) CreateNewShortcutIntent(w http.ResponseWriter, r *http.Request) {
	type createNewShortcutIntentRequest struct {
		Name      string    `json:"name"`
		Latitude  float64   `json:"latitude"`
		Longitude float64   `json:"longitude"`
		CreatedAt time.Time `json:"created_at"`
	}

	var req createNewShortcutIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode shortcut intent payload", zap.Error(err))
		http.Error(w, "Failed to decode shortcut intent payload", http.StatusBadRequest)
		return
	}

	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID")
		return
	}

	sx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		h.logger.Error("Failed to begin transaction", zap.Error(err))
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer sx.Rollback()

	envID, err := h.envelopeRepo.GetEnvelopeIdByName(req.Name, userUUID, sx)
	if err != nil {
		h.logger.Error("Failed to get envelope ID by name", zap.String("name", req.Name), zap.Error(err))
		http.Error(w, "Failed to get envelope ID by name", http.StatusInternalServerError)
		return
	}

	createdAt := utils.NowUTC()
	if !req.CreatedAt.IsZero() {
		createdAt = req.CreatedAt.UTC()
	}

	shortcutIntent := &core.ShortcutIntent{
		UserID:     userUUID,
		EnvelopeID: &envID,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		Status:     core.StatusPending,
		CreatedAt:  createdAt,
	}

	if shortcutIntent.ID, err = h.ShortcutIntentRepo.CreateShortcutIntent(shortcutIntent, sx); err != nil {
		h.logger.Error("Failed to create shortcut intent", zap.Error(err))
		http.Error(w, "Failed to create shortcut intent", http.StatusInternalServerError)
		return
	}

	err = h.IntentProcessingWorkflow(shortcutIntent, userUUID, sx)
	if err != nil {
		if err == sql.ErrNoRows {
			h.logger.Info("No matching transaction found for intent, keeping intent pending")
			if err := sx.Commit(); err != nil {
				h.logger.Error("Failed to commit transaction", zap.Error(err))
				http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			return
		}
		h.logger.Error("Failed to process shortcut intent", zap.Error(err))
		http.Error(w, "Failed to process shortcut intent", http.StatusInternalServerError)
		return
	}

	if err := sx.Commit(); err != nil {
		h.logger.Error("Failed to commit transaction", zap.Error(err))
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shortcutIntent)
}

func (h *BudgetingServiceHandler) IntentProcessingWorkflow(shortcutIntent *core.ShortcutIntent, userUUID uuid.UUID, Tx *sql.Tx) error {
	TimeLowerbound := shortcutIntent.CreatedAt.Add(-10 * time.Minute)
	TimeUpperbound := shortcutIntent.CreatedAt.Add(3 * time.Minute)
	transaction, err := h.TransactionRepo.GetTransactionByTime(TimeLowerbound, TimeUpperbound, Tx)
	if err != nil {
		return err
	}
	transaction.EnvelopeID = shortcutIntent.EnvelopeID
	transaction.ShortcutIntentID = &shortcutIntent.ID
	if err := h.TransactionRepo.UpdateTransaction(transaction, Tx); err != nil {
		h.logger.Error("Failed to update transaction with envelope ID", zap.Error(err))
		return err
	}

	shortcutIntent.Status = core.StatusSettled
	shortcutIntent.TransactionID = &transaction.ID
	err = h.ShortcutIntentRepo.UpdateShortcutIntent(shortcutIntent, Tx)
	return err
}
