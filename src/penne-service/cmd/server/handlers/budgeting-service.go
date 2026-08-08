package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type BudgetingServiceHandler struct {
	envelopeGroupRepo core.EnvelopeGroupRepository
	envelopeRepo      core.EnvelopeRepository
	allocationRepo    core.AllocationRepository
	logger            *zap.Logger
}

func NewBudgetingServiceHandler(
	envelopeGroupRepo core.EnvelopeGroupRepository,
	envelopeRepo core.EnvelopeRepository,
	allocationRepo core.AllocationRepository,
	logger *zap.Logger,
) *BudgetingServiceHandler {
	return &BudgetingServiceHandler{
		envelopeGroupRepo: envelopeGroupRepo,
		envelopeRepo:      envelopeRepo,
		allocationRepo:    allocationRepo,
		logger:            logger,
	}
}

// Envelope Group Handlers

func (h *BudgetingServiceHandler) CreateEnvelopeGroup(w http.ResponseWriter, r *http.Request) {
	var group core.EnvelopeGroup
	userUUID, ok := r.Context().Value("user_uuid").(string)
	if !ok || userUUID == "" {
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

	if err := h.envelopeGroupRepo.CreateEnvelopeGroup(&group); err != nil {
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
	userUUID, ok := r.Context().Value("user_uuid").(string)
	if !ok || userUUID == "" {
		userUUID = r.URL.Query().Get("user_uuid")
	}

	if userUUID == "" {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID")
		return
	}

	groups, err := h.envelopeGroupRepo.GetEnvelopeGroupsByUserUUID(userUUID)
	if err != nil {
		http.Error(w, "Failed to get envelope groups", http.StatusInternalServerError)
		h.logger.Error("Failed to get envelope groups", zap.String("user_uuid", userUUID), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(groups)
}

func (h *BudgetingServiceHandler) UpdateEnvelopeGroup(w http.ResponseWriter, r *http.Request) {
	var group core.EnvelopeGroup
	userUUID, ok := r.Context().Value("user_uuid").(string)
	if !ok || userUUID == "" {
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
	userUUID, ok := r.Context().Value("user_uuid").(string)
	if !ok || userUUID == "" {
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

	if err := h.envelopeRepo.CreateEnvelope(&env); err != nil {
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
	userUUID, ok := r.Context().Value("user_uuid").(string)
	if !ok || userUUID == "" {
		userUUID = r.URL.Query().Get("user_uuid")
	}

	if userUUID == "" {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID")
		return
	}

	envelopes, err := h.envelopeRepo.GetEnvelopesByUserUUID(userUUID)
	if err != nil {
		http.Error(w, "Failed to get envelopes", http.StatusInternalServerError)
		h.logger.Error("Failed to get envelopes", zap.String("user_uuid", userUUID), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(envelopes)
}

func (h *BudgetingServiceHandler) UpdateEnvelope(w http.ResponseWriter, r *http.Request) {
	var env core.Envelope
	userUUID, ok := r.Context().Value("user_uuid").(string)
	if !ok || userUUID == "" {
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

	if err := h.allocationRepo.CreateAllocation(&alloc); err != nil {
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
	userUUID, ok := r.Context().Value("user_uuid").(string)
	if !ok || userUUID == "" {
		userUUID = r.URL.Query().Get("user_uuid")
	}

	if userUUID == "" {
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

	allocations, err := h.allocationRepo.GetActiveAllocationsByUserUUID(userUUID, targetDate)
	if err != nil {
		http.Error(w, "Failed to get active allocations", http.StatusInternalServerError)
		h.logger.Error("Failed to get active allocations", zap.String("user_uuid", userUUID), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(allocations)
}
