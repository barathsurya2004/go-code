package activities

import (
	"context"
	"database/sql"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TransactionActivities struct {
	Repos  core.RepoContainer
	logger *zap.Logger
}

func NewTransactionActivities(repos core.RepoContainer, logger *zap.Logger) *TransactionActivities {
	return &TransactionActivities{
		Repos:  repos,
		logger: logger,
	}
}

func (a *TransactionActivities) CreateTransaction(ctx context.Context, txn core.Transaction) (*uuid.UUID, error) {
	txnID, err := a.Repos.Transaction.CreateTransaction(&txn, nil)
	if err != nil {
		a.logger.Error("Failed to create transaction and workflow", zap.Error(err))
		return nil, err
	}
	a.logger.Info("Waiting for the shortcut intent to trigger the attribution")
	return &txnID, nil
}

func (a *TransactionActivities) PendingShortcutIntentActivity(ctx context.Context, userUUID uuid.UUID, TimeLowerbound, TimeUpperbound time.Time) (*core.ShortcutIntent, error) {
	pendingShortcuts, err := a.Repos.ShortcutIntent.GetPendingRecentShortcutIntent(userUUID, nil, TimeLowerbound, TimeUpperbound)
	if err != nil {
		if err == sql.ErrNoRows {
			a.logger.Warn("No pending shortcut intent found", zap.String("user_uuid", userUUID.String()))
			return nil, nil
		}
		a.logger.Error("Failed to get pending shortcut intent", zap.String("user_uuid", userUUID.String()), zap.Error(err))
		return nil, err
	}
	return pendingShortcuts, nil
}

func (a *TransactionActivities) UpdateShortcutIntentActivity(ctx context.Context, shortcutIntent *core.ShortcutIntent) (*uuid.UUID, error) {
	if err := a.Repos.ShortcutIntent.UpdateShortcutIntent(shortcutIntent, nil); err != nil {
		a.logger.Error("Failed to update shortcut intent", zap.Error(err))
		return nil, err
	}
	return shortcutIntent.TransactionID, nil
}

func (a *TransactionActivities) CreateShortcutIntent(ctx context.Context, shortcutIntent core.ShortcutIntent) (*uuid.UUID, error) {
	intentID, err := a.Repos.ShortcutIntent.CreateShortcutIntent(&shortcutIntent, nil)
	if err != nil {
		a.logger.Error("Failed to create shortcut intent", zap.Error(err))
		return nil, err
	}
	return &intentID, nil
}

func (a *TransactionActivities) GetTransactionByTimeActivity(ctx context.Context, TimeLowerbound, TimeUpperbound time.Time) (*core.Transaction, error) {
	txn, err := a.Repos.Transaction.GetTransactionByTime(TimeLowerbound, TimeUpperbound, nil)
	if err != nil {
		if err == sql.ErrNoRows {
			a.logger.Warn("No matching transaction found for intent")
			return nil, nil
		}
		a.logger.Error("Failed to get transaction by time", zap.Error(err))
		return nil, err
	}
	return txn, nil
}

func (a *TransactionActivities) UpdateTransactionActivity(ctx context.Context, txn core.Transaction) error {
	if err := a.Repos.Transaction.UpdateTransaction(&txn, nil); err != nil {
		a.logger.Error("Failed to update transaction", zap.Error(err))
		return err
	}
	return nil
}
