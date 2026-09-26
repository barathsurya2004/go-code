package wishlist

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockUserRepo struct {
	core.UserRepository
	getUserByUUIDFn func(uuid.UUID) (*core.User, error)
}

func (m *mockUserRepo) GetUserByUUID(id uuid.UUID) (*core.User, error) {
	if m.getUserByUUIDFn != nil {
		return m.getUserByUUIDFn(id)
	}
	return &core.User{UUID: id, MonthlyBudgetE5: 10000000, SalaryDay: 25}, nil
}

type mockWishlistRepo struct {
	core.WishlistRepository
	createItemFn        func(*core.WishlistItem, *sql.Tx) (uuid.UUID, error)
	getItemFn           func(uuid.UUID) (*core.WishlistItem, error)
	getActiveItemsFn    func(uuid.UUID) ([]*core.WishlistItem, error)
	updateItemFn        func(*core.WishlistItem, *sql.Tx) error
	createAllocationFn  func(*core.WishlistAllocation, *sql.Tx) (uuid.UUID, error)
}

func (m *mockWishlistRepo) CreateWishlistItem(item *core.WishlistItem, tx *sql.Tx) (uuid.UUID, error) {
	if m.createItemFn != nil {
		return m.createItemFn(item, tx)
	}
	return uuid.New(), nil
}

func (m *mockWishlistRepo) GetWishlistItemByID(id uuid.UUID) (*core.WishlistItem, error) {
	if m.getItemFn != nil {
		return m.getItemFn(id)
	}
	return &core.WishlistItem{ID: id, Title: "Item", TargetAmountE5: 1000, SavedAmountE5: 0, Priority: 3, Urgency: 3, Status: "active"}, nil
}

func (m *mockWishlistRepo) GetActiveWishlistItemsByUserUUID(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
	if m.getActiveItemsFn != nil {
		return m.getActiveItemsFn(userUUID)
	}
	return nil, nil
}

func (m *mockWishlistRepo) UpdateWishlistItem(item *core.WishlistItem, tx *sql.Tx) error {
	if m.updateItemFn != nil {
		return m.updateItemFn(item, tx)
	}
	return nil
}

func (m *mockWishlistRepo) CreateWishlistAllocation(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
	if m.createAllocationFn != nil {
		return m.createAllocationFn(alloc, tx)
	}
	return uuid.New(), nil
}

func TestWishlistEngine_GetCycleDateRange(t *testing.T) {
	engine := NewWishlistEngine(core.RepoContainer{}, nil, zap.NewNop())
	loc := time.UTC

	t.Run("Salary Day 1 - Standard Month", func(t *testing.T) {
		target := time.Date(2026, time.March, 15, 12, 0, 0, 0, loc)
		start, end := engine.GetCycleDateRange(1, target)

		expectedStart := time.Date(2026, time.March, 1, 0, 0, 0, 0, loc)
		expectedEnd := time.Date(2026, time.April, 1, 0, 0, 0, 0, loc).Add(-time.Nanosecond)

		if !start.Equal(expectedStart) {
			t.Errorf("expected start %v, got %v", expectedStart, start)
		}
		if !end.Equal(expectedEnd) {
			t.Errorf("expected end %v, got %v", expectedEnd, end)
		}
	})

	t.Run("Salary Day 25 - After Salary Date", func(t *testing.T) {
		target := time.Date(2026, time.March, 28, 12, 0, 0, 0, loc)
		start, end := engine.GetCycleDateRange(25, target)

		expectedStart := time.Date(2026, time.March, 25, 0, 0, 0, 0, loc)
		expectedEnd := time.Date(2026, time.April, 25, 0, 0, 0, 0, loc).Add(-time.Nanosecond)

		if !start.Equal(expectedStart) {
			t.Errorf("expected start %v, got %v", expectedStart, start)
		}
		if !end.Equal(expectedEnd) {
			t.Errorf("expected end %v, got %v", expectedEnd, end)
		}
	})

	t.Run("Salary Day 25 - Before Salary Date", func(t *testing.T) {
		target := time.Date(2026, time.March, 10, 12, 0, 0, 0, loc)
		start, end := engine.GetCycleDateRange(25, target)

		expectedStart := time.Date(2026, time.February, 25, 0, 0, 0, 0, loc)
		expectedEnd := time.Date(2026, time.March, 25, 0, 0, 0, 0, loc).Add(-time.Nanosecond)

		if !start.Equal(expectedStart) {
			t.Errorf("expected start %v, got %v", expectedStart, start)
		}
		if !end.Equal(expectedEnd) {
			t.Errorf("expected end %v, got %v", expectedEnd, end)
		}
	})
}

func TestWishlistEngine_CalculateCycleSurplus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	userUUID := uuid.New()
	userRepo := &mockUserRepo{
		getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
			return &core.User{
				UUID:            userUUID,
				MonthlyBudgetE5: 100000,
				SalaryDay:       1,
			}, nil
		},
	}

	repos := core.RepoContainer{User: userRepo}
	engine := NewWishlistEngine(repos, db, zap.NewNop())
	ctx := context.Background()
	asOf := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)

	t.Run("Missing User UUID", func(t *testing.T) {
		_, _, _, _, _, err := engine.CalculateCycleSurplus(ctx, uuid.Nil, asOf)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID error, got %v", err)
		}
	})

	t.Run("Positive Surplus", func(t *testing.T) {
		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount_e5\\), 0\\)").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(40000)))

		budget, expenses, surplus, _, _, err := engine.CalculateCycleSurplus(ctx, userUUID, asOf)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if budget != 100000 || expenses != 40000 || surplus != 60000 {
			t.Errorf("unexpected calculation: budget=%d, expenses=%d, surplus=%d", budget, expenses, surplus)
		}
	})

	t.Run("Overspending / Deficit (Surplus is 0)", func(t *testing.T) {
		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount_e5\\), 0\\)").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(120000)))

		_, expenses, surplus, _, _, err := engine.CalculateCycleSurplus(ctx, userUUID, asOf)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if expenses != 120000 || surplus != 0 {
			t.Errorf("unexpected calculation: expenses=%d, surplus=%d", expenses, surplus)
		}
	})
}

func TestWishlistEngine_SimulateDistribution(t *testing.T) {
	engine := NewWishlistEngine(core.RepoContainer{}, nil, zap.NewNop())
	item1ID := uuid.New()
	item2ID := uuid.New()

	item1 := &core.WishlistItem{
		ID:             item1ID,
		Title:          "Emergency Fund",
		TargetAmountE5: 100000,
		SavedAmountE5:  0,
		Priority:       5,
		Urgency:        4, // Weight = 20
		Status:         "active",
	}

	item2 := &core.WishlistItem{
		ID:             item2ID,
		Title:          "Headphones",
		TargetAmountE5: 100000,
		SavedAmountE5:  0,
		Priority:       2,
		Urgency:        2, // Weight = 4
		Status:         "active",
	}

	items := []*core.WishlistItem{item1, item2}

	t.Run("Zero Surplus", func(t *testing.T) {
		res := engine.SimulateDistribution(items, 0)
		if len(res) != 2 {
			t.Fatalf("expected 2 results, got %d", len(res))
		}
		if res[0].AllocatedE5 != 0 || res[1].AllocatedE5 != 0 {
			t.Errorf("expected 0 allocations, got %d, %d", res[0].AllocatedE5, res[1].AllocatedE5)
		}
	})

	t.Run("Proportional Weighted Distribution", func(t *testing.T) {
		// Total weight = 20 + 4 = 24.
		// Surplus = 24000.
		// Item 1 gets: 24000 * 20 / 24 = 20000.
		// Item 2 gets: 24000 * 4 / 24 = 4000.
		res := engine.SimulateDistribution(items, 24000)
		if len(res) != 2 {
			t.Fatalf("expected 2 results, got %d", len(res))
		}
		if res[0].AllocatedE5 != 20000 || res[1].AllocatedE5 != 4000 {
			t.Errorf("expected 20000 and 4000, got %d and %d", res[0].AllocatedE5, res[1].AllocatedE5)
		}
	})

	t.Run("Target Capping and Overflow Cascade", func(t *testing.T) {
		// Item 1 only needs 10000 to be fulfilled (target 100000, saved 90000)
		item1.SavedAmountE5 = 90000
		item2.SavedAmountE5 = 0

		// Surplus = 30000.
		// In round 1, item 1 share is 25000, but capped at 10000.
		// Leftover 20000 cascades to item 2.
		res := engine.SimulateDistribution(items, 30000)
		if res[0].AllocatedE5 != 10000 {
			t.Errorf("expected item 1 capped at 10000, got %d", res[0].AllocatedE5)
		}
		if res[0].NewSavedE5 != 100000 || !res[0].IsFulfilled {
			t.Errorf("expected item 1 fulfilled, got %+v", res[0])
		}
		if res[1].AllocatedE5 != 20000 {
			t.Errorf("expected item 2 to receive remaining 20000, got %d", res[1].AllocatedE5)
		}
	})
}

func TestWishlistEngine_CalculateForecasts(t *testing.T) {
	engine := NewWishlistEngine(core.RepoContainer{}, nil, zap.NewNop())
	asOf := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)

	item := &core.WishlistItem{
		ID:             uuid.New(),
		Title:          "Phone",
		TargetAmountE5: 100000,
		SavedAmountE5:  20000,
		Priority:       5,
		Urgency:        5,
		Status:         "active",
	}

	t.Run("Positive Monthly Surplus Forecast", func(t *testing.T) {
		// Budget = 50000, expenses = 30000 -> Surplus = 20000
		// Remaining = 80000. Monthly contrib = 20000.
		// Estimated months = 80000 / 20000 = 4 months.
		summary := engine.CalculateForecasts([]*core.WishlistItem{item}, 50000, 30000, asOf, 1)

		if summary.ProjectedSurplusE5 != 20000 {
			t.Errorf("expected projected surplus 20000, got %d", summary.ProjectedSurplusE5)
		}
		if summary.SavingsRatePercent != 40.0 {
			t.Errorf("expected 40%% savings rate, got %f", summary.SavingsRatePercent)
		}
		if len(summary.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(summary.Items))
		}
		if summary.Items[0].EstimatedMonths != 4.0 {
			t.Errorf("expected 4 estimated months, got %f", summary.Items[0].EstimatedMonths)
		}
		if summary.Items[0].ProgressPercentage != 20.0 {
			t.Errorf("expected 20%% progress, got %f", summary.Items[0].ProgressPercentage)
		}
	})

	t.Run("Zero Surplus / Overspent Forecast", func(t *testing.T) {
		summary := engine.CalculateForecasts([]*core.WishlistItem{item}, 50000, 60000, asOf, 1)

		if summary.ProjectedSurplusE5 != 0 {
			t.Errorf("expected 0 projected surplus, got %d", summary.ProjectedSurplusE5)
		}
		if summary.Items[0].EstimatedMonths != -1.0 || summary.Items[0].EstimatedDate != nil {
			t.Errorf("expected on hold (-1 months), got %f", summary.Items[0].EstimatedMonths)
		}
	})
}

func TestWishlistEngine_ApplySurplusDistribution(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	userUUID := uuid.New()
	itemUUID := uuid.New()
	asOf := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()

	item := &core.WishlistItem{
		ID:             itemUUID,
		UserUUID:       userUUID,
		Title:          "Gadget",
		TargetAmountE5: 50000,
		SavedAmountE5:  0,
		Priority:       5,
		Urgency:        5,
		Status:         "active",
	}

	userRepo := &mockUserRepo{
		getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
			return &core.User{UUID: userUUID, MonthlyBudgetE5: 100000, SalaryDay: 1}, nil
		},
	}

	t.Run("No Surplus Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(100000)))

		repos := core.RepoContainer{User: userRepo, Wishlist: &mockWishlistRepo{}}
		engine := NewWishlistEngine(repos, db, zap.NewNop())

		_, err := engine.ApplySurplusDistribution(ctx, userUUID, asOf)
		if err == nil || err.Error() != "no surplus available to distribute" {
			t.Errorf("expected 'no surplus available to distribute', got %v", err)
		}
	})

	t.Run("No Active Items Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(50000)))

		wishlistRepo := &mockWishlistRepo{
			getActiveItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return nil, nil
			},
		}

		repos := core.RepoContainer{User: userRepo, Wishlist: wishlistRepo}
		engine := NewWishlistEngine(repos, db, zap.NewNop())

		_, err := engine.ApplySurplusDistribution(ctx, userUUID, asOf)
		if err == nil || err.Error() != "no active wishlist items found" {
			t.Errorf("expected 'no active wishlist items found', got %v", err)
		}
	})

	t.Run("Success Allocation and Item Update", func(t *testing.T) {
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(50000)))

		mock.ExpectBegin()
		mock.ExpectCommit()

		wishlistRepo := &mockWishlistRepo{
			getActiveItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{item}, nil
			},
			getItemFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return item, nil
			},
			createAllocationFn: func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
				if alloc.AmountE5 != 50000 {
					t.Errorf("expected allocation 50000, got %d", alloc.AmountE5)
				}
				return uuid.New(), nil
			},
			updateItemFn: func(updated *core.WishlistItem, tx *sql.Tx) error {
				if updated.SavedAmountE5 != 50000 || updated.Status != "fulfilled" {
					t.Errorf("unexpected updated item: %+v", updated)
				}
				return nil
			},
		}

		repos := core.RepoContainer{User: userRepo, Wishlist: wishlistRepo}
		engine := NewWishlistEngine(repos, db, zap.NewNop())

		results, err := engine.ApplySurplusDistribution(ctx, userUUID, asOf)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(results) != 1 || results[0].AllocatedE5 != 50000 || !results[0].IsFulfilled {
			t.Errorf("unexpected simulation results: %+v", results)
		}
	})

	t.Run("Transaction Rollback on Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(50000)))

		mock.ExpectBegin()
		mock.ExpectRollback()

		wishlistRepo := &mockWishlistRepo{
			getActiveItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{item}, nil
			},
			createAllocationFn: func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("failed to insert allocation")
			},
		}

		repos := core.RepoContainer{User: userRepo, Wishlist: wishlistRepo}
		engine := NewWishlistEngine(repos, db, zap.NewNop())

		_, err := engine.ApplySurplusDistribution(ctx, userUUID, asOf)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("ApplySurplusDistribution - Error Cases", func(t *testing.T) {
		repos := core.RepoContainer{User: userRepo, Wishlist: &mockWishlistRepo{}}
		engine := NewWishlistEngine(repos, db, zap.NewNop())

		// Nil UUID
		if _, err := engine.ApplySurplusDistribution(ctx, uuid.Nil, asOf); err == nil {
			t.Error("expected error for nil user UUID")
		}

		// CalculateCycleSurplus error
		errUserRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return nil, errors.New("user not found")
			},
		}
		errEngine := NewWishlistEngine(core.RepoContainer{User: errUserRepo, Wishlist: &mockWishlistRepo{}}, db, zap.NewNop())
		if _, err := errEngine.ApplySurplusDistribution(ctx, userUUID, asOf); err == nil {
			t.Error("expected error for cycle surplus failure")
		}

		// GetActiveWishlistItems error
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(50000)))
		errWishlistRepo := &mockWishlistRepo{
			getActiveItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return nil, errors.New("db error")
			},
		}
		activeErrEngine := NewWishlistEngine(core.RepoContainer{User: userRepo, Wishlist: errWishlistRepo}, db, zap.NewNop())
		if _, err := activeErrEngine.ApplySurplusDistribution(ctx, userUUID, asOf); err == nil {
			t.Error("expected error for get active items error")
		}

		// GetWishlistItemByID error inside transaction
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(50000)))
		mock.ExpectBegin()
		mock.ExpectRollback()
		itemGetErrRepo := &mockWishlistRepo{
			getActiveItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{item}, nil
			},
			createAllocationFn: func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.New(), nil
			},
			getItemFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return nil, errors.New("get item failed")
			},
		}
		itemErrEngine := NewWishlistEngine(core.RepoContainer{User: userRepo, Wishlist: itemGetErrRepo}, db, zap.NewNop())
		if _, err := itemErrEngine.ApplySurplusDistribution(ctx, userUUID, asOf); err == nil {
			t.Error("expected error for get item by id failure")
		}

		// UpdateWishlistItem error inside transaction
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(50000)))
		mock.ExpectBegin()
		mock.ExpectRollback()
		updateErrRepo := &mockWishlistRepo{
			getActiveItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{item}, nil
			},
			createAllocationFn: func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.New(), nil
			},
			getItemFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return item, nil
			},
			updateItemFn: func(updated *core.WishlistItem, tx *sql.Tx) error {
				return errors.New("update item failed")
			},
		}
		updateErrEngine := NewWishlistEngine(core.RepoContainer{User: userRepo, Wishlist: updateErrRepo}, db, zap.NewNop())
		if _, err := updateErrEngine.ApplySurplusDistribution(ctx, userUUID, asOf); err == nil {
			t.Error("expected error for update item failure")
		}
	})
}

func TestWishlistEngine_CalculateCycleSurplus_Errors(t *testing.T) {
	ctx := context.Background()
	userUUID := uuid.New()
	asOf := time.Now()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected sqlmock error: %v", err)
	}
	defer db.Close()

	t.Run("Nil User UUID", func(t *testing.T) {
		engine := NewWishlistEngine(core.RepoContainer{}, db, zap.NewNop())
		_, _, _, _, _, err := engine.CalculateCycleSurplus(ctx, uuid.Nil, asOf)
		if err == nil {
			t.Error("expected error for nil user UUID")
		}
	})

	t.Run("User Fetch Error", func(t *testing.T) {
		userRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return nil, errors.New("user not found")
			},
		}
		engine := NewWishlistEngine(core.RepoContainer{User: userRepo}, db, zap.NewNop())
		_, _, _, _, _, err := engine.CalculateCycleSurplus(ctx, userUUID, asOf)
		if err == nil {
			t.Error("expected error when user fetch fails")
		}
	})

	t.Run("DB Query Error", func(t *testing.T) {
		userRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: userUUID, MonthlyBudgetE5: 10000000, SalaryDay: 1}, nil
			},
		}
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("db error"))

		engine := NewWishlistEngine(core.RepoContainer{User: userRepo}, db, zap.NewNop())
		_, _, _, _, _, err := engine.CalculateCycleSurplus(ctx, userUUID, asOf)
		if err == nil {
			t.Error("expected error when db query fails")
		}
	})

	t.Run("Expenses Exceed Budget (Negative Surplus Clamped)", func(t *testing.T) {
		userRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: userUUID, MonthlyBudgetE5: 10000000, SalaryDay: 1}, nil
			},
		}
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(15000000)))

		engine := NewWishlistEngine(core.RepoContainer{User: userRepo}, db, zap.NewNop())
		budget, expenses, surplus, _, _, err := engine.CalculateCycleSurplus(ctx, userUUID, asOf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if budget != 10000000 || expenses != 15000000 || surplus != 0 {
			t.Errorf("expected clamped surplus 0, got %d", surplus)
		}
	})
}

func TestWishlistEngine_SimulateDistribution_EdgeCases(t *testing.T) {
	engine := NewWishlistEngine(core.RepoContainer{}, nil, zap.NewNop())

	t.Run("Non-active Items Skipped", func(t *testing.T) {
		items := []*core.WishlistItem{
			{ID: uuid.New(), Title: "Fulfilled Item", TargetAmountE5: 1000, SavedAmountE5: 1000, Status: "fulfilled"},
			{ID: uuid.New(), Title: "Active Item", TargetAmountE5: 1000, SavedAmountE5: 0, Priority: 0, Urgency: 0, Status: "active"},
		}
		results := engine.SimulateDistribution(items, 500)
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0].AllocatedE5 != 0 {
			t.Errorf("expected 0 allocated for fulfilled item, got %d", results[0].AllocatedE5)
		}
		if results[1].AllocatedE5 != 500 {
			t.Errorf("expected 500 allocated for active item, got %d", results[1].AllocatedE5)
		}
	})

	t.Run("Empty Items or Zero Surplus", func(t *testing.T) {
		res1 := engine.SimulateDistribution(nil, 1000)
		if len(res1) != 0 {
			t.Errorf("expected empty results for nil items")
		}
		res2 := engine.SimulateDistribution([]*core.WishlistItem{}, 0)
		if len(res2) != 0 {
			t.Errorf("expected empty results for 0 surplus")
		}
	})
}

func TestWishlistEngine_CalculateForecasts_EdgeCases(t *testing.T) {
	engine := NewWishlistEngine(core.RepoContainer{}, nil, zap.NewNop())
	now := time.Now()

	t.Run("Zero or Negative Monthly Surplus", func(t *testing.T) {
		items := []*core.WishlistItem{
			{ID: uuid.New(), Title: "Item 1", TargetAmountE5: 10000, SavedAmountE5: 2000, Priority: 3, Urgency: 3, Status: "active"},
		}
		summary := engine.CalculateForecasts(items, 50000, 60000, now, 1)
		if summary.ProjectedSurplusE5 != 0 {
			t.Errorf("expected 0 projected surplus, got %d", summary.ProjectedSurplusE5)
		}
		if len(summary.Items) != 1 || summary.Items[0].EstimatedMonths != -1 {
			t.Errorf("expected -1 estimated months for zero surplus, got %+v", summary.Items[0])
		}
	})

	t.Run("Already Fulfilled Item in Forecast", func(t *testing.T) {
		items := []*core.WishlistItem{
			{ID: uuid.New(), Title: "Done Item", TargetAmountE5: 10000, SavedAmountE5: 15000, Priority: 0, Urgency: 0, Status: "fulfilled"},
		}
		summary := engine.CalculateForecasts(items, 50000, 20000, now, 1)
		if len(summary.Items) != 1 || summary.Items[0].EstimatedMonths != 0 || summary.Items[0].ProgressPercentage != 100.0 {
			t.Errorf("expected 0 estimated months and 100%% progress for over-fulfilled item, got %+v", summary.Items[0])
		}
	})

	t.Run("Salary Day Greater than 28 Capped", func(t *testing.T) {
		start, end := engine.GetCycleDateRange(31, time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
		if start.Day() != 28 {
			t.Errorf("expected salary day capped at 28, got %d", start.Day())
		}
		_ = end
	})

	t.Run("SimulateDistribution - Round Remainder Allocation", func(t *testing.T) {
		items := []*core.WishlistItem{
			{ID: uuid.New(), Title: "Item 1", TargetAmountE5: 100, SavedAmountE5: 0, Priority: 1, Urgency: 1, Status: "active"},
			{ID: uuid.New(), Title: "Item 2", TargetAmountE5: 100, SavedAmountE5: 0, Priority: 100, Urgency: 100, Status: "active"},
		}
		// Surplus 1: share for item 1 is 0, so allocatedInRound == 0 remainder logic fires
		res := engine.SimulateDistribution(items, 1)
		if len(res) != 2 {
			t.Fatalf("expected 2 results")
		}
	})

	t.Run("ApplySurplusDistribution - BeginTx Error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		userUUID := uuid.New()
		userRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: userUUID, MonthlyBudgetE5: 50000, SalaryDay: 1}, nil
			},
		}
		wishlistRepo := &mockWishlistRepo{
			getActiveItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{
					{ID: uuid.New(), Title: "Item", TargetAmountE5: 1000, SavedAmountE5: 0, Priority: 3, Urgency: 3, Status: "active"},
				}, nil
			},
		}

		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(10000)))
		mock.ExpectBegin().WillReturnError(errors.New("begin tx failed"))

		eng := NewWishlistEngine(core.RepoContainer{User: userRepo, Wishlist: wishlistRepo}, db, zap.NewNop())
		if _, err := eng.ApplySurplusDistribution(context.Background(), userUUID, time.Now()); err == nil {
			t.Error("expected error for begin tx failure")
		}
	})

	t.Run("ApplySurplusDistribution - Commit Error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		userUUID := uuid.New()
		userRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: userUUID, MonthlyBudgetE5: 50000, SalaryDay: 1}, nil
			},
		}
		item := &core.WishlistItem{ID: uuid.New(), Title: "Item", TargetAmountE5: 1000, SavedAmountE5: 0, Priority: 3, Urgency: 3, Status: "active"}
		wishlistRepo := &mockWishlistRepo{
			getActiveItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{item}, nil
			},
			createAllocationFn: func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.New(), nil
			},
			getItemFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return item, nil
			},
			updateItemFn: func(updated *core.WishlistItem, tx *sql.Tx) error {
				return nil
			},
		}

		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(10000)))
		mock.ExpectBegin()
		mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

		eng := NewWishlistEngine(core.RepoContainer{User: userRepo, Wishlist: wishlistRepo}, db, zap.NewNop())
		if _, err := eng.ApplySurplusDistribution(context.Background(), userUUID, time.Now()); err == nil {
			t.Error("expected error for commit failure")
		}
	})

	t.Run("ApplyManualAllocation - Validation and Success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to open sqlmock: %v", err)
		}
		defer db.Close()

		userUUID := uuid.New()
		itemID := uuid.New()
		item := &core.WishlistItem{
			ID:             itemID,
			UserUUID:       userUUID,
			Title:          "Guitar",
			TargetAmountE5: 10000,
			SavedAmountE5:  4000,
			Priority:       4,
			Urgency:        3,
			Status:         "active",
		}

		wishlistRepo := &mockWishlistRepo{
			getItemFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				if id == itemID {
					// Return copy
					cp := *item
					return &cp, nil
				}
				return nil, errors.New("not found")
			},
			createAllocationFn: func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.New(), nil
			},
			updateItemFn: func(updated *core.WishlistItem, tx *sql.Tx) error {
				return nil
			},
		}

		eng := NewWishlistEngine(core.RepoContainer{Wishlist: wishlistRepo}, db, zap.NewNop())

		// Nil checks
		if _, err := eng.ApplyManualAllocation(context.Background(), uuid.Nil, itemID, 1000, time.Now()); err == nil {
			t.Error("expected error for nil userUUID")
		}
		if _, err := eng.ApplyManualAllocation(context.Background(), userUUID, uuid.Nil, 1000, time.Now()); err == nil {
			t.Error("expected error for nil itemID")
		}

		// Wrong user
		diffUser := uuid.New()
		if _, err := eng.ApplyManualAllocation(context.Background(), diffUser, itemID, 1000, time.Now()); err == nil {
			t.Error("expected error when item belongs to different user")
		}

		// Success partial allocation
		mock.ExpectBegin()
		mock.ExpectCommit()
		sim, err := eng.ApplyManualAllocation(context.Background(), userUUID, itemID, 2000, time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sim.AllocatedE5 != 2000 || sim.NewSavedE5 != 6000 || sim.IsFulfilled {
			t.Errorf("unexpected simulation result: %+v", sim)
		}

		// Success full allocation (amountE5 = 0 means allocate remaining needed)
		mock.ExpectBegin()
		mock.ExpectCommit()
		simFull, err := eng.ApplyManualAllocation(context.Background(), userUUID, itemID, 0, time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if simFull.AllocatedE5 != 6000 || simFull.NewSavedE5 != 10000 || !simFull.IsFulfilled {
			t.Errorf("unexpected full simulation result: %+v", simFull)
		}
	})

	t.Run("ApplyManualAllocation - Additional Error Branches", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		defer db.Close()

		userUUID := uuid.New()
		itemID := uuid.New()

		// 1. GetItem error
		wishlistRepo := &mockWishlistRepo{
			getItemFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return nil, errors.New("db error")
			},
		}
		eng := NewWishlistEngine(core.RepoContainer{Wishlist: wishlistRepo}, db, zap.NewNop())
		if _, err := eng.ApplyManualAllocation(context.Background(), userUUID, itemID, 1000, time.Now()); err == nil {
			t.Error("expected error when GetWishlistItemByID fails")
		}

		// 2. Item already fulfilled
		fulfilledItem := &core.WishlistItem{
			ID: itemID, UserUUID: userUUID, TargetAmountE5: 1000, SavedAmountE5: 1000, Status: "fulfilled",
		}
		wishlistRepo.getItemFn = func(id uuid.UUID) (*core.WishlistItem, error) {
			return fulfilledItem, nil
		}
		if _, err := eng.ApplyManualAllocation(context.Background(), userUUID, itemID, 1000, time.Now()); err == nil {
			t.Error("expected error when item already fulfilled")
		}

		// 3. BeginTx error
		activeItem := &core.WishlistItem{
			ID: itemID, UserUUID: userUUID, TargetAmountE5: 5000, SavedAmountE5: 1000, Status: "active",
		}
		wishlistRepo.getItemFn = func(id uuid.UUID) (*core.WishlistItem, error) {
			cp := *activeItem
			return &cp, nil
		}
		mock.ExpectBegin().WillReturnError(errors.New("begin error"))
		if _, err := eng.ApplyManualAllocation(context.Background(), userUUID, itemID, 1000, time.Now()); err == nil {
			t.Error("expected error when begin tx fails")
		}

		// 4. CreateAllocation error
		mock.ExpectBegin()
		wishlistRepo.createAllocationFn = func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
			return uuid.Nil, errors.New("insert error")
		}
		mock.ExpectRollback()
		if _, err := eng.ApplyManualAllocation(context.Background(), userUUID, itemID, 1000, time.Now()); err == nil {
			t.Error("expected error when create allocation fails")
		}

		// 5. UpdateItem error
		mock.ExpectBegin()
		wishlistRepo.createAllocationFn = func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
			return uuid.New(), nil
		}
		wishlistRepo.updateItemFn = func(item *core.WishlistItem, tx *sql.Tx) error {
			return errors.New("update error")
		}
		mock.ExpectRollback()
		if _, err := eng.ApplyManualAllocation(context.Background(), userUUID, itemID, 1000, time.Now()); err == nil {
			t.Error("expected error when update item fails")
		}

		// 6. Commit error
		mock.ExpectBegin()
		wishlistRepo.updateItemFn = func(item *core.WishlistItem, tx *sql.Tx) error {
			return nil
		}
		mock.ExpectCommit().WillReturnError(errors.New("commit error"))
		if _, err := eng.ApplyManualAllocation(context.Background(), userUUID, itemID, 1000, time.Now()); err == nil {
			t.Error("expected error when commit fails")
		}
	})

	t.Run("ApplySurplusDistribution - Error Branches", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		defer db.Close()

		userUUID := uuid.New()
		userRepo := &mockUserRepo{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: userUUID, MonthlyBudgetE5: 10000, SalaryDay: 1}, nil
			},
		}

		// 1. Surplus <= 0
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(10000))) // expenses == budget => surplus 0
		eng := NewWishlistEngine(core.RepoContainer{User: userRepo}, db, zap.NewNop())
		if _, err := eng.ApplySurplusDistribution(context.Background(), userUUID, time.Now()); err == nil {
			t.Error("expected error when surplus is 0")
		}

		// 2. GetActiveWishlistItems error
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(2000)))
		wishlistRepo := &mockWishlistRepo{
			getActiveItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return nil, errors.New("db error")
			},
		}
		eng = NewWishlistEngine(core.RepoContainer{User: userRepo, Wishlist: wishlistRepo}, db, zap.NewNop())
		if _, err := eng.ApplySurplusDistribution(context.Background(), userUUID, time.Now()); err == nil {
			t.Error("expected error when getActiveItems fails")
		}

		// 3. No active items (len == 0)
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(2000)))
		wishlistRepo.getActiveItemsFn = func(u uuid.UUID) ([]*core.WishlistItem, error) {
			return []*core.WishlistItem{}, nil
		}
		if _, err := eng.ApplySurplusDistribution(context.Background(), userUUID, time.Now()); err == nil {
			t.Error("expected error when active items is empty")
		}

		// 4. CreateAllocation error
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(2000)))
		item := &core.WishlistItem{ID: uuid.New(), Title: "Item", TargetAmountE5: 1000, SavedAmountE5: 0, Priority: 3, Urgency: 3, Status: "active"}
		wishlistRepo.getActiveItemsFn = func(u uuid.UUID) ([]*core.WishlistItem, error) {
			return []*core.WishlistItem{item}, nil
		}
		mock.ExpectBegin()
		wishlistRepo.createAllocationFn = func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
			return uuid.Nil, errors.New("create alloc failed")
		}
		mock.ExpectRollback()
		if _, err := eng.ApplySurplusDistribution(context.Background(), userUUID, time.Now()); err == nil {
			t.Error("expected error when create allocation fails")
		}

		// 5. GetItem error during update
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(2000)))
		mock.ExpectBegin()
		wishlistRepo.createAllocationFn = func(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
			return uuid.New(), nil
		}
		wishlistRepo.getItemFn = func(id uuid.UUID) (*core.WishlistItem, error) {
			return nil, errors.New("get item failed")
		}
		mock.ExpectRollback()
		if _, err := eng.ApplySurplusDistribution(context.Background(), userUUID, time.Now()); err == nil {
			t.Error("expected error when get item fails")
		}

		// 6. UpdateItem error
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(2000)))
		mock.ExpectBegin()
		wishlistRepo.getItemFn = func(id uuid.UUID) (*core.WishlistItem, error) {
			return item, nil
		}
		wishlistRepo.updateItemFn = func(i *core.WishlistItem, tx *sql.Tx) error {
			return errors.New("update item failed")
		}
		mock.ExpectRollback()
		if _, err := eng.ApplySurplusDistribution(context.Background(), userUUID, time.Now()); err == nil {
			t.Error("expected error when update item fails")
		}
	})
}

