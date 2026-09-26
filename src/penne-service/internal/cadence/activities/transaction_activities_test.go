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
	getTxnByUUIDFn     func(id uuid.UUID) (*core.Transaction, error)
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

func (m *mockTxnRepo) GetTransactionByUUID(id uuid.UUID) (*core.Transaction, error) {
	if m.getTxnByUUIDFn != nil {
		return m.getTxnByUUIDFn(id)
	}
	return nil, nil
}

type mockAllocRepo struct {
	core.AllocationRepository
	updateSpentFn func(envelopeID uuid.UUID, targetDate time.Time, amountDeltaE5 int64, Tx *sql.Tx) error
}

func (m *mockAllocRepo) UpdateSpentAmount(envelopeID uuid.UUID, targetDate time.Time, amountDeltaE5 int64, Tx *sql.Tx) error {
	if m.updateSpentFn != nil {
		return m.updateSpentFn(envelopeID, targetDate, amountDeltaE5, Tx)
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

func TestTransactionActivities_GetTransactionByIDActivity(t *testing.T) {
	logger := zap.NewNop()
	txnID := uuid.New()

	// Success path
	mockTxn := &mockTxnRepo{
		getTxnByUUIDFn: func(id uuid.UUID) (*core.Transaction, error) {
			return &core.Transaction{ID: id, AmountE5: 100000}, nil
		},
	}
	acts := NewTransactionActivities(core.RepoContainer{Transaction: mockTxn}, logger)
	txn, err := acts.GetTransactionByIDActivity(context.Background(), txnID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if txn == nil || txn.ID != txnID {
		t.Fatalf("expected transaction with id %v, got %v", txnID, txn)
	}

	// Error path
	mockErr := &mockTxnRepo{
		getTxnByUUIDFn: func(id uuid.UUID) (*core.Transaction, error) {
			return nil, errors.New("not found")
		},
	}
	actsErr := NewTransactionActivities(core.RepoContainer{Transaction: mockErr}, logger)
	_, errNotFound := actsErr.GetTransactionByIDActivity(context.Background(), txnID)
	if errNotFound == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTransactionActivities_UpdateAllocationSpentActivity(t *testing.T) {
	logger := zap.NewNop()
	envID := uuid.New()
	targetDate := time.Now().UTC()

	// Success path
	called := false
	mockAlloc := &mockAllocRepo{
		updateSpentFn: func(envelopeID uuid.UUID, date time.Time, amountDeltaE5 int64, Tx *sql.Tx) error {
			called = true
			if envelopeID != envID {
				t.Errorf("expected envID %v, got %v", envID, envelopeID)
			}
			if amountDeltaE5 != 50000 {
				t.Errorf("expected delta 50000, got %d", amountDeltaE5)
			}
			return nil
		},
	}
	acts := NewTransactionActivities(core.RepoContainer{Allocation: mockAlloc}, logger)
	err := acts.UpdateAllocationSpentActivity(context.Background(), envID, targetDate, 50000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected updateSpentFn to be called")
	}

	// Error path
	mockErr := &mockAllocRepo{
		updateSpentFn: func(envelopeID uuid.UUID, date time.Time, amountDeltaE5 int64, Tx *sql.Tx) error {
			return errors.New("db error")
		},
	}
	actsErr := NewTransactionActivities(core.RepoContainer{Allocation: mockErr}, logger)
	errUpdate := actsErr.UpdateAllocationSpentActivity(context.Background(), envID, targetDate, -50000)
	if errUpdate == nil {
		t.Fatal("expected error, got nil")
	}

	// Nil allocation repo path
	actsNil := NewTransactionActivities(core.RepoContainer{Allocation: nil}, logger)
	errNil := actsNil.UpdateAllocationSpentActivity(context.Background(), envID, targetDate, 50000)
	if errNil != nil {
		t.Fatalf("expected no error for nil repo, got %v", errNil)
	}
}

type mockWishlistRepo struct {
	core.WishlistRepository
	getItemByIDFn func(id uuid.UUID) (*core.WishlistItem, error)
	updateItemFn  func(item *core.WishlistItem, Tx *sql.Tx) error
	createAllocFn func(alloc *core.WishlistAllocation, Tx *sql.Tx) (uuid.UUID, error)
}

func (m *mockWishlistRepo) GetWishlistItemByID(id uuid.UUID) (*core.WishlistItem, error) {
	if m.getItemByIDFn != nil {
		return m.getItemByIDFn(id)
	}
	return nil, nil
}

func (m *mockWishlistRepo) UpdateWishlistItem(item *core.WishlistItem, Tx *sql.Tx) error {
	if m.updateItemFn != nil {
		return m.updateItemFn(item, Tx)
	}
	return nil
}

func (m *mockWishlistRepo) CreateWishlistAllocation(alloc *core.WishlistAllocation, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createAllocFn != nil {
		return m.createAllocFn(alloc, Tx)
	}
	return uuid.Nil, nil
}

func TestTransactionActivities_FundWishlistItemActivity(t *testing.T) {
	logger := zap.NewNop()
	itemID := uuid.New()
	userUUID := uuid.New()
	txnID := uuid.New()
	now := time.Now()

	// Nil wishlist repo
	actsNil := NewTransactionActivities(core.RepoContainer{Wishlist: nil}, logger)
	if err := actsNil.FundWishlistItemActivity(context.Background(), itemID, userUUID, 500, txnID, now); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Item not found or error
	mockNotFound := &mockWishlistRepo{
		getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
			return nil, errors.New("not found")
		},
	}
	actsNotFound := NewTransactionActivities(core.RepoContainer{Wishlist: mockNotFound}, logger)
	if err := actsNotFound.FundWishlistItemActivity(context.Background(), itemID, userUUID, 500, txnID, now); err == nil {
		t.Fatal("expected error, got nil")
	}

	// UpdateWishlistItem error
	mockUpdateErr := &mockWishlistRepo{
		getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
			return &core.WishlistItem{ID: id, SavedAmountE5: 100, TargetAmountE5: 1000}, nil
		},
		updateItemFn: func(item *core.WishlistItem, Tx *sql.Tx) error {
			return errors.New("update err")
		},
	}
	actsUpdateErr := NewTransactionActivities(core.RepoContainer{Wishlist: mockUpdateErr}, logger)
	if err := actsUpdateErr.FundWishlistItemActivity(context.Background(), itemID, userUUID, 500, txnID, now); err == nil {
		t.Fatal("expected error, got nil")
	}

	// CreateWishlistAllocation error
	mockAllocErr := &mockWishlistRepo{
		getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
			return &core.WishlistItem{ID: id, SavedAmountE5: 500, TargetAmountE5: 1000}, nil
		},
		updateItemFn: func(item *core.WishlistItem, Tx *sql.Tx) error {
			return nil
		},
		createAllocFn: func(alloc *core.WishlistAllocation, Tx *sql.Tx) (uuid.UUID, error) {
			return uuid.Nil, errors.New("alloc err")
		},
	}
	actsAllocErr := NewTransactionActivities(core.RepoContainer{Wishlist: mockAllocErr}, logger)
	if err := actsAllocErr.FundWishlistItemActivity(context.Background(), itemID, userUUID, 500, txnID, now); err == nil {
		t.Fatal("expected error, got nil")
	}

	// Success with fulfillment
	var updatedItem *core.WishlistItem
	var createdAlloc *core.WishlistAllocation
	mockSuccess := &mockWishlistRepo{
		getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
			return &core.WishlistItem{ID: id, SavedAmountE5: 800, TargetAmountE5: 1000, Status: "active"}, nil
		},
		updateItemFn: func(item *core.WishlistItem, Tx *sql.Tx) error {
			updatedItem = item
			return nil
		},
		createAllocFn: func(alloc *core.WishlistAllocation, Tx *sql.Tx) (uuid.UUID, error) {
			createdAlloc = alloc
			return uuid.New(), nil
		},
	}
	actsSuccess := NewTransactionActivities(core.RepoContainer{Wishlist: mockSuccess}, logger)
	if err := actsSuccess.FundWishlistItemActivity(context.Background(), itemID, userUUID, 300, txnID, now); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if updatedItem == nil || updatedItem.SavedAmountE5 != 1100 || updatedItem.Status != "fulfilled" {
		t.Fatalf("expected updated item to have 1100 and fulfilled, got %v", updatedItem)
	}
	if createdAlloc == nil || createdAlloc.AmountE5 != 300 || *createdAlloc.TransactionID != txnID {
		t.Fatalf("expected alloc with amount 300 and txnID, got %v", createdAlloc)
	}
}

func TestTransactionActivities_UpdateWishlistSpentActivity(t *testing.T) {
	logger := zap.NewNop()
	itemID := uuid.New()
	txnID := uuid.New()

	// Nil wishlist repo
	actsNil := NewTransactionActivities(core.RepoContainer{Wishlist: nil}, logger)
	if err := actsNil.UpdateWishlistSpentActivity(context.Background(), itemID, 500, txnID); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Get error
	mockGetErr := &mockWishlistRepo{
		getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
			return nil, errors.New("item not found")
		},
	}
	actsGetErr := NewTransactionActivities(core.RepoContainer{Wishlist: mockGetErr}, logger)
	if err := actsGetErr.UpdateWishlistSpentActivity(context.Background(), itemID, 500, txnID); err == nil {
		t.Fatal("expected error, got nil")
	}

	// Clamping to 0 and reverting status to active
	var clampedItem *core.WishlistItem
	mockClamp := &mockWishlistRepo{
		getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
			return &core.WishlistItem{ID: id, SavedAmountE5: 100, TargetAmountE5: 1000, Status: "fulfilled"}, nil
		},
		updateItemFn: func(item *core.WishlistItem, Tx *sql.Tx) error {
			clampedItem = item
			return nil
		},
	}
	actsClamp := NewTransactionActivities(core.RepoContainer{Wishlist: mockClamp}, logger)
	if err := actsClamp.UpdateWishlistSpentActivity(context.Background(), itemID, -500, txnID); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if clampedItem == nil || clampedItem.SavedAmountE5 != 0 || clampedItem.Status != "active" {
		t.Fatalf("expected clamped item to be 0 and active, got %v", clampedItem)
	}

	// Reaching fulfillment
	var fulfilledItem *core.WishlistItem
	mockFulfill := &mockWishlistRepo{
		getItemByIDFn: func(id uuid.UUID) (*core.WishlistItem, error) {
			return &core.WishlistItem{ID: id, SavedAmountE5: 800, TargetAmountE5: 1000, Status: "active"}, nil
		},
		updateItemFn: func(item *core.WishlistItem, Tx *sql.Tx) error {
			fulfilledItem = item
			return nil
		},
	}
	actsFulfill := NewTransactionActivities(core.RepoContainer{Wishlist: mockFulfill}, logger)
	if err := actsFulfill.UpdateWishlistSpentActivity(context.Background(), itemID, 300, txnID); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fulfilledItem == nil || fulfilledItem.SavedAmountE5 != 1100 || fulfilledItem.Status != "fulfilled" {
		t.Fatalf("expected fulfilled item, got %v", fulfilledItem)
	}
}


