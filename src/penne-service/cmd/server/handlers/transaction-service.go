package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TransactionServiceHandler struct {
	transactionRepo core.TransactionRepository
	logger          *zap.Logger
}

func NewTransactionServiceHandler(transactionRepo core.TransactionRepository, logger *zap.Logger) *TransactionServiceHandler {
	return &TransactionServiceHandler{
		transactionRepo: transactionRepo,
		logger:          logger,
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

	txn.UserUUID = userUUID

	if _, err := h.transactionRepo.CreateTransaction(&txn, nil); err != nil {
		http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
		h.logger.Error("Failed to create transaction", zap.Error(err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(txn)
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

	txs, err := h.transactionRepo.GetTransactionsByUserUUID(userUUID)
	if err != nil {
		http.Error(w, "Failed to retrieve transactions", http.StatusInternalServerError)
		h.logger.Error("Failed to retrieve transactions", zap.String("user_uuid", userUUID.String()), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}

func (h *TransactionServiceHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
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
	txn.UserUUID = userUUID
	if err := h.transactionRepo.UpdateTransaction(&txn); err != nil {
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
