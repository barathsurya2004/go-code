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
	userUUID, ok := r.Context().Value("user_uuid").(string)
	if !ok || userUUID == "" {
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

	if err := h.transactionRepo.CreateTransaction(&txn); err != nil {
		http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
		h.logger.Error("Failed to create transaction", zap.Error(err))
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(txn)
}

func (h *TransactionServiceHandler) GetTransactionByUUID(w http.ResponseWriter, r *http.Request) {
	txnUUID := r.URL.Query().Get("txn_uuid")
	if txnUUID == "" || uuid.Validate(txnUUID) != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Invalid request payload")
		return
	}

	txn, err := h.transactionRepo.GetTransactionByUUID(txnUUID)
	if err != nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		h.logger.Error("Transaction not found", zap.String("uuid", txnUUID), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txn)
}

func (h *TransactionServiceHandler) GetTransactionsByUserUUID(w http.ResponseWriter, r *http.Request) {
	userUUID := r.URL.Query().Get("user_uuid")
	if userUUID == "" || uuid.Validate(userUUID) != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Invalid request payload")
		return
	}

	txs, err := h.transactionRepo.GetTransactionsByUserUUID(userUUID)
	if err != nil {
		http.Error(w, "Failed to retrieve transactions", http.StatusInternalServerError)
		h.logger.Error("Failed to retrieve transactions", zap.String("user_uuid", userUUID), zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txs)
}

func (h *TransactionServiceHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	var txn core.Transaction
	userUUID, ok := r.Context().Value("user_uuid").(string)
	if !ok || userUUID == "" {
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
	userUUID := r.Context().Value("user_uuid").(string)
	if userUUID == "" || uuid.Validate(userUUID) != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		h.logger.Error("Invalid request payload")
		return
	}

	if err := h.transactionRepo.DeleteTransaction(userUUID); err != nil {
		http.Error(w, "Failed to delete transaction", http.StatusInternalServerError)
		h.logger.Error("Failed to delete transaction", zap.String("uuid", userUUID), zap.Error(err))
		return
	}
	w.WriteHeader(http.StatusOK)
}
