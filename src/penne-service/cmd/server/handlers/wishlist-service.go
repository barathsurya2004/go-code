package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/barathsurya2004/go-code/penne-service/internal/wishlist"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
)

type WishlistServiceHandler struct {
	repos         core.RepoContainer
	engine        *wishlist.WishlistEngine
	logger        *zap.Logger
	db            *sql.DB
	cadenceClient client.Client
}

func NewWishlistServiceHandler(
	repos core.RepoContainer,
	logger *zap.Logger,
	db *sql.DB,
	cc client.Client,
) *WishlistServiceHandler {
	engine := wishlist.NewWishlistEngine(repos, db, logger)
	return &WishlistServiceHandler{
		repos:         repos,
		engine:        engine,
		logger:        logger,
		db:            db,
		cadenceClient: cc,
	}
}

func (h *WishlistServiceHandler) CreateWishlistItem(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID in context/query")
		return
	}

	var item core.WishlistItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode wishlist item payload", zap.Error(err))
		return
	}

	item.UserUUID = userUUID
	id, err := h.repos.Wishlist.CreateWishlistItem(&item, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Failed to create wishlist item", zap.Error(err))
		return
	}

	item.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (h *WishlistServiceHandler) GetWishlistItems(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID in context/query")
		return
	}

	user, err := h.repos.User.GetUserByUUID(userUUID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		h.logger.Error("User not found", zap.Error(err))
		return
	}

	now := utils.NowUTC()
	budget, expenses, _, _, _, err := h.engine.CalculateCycleSurplus(r.Context(), userUUID, now)
	if err != nil {
		http.Error(w, "Failed to calculate budget cycle", http.StatusInternalServerError)
		h.logger.Error("Failed to calculate cycle surplus", zap.Error(err))
		return
	}

	items, err := h.repos.Wishlist.GetWishlistItemsByUserUUID(userUUID)
	if err != nil {
		http.Error(w, "Failed to fetch wishlist items", http.StatusInternalServerError)
		h.logger.Error("Failed to fetch wishlist items", zap.Error(err))
		return
	}

	forecast := h.engine.CalculateForecasts(items, budget, expenses, now, user.SalaryDay)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(forecast)
}

func (h *WishlistServiceHandler) GetWishlistItemByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		vars := mux.Vars(r)
		idStr = vars["id"]
	}

	id, err := uuid.Parse(idStr)
	if err != nil || id == uuid.Nil {
		http.Error(w, "Invalid or missing wishlist item ID", http.StatusBadRequest)
		h.logger.Error("Invalid wishlist item ID")
		return
	}

	item, err := h.repos.Wishlist.GetWishlistItemByID(id)
	if err != nil {
		http.Error(w, "Wishlist item not found", http.StatusNotFound)
		h.logger.Error("Wishlist item not found", zap.Error(err))
		return
	}

	allocations, err := h.repos.Wishlist.GetWishlistAllocationsByItemID(id)
	if err != nil {
		allocations = []*core.WishlistAllocation{}
	}

	type response struct {
		Item        *core.WishlistItem         `json:"item"`
		Allocations []*core.WishlistAllocation `json:"allocations"`
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response{
		Item:        item,
		Allocations: allocations,
	})
}

func (h *WishlistServiceHandler) UpdateWishlistItem(w http.ResponseWriter, r *http.Request) {
	var item core.WishlistItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode update wishlist item payload", zap.Error(err))
		return
	}

	if item.ID == uuid.Nil {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			vars := mux.Vars(r)
			idStr = vars["id"]
		}
		if parsed, err := uuid.Parse(idStr); err == nil {
			item.ID = parsed
		}
	}

	if err := h.repos.Wishlist.UpdateWishlistItem(&item, nil); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Failed to update wishlist item", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "wishlist item updated successfully"})
}

func (h *WishlistServiceHandler) DeleteWishlistItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		vars := mux.Vars(r)
		idStr = vars["id"]
	}

	id, err := uuid.Parse(idStr)
	if err != nil || id == uuid.Nil {
		http.Error(w, "Invalid or missing wishlist item ID", http.StatusBadRequest)
		h.logger.Error("Invalid wishlist item ID")
		return
	}

	if err := h.repos.Wishlist.DeleteWishlistItem(id); err != nil {
		http.Error(w, "Failed to delete wishlist item", http.StatusInternalServerError)
		h.logger.Error("Failed to delete wishlist item", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "wishlist item deleted successfully"})
}

func (h *WishlistServiceHandler) GetForecast(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID in context/query")
		return
	}

	user, err := h.repos.User.GetUserByUUID(userUUID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		h.logger.Error("User not found", zap.Error(err))
		return
	}

	now := utils.NowUTC()
	budget, expenses, _, _, _, err := h.engine.CalculateCycleSurplus(r.Context(), userUUID, now)
	if err != nil {
		http.Error(w, "Failed to calculate budget surplus", http.StatusInternalServerError)
		h.logger.Error("Failed to calculate cycle surplus", zap.Error(err))
		return
	}

	items, err := h.repos.Wishlist.GetWishlistItemsByUserUUID(userUUID)
	if err != nil {
		http.Error(w, "Failed to fetch wishlist items", http.StatusInternalServerError)
		h.logger.Error("Failed to fetch wishlist items", zap.Error(err))
		return
	}

	forecast := h.engine.CalculateForecasts(items, budget, expenses, now, user.SalaryDay)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(forecast)
}

func (h *WishlistServiceHandler) DistributeSurplus(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID in context/query")
		return
	}

	var results []*core.ItemAllocationSimulation

	if h.cadenceClient != nil {
		wfOptions := client.StartWorkflowOptions{
			ID:                           "settle-wishlist-" + uuid.NewString(),
			TaskList:                     cadence.TaskListName,
			ExecutionStartToCloseTimeout: 5 * time.Minute,
		}

		workflowRun, err := h.cadenceClient.ExecuteWorkflow(
			r.Context(),
			wfOptions,
			"SettleWishlistWorkflow",
			userUUID,
		)
		if err != nil {
			h.logger.Error("Failed to start Cadence settle wishlist workflow", zap.Error(err))
			http.Error(w, "Failed to execute surplus distribution", http.StatusInternalServerError)
			return
		}

		if err := workflowRun.Get(r.Context(), &results); err != nil {
			h.logger.Error("Cadence settle wishlist workflow failed", zap.Error(err))
			http.Error(w, "Failed to execute surplus distribution", http.StatusInternalServerError)
			return
		}
	} else {
		now := utils.NowUTC()
		var err error
		results, err = h.engine.ApplySurplusDistribution(r.Context(), userUUID, now)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			h.logger.Error("Failed to apply surplus distribution", zap.Error(err))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Surplus distributed successfully",
		"allocations": results,
	})
}

func (h *WishlistServiceHandler) UpdateBudgetSettings(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID in context/query")
		return
	}

	type request struct {
		MonthlyBudgetE5 int64 `json:"monthly_budget_e5"`
		SalaryDay       int   `json:"salary_day"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode budget settings payload", zap.Error(err))
		return
	}

	if req.SalaryDay <= 0 {
		req.SalaryDay = 1
	}

	if err := h.repos.User.UpdateBudgetSettings(userUUID, req.MonthlyBudgetE5, req.SalaryDay, nil); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Failed to update budget settings", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "budget settings updated successfully"})
}
