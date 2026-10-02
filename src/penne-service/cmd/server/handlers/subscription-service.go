package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence"
	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/activities"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
)

type SubscriptionServiceHandler struct {
	repos                  core.RepoContainer
	logger                 *zap.Logger
	db                     *sql.DB
	cadenceClient          client.Client
	subscriptionActivities *activities.SubscriptionActivities
}

func NewSubscriptionServiceHandler(
	repos core.RepoContainer,
	logger *zap.Logger,
	db *sql.DB,
	cc client.Client,
) *SubscriptionServiceHandler {
	subActs := activities.NewSubscriptionActivities(repos, logger)
	return &SubscriptionServiceHandler{
		repos:                  repos,
		logger:                 logger,
		db:                     db,
		cadenceClient:          cc,
		subscriptionActivities: subActs,
	}
}

func (h *SubscriptionServiceHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID in context/query")
		return
	}

	var sub core.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode subscription payload", zap.Error(err))
		return
	}

	sub.UserUUID = userUUID

	var id uuid.UUID
	if h.cadenceClient != nil {
		wfOptions := client.StartWorkflowOptions{
			ID:                           "create-subscription-" + uuid.NewString(),
			TaskList:                     cadence.TaskListName,
			ExecutionStartToCloseTimeout: 5 * time.Minute,
		}

		workflowRun, err := h.cadenceClient.ExecuteWorkflow(
			r.Context(),
			wfOptions,
			"CreateSubscriptionWorkflow",
			&sub,
		)
		if err != nil {
			h.logger.Error("Failed to start Cadence create subscription workflow", zap.Error(err))
			http.Error(w, "Failed to create subscription via workflow", http.StatusInternalServerError)
			return
		}

		if err := workflowRun.Get(r.Context(), &id); err != nil {
			h.logger.Error("Cadence create subscription workflow failed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		var err error
		id, err = h.repos.Subscription.CreateSubscription(&sub, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			h.logger.Error("Failed to create subscription in repo", zap.Error(err))
			return
		}
	}

	sub.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sub)
}

func (h *SubscriptionServiceHandler) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID", http.StatusBadRequest)
		h.logger.Error("Missing user UUID in context/query")
		return
	}

	subs, err := h.repos.Subscription.GetSubscriptionsByUserUUID(userUUID)
	if err != nil {
		http.Error(w, "Failed to fetch subscriptions", http.StatusInternalServerError)
		h.logger.Error("Failed to fetch subscriptions", zap.Error(err))
		return
	}

	var totalMonthlyCommitmentE5 int64
	var activeCount int
	var pausedCount int
	var nextUpcoming *core.Subscription

	now := utils.NowUTC()
	for _, sub := range subs {
		if sub.Status == core.SubscriptionStatusActive {
			activeCount++
			totalMonthlyCommitmentE5 += core.CalculateMonthlyEquivalentE5(sub.AmountE5, sub.BillingCycle)

			if nextUpcoming == nil {
				nextUpcoming = sub
			} else if sub.NextBillingDate.Before(nextUpcoming.NextBillingDate) && (sub.NextBillingDate.After(now) || sub.NextBillingDate.Equal(now)) {
				nextUpcoming = sub
			}
		} else if sub.Status == core.SubscriptionStatusPaused {
			pausedCount++
		}
	}

	if subs == nil {
		subs = []*core.Subscription{}
	}

	summary := &core.SubscriptionSummary{
		TotalMonthlyCommitmentE5: totalMonthlyCommitmentE5,
		ActiveCount:              activeCount,
		PausedCount:              pausedCount,
		NextUpcoming:             nextUpcoming,
		Subscriptions:            subs,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(summary)
}

func (h *SubscriptionServiceHandler) GetSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing subscription ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid subscription ID", http.StatusBadRequest)
		return
	}

	sub, err := h.repos.Subscription.GetSubscriptionByID(id)
	if err != nil {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		h.logger.Error("Subscription not found", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sub)
}

func (h *SubscriptionServiceHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	var sub core.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode subscription update payload", zap.Error(err))
		return
	}

	if sub.ID == uuid.Nil {
		idStr := r.URL.Query().Get("id")
		if parsed, err := uuid.Parse(idStr); err == nil {
			sub.ID = parsed
		}
	}

	existing, err := h.repos.Subscription.GetSubscriptionByID(sub.ID)
	if err != nil {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		h.logger.Error("Subscription not found for update", zap.Error(err))
		return
	}

	if existing != nil {
		if sub.UserUUID == uuid.Nil {
			sub.UserUUID = existing.UserUUID
		}
		if sub.OccurrenceCount == 0 && existing.OccurrenceCount > 0 {
			sub.OccurrenceCount = existing.OccurrenceCount
		}
		if sub.ChargeWindowHours == 0 && existing.ChargeWindowHours > 0 {
			sub.ChargeWindowHours = existing.ChargeWindowHours
		}
		if sub.LastChargedAt == nil && existing.LastChargedAt != nil {
			sub.LastChargedAt = existing.LastChargedAt
		}
		if sub.LastTransactionID == nil && existing.LastTransactionID != nil {
			sub.LastTransactionID = existing.LastTransactionID
		}
		if sub.MerchantPattern == "" && existing.MerchantPattern != "" {
			sub.MerchantPattern = existing.MerchantPattern
		}
	}

	if err := h.repos.Subscription.UpdateSubscription(&sub, nil); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Failed to update subscription", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sub)
}

func (h *SubscriptionServiceHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "Missing subscription ID", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid subscription ID", http.StatusBadRequest)
		return
	}

	if err := h.repos.Subscription.DeleteSubscription(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error("Failed to delete subscription", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "subscription deleted successfully"})
}

func (h *SubscriptionServiceHandler) RenewSubscription(w http.ResponseWriter, r *http.Request) {
	type request struct {
		ID uuid.UUID `json:"id"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		idStr := r.URL.Query().Get("id")
		if parsed, err := uuid.Parse(idStr); err == nil {
			req.ID = parsed
		} else {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			h.logger.Error("Failed to decode renewal payload", zap.Error(err))
			return
		}
	}

	if req.ID == uuid.Nil {
		idStr := r.URL.Query().Get("id")
		if parsed, err := uuid.Parse(idStr); err == nil {
			req.ID = parsed
		}
	}

	if req.ID == uuid.Nil {
		http.Error(w, "Missing subscription ID for renewal", http.StatusBadRequest)
		return
	}

	var txn *core.Transaction
	if h.cadenceClient != nil {
		wfOptions := client.StartWorkflowOptions{
			ID:                           "renew-subscription-" + uuid.NewString(),
			TaskList:                     cadence.TaskListName,
			ExecutionStartToCloseTimeout: 5 * time.Minute,
		}

		workflowRun, err := h.cadenceClient.ExecuteWorkflow(
			r.Context(),
			wfOptions,
			"RenewSubscriptionWorkflow",
			req.ID,
		)
		if err != nil {
			h.logger.Error("Failed to start Cadence renew subscription workflow", zap.Error(err))
			http.Error(w, "Failed to renew subscription via workflow", http.StatusInternalServerError)
			return
		}

		if err := workflowRun.Get(r.Context(), &txn); err != nil {
			h.logger.Error("Cadence renew subscription workflow failed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		var err error
		txn, err = h.subscriptionActivities.RenewSubscriptionActivity(r.Context(), req.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			h.logger.Error("Failed to renew subscription via activity", zap.Error(err))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Subscription renewed successfully",
		"transaction": txn,
	})
}

func (h *SubscriptionServiceHandler) ScanDueSubscriptions(w http.ResponseWriter, r *http.Request) {
	asOf := utils.NowUTC()
	if asOfStr := r.URL.Query().Get("as_of"); asOfStr != "" {
		if parsed, err := time.Parse(time.RFC3339, asOfStr); err == nil {
			asOf = parsed
		}
	}

	var renewedIDs []uuid.UUID
	if h.cadenceClient != nil {
		wfOptions := client.StartWorkflowOptions{
			ID:                           "scan-due-subscriptions-" + uuid.NewString(),
			TaskList:                     cadence.TaskListName,
			ExecutionStartToCloseTimeout: 10 * time.Minute,
		}

		workflowRun, err := h.cadenceClient.ExecuteWorkflow(
			r.Context(),
			wfOptions,
			"ScanAndRenewDueSubscriptionsWorkflow",
			asOf,
		)
		if err != nil {
			h.logger.Error("Failed to start Cadence scan due subscriptions workflow", zap.Error(err))
			http.Error(w, "Failed to scan due subscriptions via workflow", http.StatusInternalServerError)
			return
		}

		if err := workflowRun.Get(r.Context(), &renewedIDs); err != nil {
			h.logger.Error("Cadence scan due subscriptions workflow failed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		dueSubs, err := h.repos.Subscription.GetDueSubscriptions(asOf, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			h.logger.Error("Failed to fetch due subscriptions", zap.Error(err))
			return
		}
		for _, sub := range dueSubs {
			_, err := h.subscriptionActivities.RenewSubscriptionActivity(r.Context(), sub.ID)
			if err == nil {
				renewedIDs = append(renewedIDs, sub.ID)
			}
		}
	}

	if renewedIDs == nil {
		renewedIDs = []uuid.UUID{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Due subscriptions processed",
		"renewed_ids": renewedIDs,
	})
}
