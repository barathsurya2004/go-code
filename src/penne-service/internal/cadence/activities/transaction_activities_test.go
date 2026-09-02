package activities

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockTxnRepo struct {
	core.TransactionRepository
	createTxnFn        func(txn *core.Transaction, Tx *sql.Tx) (uuid.UUID, error)
	getTxnByTimeFn     func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error)
	updateTxnFn        func(txn *core.Transaction, Tx *sql.Tx) error
}

func (m *mockTxnRepo) CreateTransaction(txn *core.Transaction, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createTxnFn != nil {
		return m.createTxnFn(txn, Tx)
	}
	return uuid.Nil, nil
}

func (m *mockTxnRepo) GetTransactionByTime(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
	if m.getTxnByTimeFn != nil {
		return m.getTxnByTimeFn(time_lowerbound, time_upperbound, Tx)
	}
	return nil, nil
}

func (m *mockTxnRepo) UpdateTransaction(txn *core.Transaction, Tx *sql.Tx) error {
	if m.updateTxnFn != nil {
		return m.updateTxnFn(txn, Tx)
	}
	return nil
}

type mockShortcutRepo struct {
	core.ShortcutIntentRepository
	getPendingFn func(userUUID uuid.UUID, Tx *sql.Tx, low, high time.Time) (*core.ShortcutIntent, error)
	updateFn     func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error
	createFn     func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error)
}

func (m *mockShortcutRepo) GetPendingRecentShortcutIntent(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*core.ShortcutIntent, error) {
	if m.getPendingFn != nil {
		return m.getPendingFn(userUUID, Tx, time_lowerbound, time_upperbound)
	}
	return nil, nil
}

func (m *mockShortcutRepo) UpdateShortcutIntent(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
	if m.updateFn != nil {
		return m.updateFn(shortcutIntent, Tx)
	}
	return nil
}

func (m *mockShortcutRepo) CreateShortcutIntent(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(shortcutIntent, Tx)
	}
	return uuid.Nil, nil
}

func TestTransactionActivities_CreateTransaction(t *testing.T) {
	logger := zap.NewNop()
	expectedID := uuid.New()

	mockTxn := &mockTxnRepo{
		createTxnFn: func(txn *core.Transaction, Tx *sql.Tx) (uuid.UUID, error) {
			return expectedID, nil
		},
	}
	repos := core.RepoContainer{Transaction: mockTxn}
	acts := NewTransactionActivities(repos, logger)

	res, err := acts.CreateTransaction(context.Background(), core.Transaction{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res == nil || *res != expectedID {
		t.Fatalf("expected %v, got %v", expectedID, res)
	}

	// Error path
	mockTxnErr := &mockTxnRepo{
		createTxnFn: func(txn *core.Transaction, Tx *sql.Tx) (uuid.UUID, error) {
			return uuid.Nil, errors.New("create error")
		},
	}
	actsErr := NewTransactionActivities(core.RepoContainer{Transaction: mockTxnErr}, logger)
	_, err = actsErr.CreateTransaction(context.Background(), core.Transaction{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTransactionActivities_PendingShortcutIntentActivity(t *testing.T) {
	logger := zap.NewNop()
	intentID := uuid.New()

	// Success path
	mockShortcut := &mockShortcutRepo{
		getPendingFn: func(userUUID uuid.UUID, Tx *sql.Tx, low, high time.Time) (*core.ShortcutIntent, error) {
			return &core.ShortcutIntent{ID: intentID}, nil
		},
	}
	acts := NewTransactionActivities(core.RepoContainer{ShortcutIntent: mockShortcut}, logger)
	res, err := acts.PendingShortcutIntentActivity(context.Background(), uuid.New(), time.Now(), time.Now())
	if err != nil || res == nil || res.ID != intentID {
		t.Fatalf("expected intentID %v, got res %v, err %v", intentID, res, err)
	}

	// ErrNoRows path
	mockNoRows := &mockShortcutRepo{
		getPendingFn: func(userUUID uuid.UUID, Tx *sql.Tx, low, high time.Time) (*core.ShortcutIntent, error) {
			return nil, sql.ErrNoRows
		},
	}
	actsNoRows := NewTransactionActivities(core.RepoContainer{ShortcutIntent: mockNoRows}, logger)
	resNoRows, errNoRows := actsNoRows.PendingShortcutIntentActivity(context.Background(), uuid.New(), time.Now(), time.Now())
	if errNoRows != nil || resNoRows != nil {
		t.Fatalf("expected nil result and nil error for ErrNoRows, got res %v, err %v", resNoRows, errNoRows)
	}

	// General Error path
	mockErr := &mockShortcutRepo{
		getPendingFn: func(userUUID uuid.UUID, Tx *sql.Tx, low, high time.Time) (*core.ShortcutIntent, error) {
			return nil, errors.New("db error")
		},
	}
	actsErr := NewTransactionActivities(core.RepoContainer{ShortcutIntent: mockErr}, logger)
	_, errGen := actsErr.PendingShortcutIntentActivity(context.Background(), uuid.New(), time.Now(), time.Now())
	if errGen == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTransactionActivities_UpdateShortcutIntentActivity(t *testing.T) {
	logger := zap.NewNop()
	txnID := uuid.New()

	// Success path
	mockShortcut := &mockShortcutRepo{
		updateFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
			return nil
		},
	}
	acts := NewTransactionActivities(core.RepoContainer{ShortcutIntent: mockShortcut}, logger)
	intent := &core.ShortcutIntent{ID: uuid.New(), TransactionID: &txnID}
	res, err := acts.UpdateShortcutIntentActivity(context.Background(), intent)
	if err != nil || res == nil || *res != txnID {
		t.Fatalf("expected txnID %v, got %v, err %v", txnID, res, err)
	}

	// Error path
	mockErr := &mockShortcutRepo{
		updateFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
			return errors.New("update error")
		},
	}
	actsErr := NewTransactionActivities(core.RepoContainer{ShortcutIntent: mockErr}, logger)
	_, errUpdate := actsErr.UpdateShortcutIntentActivity(context.Background(), intent)
	if errUpdate == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTransactionActivities_CreateShortcutIntent(t *testing.T) {
	logger := zap.NewNop()
	intentID := uuid.New()

	// Success path
	mockShortcut := &mockShortcutRepo{
		createFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
			return intentID, nil
		},
	}
	acts := NewTransactionActivities(core.RepoContainer{ShortcutIntent: mockShortcut}, logger)
	res, err := acts.CreateShortcutIntent(context.Background(), core.ShortcutIntent{})
	if err != nil || res == nil || *res != intentID {
		t.Fatalf("expected intentID %v, got %v, err %v", intentID, res, err)
	}

	// Error path
	mockErr := &mockShortcutRepo{
		createFn: func(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
			return uuid.Nil, errors.New("create error")
		},
	}
	actsErr := NewTransactionActivities(core.RepoContainer{ShortcutIntent: mockErr}, logger)
	_, errCreate := actsErr.CreateShortcutIntent(context.Background(), core.ShortcutIntent{})
	if errCreate == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTransactionActivities_GetTransactionByTimeActivity(t *testing.T) {
	logger := zap.NewNop()
	txnID := uuid.New()

	// Success path
	mockTxn := &mockTxnRepo{
		getTxnByTimeFn: func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
			return &core.Transaction{ID: txnID}, nil
		},
	}
	acts := NewTransactionActivities(core.RepoContainer{Transaction: mockTxn}, logger)
	res, err := acts.GetTransactionByTimeActivity(context.Background(), time.Now(), time.Now())
	if err != nil || res == nil || res.ID != txnID {
		t.Fatalf("expected txnID %v, got %v, err %v", txnID, res, err)
	}

	// ErrNoRows path
	mockNoRows := &mockTxnRepo{
		getTxnByTimeFn: func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
			return nil, sql.ErrNoRows
		},
	}
	actsNoRows := NewTransactionActivities(core.RepoContainer{Transaction: mockNoRows}, logger)
	resNoRows, errNoRows := actsNoRows.GetTransactionByTimeActivity(context.Background(), time.Now(), time.Now())
	if errNoRows != nil || resNoRows != nil {
		t.Fatalf("expected nil result and nil error for ErrNoRows, got res %v, err %v", resNoRows, errNoRows)
	}

	// General Error path
	mockErr := &mockTxnRepo{
		getTxnByTimeFn: func(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
			return nil, errors.New("db error")
		},
	}
	actsErr := NewTransactionActivities(core.RepoContainer{Transaction: mockErr}, logger)
	_, errGen := actsErr.GetTransactionByTimeActivity(context.Background(), time.Now(), time.Now())
	if errGen == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTransactionActivities_UpdateTransactionActivity(t *testing.T) {
	logger := zap.NewNop()

	// Success path
	mockTxn := &mockTxnRepo{
		updateTxnFn: func(txn *core.Transaction, Tx *sql.Tx) error {
			return nil
		},
	}
	acts := NewTransactionActivities(core.RepoContainer{Transaction: mockTxn}, logger)
	err := acts.UpdateTransactionActivity(context.Background(), core.Transaction{ID: uuid.New()})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Error path
	mockErr := &mockTxnRepo{
		updateTxnFn: func(txn *core.Transaction, Tx *sql.Tx) error {
			return errors.New("update error")
		},
	}
	actsErr := NewTransactionActivities(core.RepoContainer{Transaction: mockErr}, logger)
	errUpdate := actsErr.UpdateTransactionActivity(context.Background(), core.Transaction{ID: uuid.New()})
	if errUpdate == nil {
		t.Fatal("expected error, got nil")
	}
}

