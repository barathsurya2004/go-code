package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence"
	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/workflows"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/emailparser"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
)

type TransactionServiceHandler struct {
	transactionRepo    core.TransactionRepository
	shortcutIntentRepo core.ShortcutIntentRepository
	repos              core.RepoContainer
	logger             *zap.Logger
	db                 *sql.DB
	cadenceClient      client.Client
}

func NewTransactionServiceHandler(transactionRepo core.TransactionRepository, shortcutIntentRepo core.ShortcutIntentRepository, logger *zap.Logger, db *sql.DB, cc client.Client, repos core.RepoContainer) *TransactionServiceHandler {
	return &TransactionServiceHandler{
		transactionRepo:    transactionRepo,
		shortcutIntentRepo: shortcutIntentRepo,
		logger:             logger,
		db:                 db,
		cadenceClient:      cc,
		repos:              repos,
	}
}

func (h *TransactionServiceHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var txn core.Transaction
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID in context", http.StatusBadRequest)
		h.logger.Error("No user UUID found in request context")
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&txn); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode transaction payload", zap.Error(err))
		return
	}

	txn.UserID = userUUID

	// tx, err := h.db.BeginTx(r.Context(), nil)
	// if err != nil {
	// 	http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
	// 	h.logger.Error("Failed to begin transaction", zap.Error(err))
	// 	return
	// }
	// defer tx.Rollback()

	var resultUUID *uuid.UUID
	if h.cadenceClient != nil {
		wfOptions := client.StartWorkflowOptions{
			ID:                           uuid.NewString(),
			TaskList:                     cadence.TaskListName,
			ExecutionStartToCloseTimeout: 5 * time.Minute,
		}
		workflowRun, err := h.cadenceClient.ExecuteWorkflow(
			r.Context(),
			wfOptions,
			"CreateTransactionWorkflow",
			txn,
		)

		if err != nil {
			h.logger.Error("Failed to start cadence workflow", zap.Error(err))
			// return workflow failed status
			http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
			return
		} else if workflowRun != nil {
			if err := workflowRun.Get(r.Context(), &resultUUID); err != nil {
				h.logger.Error("workflow Excecution failed", zap.Error(err))
				// return workflow failed status
				http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
				return
			}

			if resultUUID != nil {
				fmt.Println(*resultUUID)
			} else {
				fmt.Println("workflow completed with nor result")
			}
		}
	}

	// txnID, err := h.CreateTransactionWorkflow(&txn, userUUID, tx)
	// if err != nil {
	// 	http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
	// 	h.logger.Error("Failed to create transaction", zap.Error(err))
	// 	return
	// }

	// if err := tx.Commit(); err != nil {
	// 	http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
	// 	h.logger.Error("Failed to commit transaction", zap.Error(err))
	// 	return
	// }

	var respUUID uuid.UUID
	if resultUUID != nil {
		respUUID = *resultUUID
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		TxnUUID uuid.UUID `json:"txn_uuid"`
	}{
		TxnUUID: respUUID,
	})
}

func (h *TransactionServiceHandler) GetTransactionByUUID(w http.ResponseWriter, r *http.Request) {
	txnUUIDStr := r.URL.Query().Get("txn_uuid")
	txnUUID, err := uuid.Parse(txnUUIDStr)
	if err != nil || txnUUID == uuid.Nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Invalid request payload")
		return
	}

	txn, err := h.transactionRepo.GetTransactionByUUID(txnUUID)
	if err != nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		h.logger.Error("Transaction not found", zap.String("uuid", txnUUIDStr), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
}

func (h *TransactionServiceHandler) GetTransactionsByUserUUID(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		userUUIDStr := r.URL.Query().Get("user_uuid")
		var err error
		userUUID, err = uuid.Parse(userUUIDStr)
		if err != nil || userUUID == uuid.Nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			h.logger.Error("Invalid request payload")
			return
		}
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var lastTransactionCreatedAt time.Time
	lastTransactionCreatedAtStr := r.URL.Query().Get("lastTransactionCreatedAt")
	if lastTransactionCreatedAtStr != "" {
		var err error
		lastTransactionCreatedAt, err = time.Parse(time.RFC3339, lastTransactionCreatedAtStr)
		if err != nil {
			lastTransactionCreatedAt, _ = time.Parse("2006-01-02T15:04:05Z07:00", lastTransactionCreatedAtStr)
		}
	}

	var lastTransactionID uuid.UUID
	lastTransactionIDStr := r.URL.Query().Get("lastTransactionID")
	if lastTransactionIDStr != "" {
		lastTransactionID, _ = uuid.Parse(lastTransactionIDStr)
	}

	txs, err := h.transactionRepo.GetTransactionByUserUUIDPaginated(userUUID, lastTransactionCreatedAt, lastTransactionID, limit)
	if err != nil {
		http.Error(w, "Failed to retrieve transactions", http.StatusInternalServerError)
		h.logger.Error("Failed to retrieve transactions", zap.String("user_uuid", userUUID.String()), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(txs)
}

func (h *TransactionServiceHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	var txn core.Transaction
	if err := json.NewDecoder(r.Body).Decode(&txn); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode transaction payload", zap.Error(err))
		return
	}
	//get the existing transaction
	txnToUpdate, err := h.transactionRepo.GetTransactionByUUID(txn.ID)
	if err != nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		h.logger.Error("Transaction not found", zap.String("uuid", txn.ID.String()), zap.Error(err))
		return
	}

	newTxn := txnToUpdate
	if txn.AmountE5 != 0 {
		newTxn.AmountE5 = txn.AmountE5
	}
	if txn.Type != "" {
		newTxn.Type = txn.Type
	}
	if txn.EnvelopeID != nil {
		newTxn.EnvelopeID = txn.EnvelopeID
	}

	if err := h.transactionRepo.UpdateTransaction(newTxn, nil); err != nil {
		http.Error(w, "Failed to update transaction", http.StatusInternalServerError)
		h.logger.Error("Failed to update transaction", zap.Error(err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TransactionServiceHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	txnUUIDStr := r.URL.Query().Get("uuid")
	if txnUUIDStr == "" {
		txnUUIDStr = r.URL.Query().Get("txn_uuid")
	}
	txnUUID, err := uuid.Parse(txnUUIDStr)
	if err != nil || txnUUID == uuid.Nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Invalid request payload")
		return
	}

	if err := h.transactionRepo.DeleteTransaction(txnUUID); err != nil {
		http.Error(w, "Failed to delete transaction", http.StatusInternalServerError)
		h.logger.Error("Failed to delete transaction", zap.String("uuid", txnUUIDStr), zap.Error(err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TransactionServiceHandler) CreateTransactionWorkflow(txn *core.Transaction, userUUID uuid.UUID, Tx *sql.Tx) (*uuid.UUID, error) {
	if txn.CreatedAt.IsZero() {
		txn.CreatedAt = utils.NowUTC()
	} else {
		txn.CreatedAt = txn.CreatedAt.UTC()
	}
	TimeUpperbound := txn.CreatedAt.Add(3 * time.Minute)
	TimeLowerbound := txn.CreatedAt.Add(-10 * time.Minute)
	pendingShortcutIntent, err := h.shortcutIntentRepo.GetPendingRecentShortcutIntent(userUUID, Tx, TimeLowerbound, TimeUpperbound)
	if err != nil {
		if err != sql.ErrNoRows {
			h.logger.Error("error in fetching pending shortcuts for transaction", zap.Error(err))
			return nil, err
		}
		h.logger.Info("no pending shortcuts found for transaction")
	}
	if pendingShortcutIntent != nil {
		txn.ShortcutIntentID = &pendingShortcutIntent.ID
		txnID, err := h.transactionRepo.CreateTransaction(txn, Tx)
		if err != nil {
			h.logger.Error("Failed to create transaction workflow", zap.Error(err))
			return nil, err
		}
		pendingShortcutIntent.TransactionID = &txnID
		pendingShortcutIntent.Status = core.StatusSettled
		if err := h.shortcutIntentRepo.UpdateShortcutIntent(pendingShortcutIntent, Tx); err != nil {
			h.logger.Error("Failed to create transaction workflow", zap.Error(err))
			return nil, err
		}
		return &txnID, nil
	} else {
		txnID, err := h.transactionRepo.CreateTransaction(txn, Tx)
		if err != nil {
			h.logger.Error("Failed to create transaction and workflow", zap.Error(err))
			return nil, err
		}
		h.logger.Info("Waiting for the shortcut intent to trigger the attribution")
		return &txnID, nil
	}

}

func (h *TransactionServiceHandler) DashboardSummaryHandler(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		userUUIDStr := r.URL.Query().Get("user_uuid")
		var err error
		userUUID, err = uuid.Parse(userUUIDStr)
		if err != nil || userUUID == uuid.Nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			h.logger.Error("Invalid request payload")
			return
		}
	}
	DashboardSummary, err := h.transactionRepo.GetDashboardSummary(userUUID)
	if err != nil {
		http.Error(w, "Failed to fetch dashboard summary", http.StatusInternalServerError)
		h.logger.Error("Failed to fetch dashboard summary", zap.String("user_uuid", userUUID.String()), zap.Error(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(DashboardSummary)

}

type changeTransactionToTransferRequest struct {
	AmountE5  int64     `json:"amount_e5"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *TransactionServiceHandler) ChangeTransactionToTransfer(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := getUserUUIDFromContextOrQuery(r)
	if !ok {
		http.Error(w, "Missing user UUID in context", http.StatusBadRequest)
		h.logger.Error("No user UUID found in request context")
		return
	}

	var req changeTransactionToTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode payload", zap.Error(err))
		return
	}

	if req.AmountE5 <= 0 {
		http.Error(w, "Amount must be greater than zero", http.StatusBadRequest)
		h.logger.Error("Invalid amount_e5 provided")
		return
	}

	targetTime := utils.NowUTC()
	if !req.CreatedAt.IsZero() {
		targetTime = req.CreatedAt.UTC()
	}

	timeLowerbound := targetTime.Add(-5 * time.Minute)
	timeUpperbound := targetTime.Add(5 * time.Minute)

	txn, err := h.transactionRepo.GetTransactionByAmountAndTime(userUUID, req.AmountE5, timeLowerbound, timeUpperbound, nil)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Transaction not found", http.StatusNotFound)
			h.logger.Warn("Transaction not found for transfer conversion", zap.String("user_uuid", userUUID.String()), zap.Int64("amount_e5", req.AmountE5))
			return
		}
		http.Error(w, "Failed to find transaction", http.StatusInternalServerError)
		h.logger.Error("Failed to find transaction for transfer conversion", zap.Error(err))
		return
	}

	txn.Type = core.TxnTypeTransfer
	if err := h.transactionRepo.UpdateTransaction(txn, nil); err != nil {
		http.Error(w, "Failed to update transaction", http.StatusInternalServerError)
		h.logger.Error("Failed to update transaction type to transfer", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(txn)
}

type ProcessEmailRequest struct {
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Subject   string     `json:"subject"`
	Body      string     `json:"body"`
	EmailDate *time.Time `json:"email_date,omitempty"`
}

func (h *TransactionServiceHandler) ProcessEmailTransaction(w http.ResponseWriter, r *http.Request) {
	var req ProcessEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Failed to decode email transaction payload", zap.Error(err))
		return
	}

	var userUUID uuid.UUID
	if req.UserID != nil && *req.UserID != uuid.Nil {
		userUUID = *req.UserID
	} else {
		var ok bool
		userUUID, ok = getUserUUIDFromContextOrQuery(r)
		if !ok {
			http.Error(w, "Missing user UUID in request or authentication context", http.StatusBadRequest)
			h.logger.Error("No user UUID found for email transaction")
			return
		}
	}

	emailDate := utils.NowUTC()
	if req.EmailDate != nil && !req.EmailDate.IsZero() {
		emailDate = req.EmailDate.UTC()
	}

	if h.cadenceClient != nil {
		wfOptions := client.StartWorkflowOptions{
			ID:                           uuid.NewString(),
			TaskList:                     cadence.TaskListName,
			ExecutionStartToCloseTimeout: 5 * time.Minute,
		}
		workflowRun, err := h.cadenceClient.ExecuteWorkflow(
			r.Context(),
			wfOptions,
			"ProcessEmailWorkflow",
			workflows.EmailTransactionInput{
				UserID:    userUUID,
				Subject:   req.Subject,
				Body:      req.Body,
				EmailDate: emailDate,
			},
		)

		if err != nil {
			h.logger.Error("Failed to start ProcessEmailWorkflow", zap.Error(err))
			http.Error(w, "Failed to process email transaction", http.StatusInternalServerError)
			return
		}

		var result workflows.ProcessEmailWorkflowResult
		if err := workflowRun.Get(r.Context(), &result); err != nil {
			h.logger.Error("ProcessEmailWorkflow execution failed", zap.Error(err))
			http.Error(w, fmt.Sprintf("Failed to process email transaction: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(result)
		return
	}

	// Fallback when running without Cadence client
	parsed, err := emailparser.ParseIDFCEmail(req.Subject, req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	txn := core.Transaction{
		UserID:        userUUID,
		AmountE5:      parsed.AmountE5,
		Type:          parsed.Type,
		PaymentMethod: parsed.PaymentMethod,
		CountryISO:    "IN",
		CreatedAt:     emailDate,
	}

	txnID, err := h.CreateTransactionWorkflow(&txn, userUUID, nil)
	if err != nil {
		http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
		return
	}

	res := workflows.ProcessEmailWorkflowResult{
		TransactionID: *txnID,
		ParsedDetails: *parsed,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

