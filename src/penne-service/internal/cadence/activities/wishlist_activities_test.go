package activities

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockWishlistRepoForActivity struct {
	core.WishlistRepository
	getItemsFn         func(uuid.UUID) ([]*core.WishlistItem, error)
	getActiveItemsFn   func(uuid.UUID) ([]*core.WishlistItem, error)
	getItemFn          func(uuid.UUID) (*core.WishlistItem, error)
	createAllocationFn func(*core.WishlistAllocation, *sql.Tx) (uuid.UUID, error)
	updateItemFn       func(*core.WishlistItem, *sql.Tx) error
}

func (m *mockWishlistRepoForActivity) GetWishlistItemsByUserUUID(id uuid.UUID) ([]*core.WishlistItem, error) {
	if m.getItemsFn != nil {
		return m.getItemsFn(id)
	}
	return nil, nil
}

func (m *mockWishlistRepoForActivity) GetActiveWishlistItemsByUserUUID(id uuid.UUID) ([]*core.WishlistItem, error) {
	if m.getActiveItemsFn != nil {
		return m.getActiveItemsFn(id)
	}
	return nil, nil
}

func (m *mockWishlistRepoForActivity) GetWishlistItemByID(id uuid.UUID) (*core.WishlistItem, error) {
	if m.getItemFn != nil {
		return m.getItemFn(id)
	}
	return &core.WishlistItem{ID: id, Title: "Item", TargetAmountE5: 1000, SavedAmountE5: 0, Priority: 3, Urgency: 3, Status: "active"}, nil
}

func (m *mockWishlistRepoForActivity) CreateWishlistAllocation(alloc *core.WishlistAllocation, tx *sql.Tx) (uuid.UUID, error) {
	if m.createAllocationFn != nil {
		return m.createAllocationFn(alloc, tx)
	}
	return uuid.New(), nil
}

func (m *mockWishlistRepoForActivity) UpdateWishlistItem(item *core.WishlistItem, tx *sql.Tx) error {
	if m.updateItemFn != nil {
		return m.updateItemFn(item, tx)
	}
	return nil
}

func TestWishlistActivities(t *testing.T) {
	logger := zap.NewNop()
	userUUID := uuid.New()
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected sqlmock error: %v", err)
	}
	defer db.Close()

	userRepo := &mockUserRepo{
		createFn: func(u *core.User, tx *sql.Tx) (uuid.UUID, error) {
			return userUUID, nil
		},
	}

	t.Run("CalculateWishlistForecastActivity - Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(20000)))

		wishlistRepo := &mockWishlistRepoForActivity{
			getItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{
					{
						ID:             uuid.New(),
						Title:          "Laptop",
						TargetAmountE5: 100000,
						SavedAmountE5:  10000,
						Priority:       5,
						Urgency:        5,
						Status:         "active",
					},
				}, nil
			},
		}

		userWithBudget := &mockUserRepoWithBudget{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: userUUID, MonthlyBudgetE5: 50000, SalaryDay: 1}, nil
			},
		}

		repos := core.RepoContainer{User: userWithBudget, Wishlist: wishlistRepo}
		acts := NewWishlistActivities(repos, db, logger)

		forecast, err := acts.CalculateWishlistForecastActivity(ctx, userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if forecast.ProjectedSurplusE5 != 30000 {
			t.Errorf("expected projected surplus 30000, got %d", forecast.ProjectedSurplusE5)
		}
	})

	t.Run("ApplyWishlistSurplusActivity - Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(int64(20000)))

		mock.ExpectBegin()
		mock.ExpectCommit()

		item := &core.WishlistItem{
			ID:             uuid.New(),
			UserUUID:       userUUID,
			Title:          "Laptop",
			TargetAmountE5: 100000,
			SavedAmountE5:  10000,
			Priority:       5,
			Urgency:        5,
			Status:         "active",
		}

		wishlistRepo := &mockWishlistRepoForActivity{
			getActiveItemsFn: func(u uuid.UUID) ([]*core.WishlistItem, error) {
				return []*core.WishlistItem{item}, nil
			},
			getItemFn: func(id uuid.UUID) (*core.WishlistItem, error) {
				return item, nil
			},
		}

		userWithBudget := &mockUserRepoWithBudget{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: userUUID, MonthlyBudgetE5: 50000, SalaryDay: 1}, nil
			},
		}

		repos := core.RepoContainer{User: userWithBudget, Wishlist: wishlistRepo}
		acts := NewWishlistActivities(repos, db, logger)

		results, err := acts.ApplyWishlistSurplusActivity(ctx, userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(results) != 1 || results[0].AllocatedE5 != 30000 {
			t.Errorf("unexpected results: %+v", results)
		}
	})

	t.Run("ApplyWishlistSurplusActivity - Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COALESCE").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("db query error"))

		userWithBudget := &mockUserRepoWithBudget{
			getUserByUUIDFn: func(id uuid.UUID) (*core.User, error) {
				return &core.User{UUID: userUUID, MonthlyBudgetE5: 50000, SalaryDay: 1}, nil
			},
		}

		repos := core.RepoContainer{User: userWithBudget, Wishlist: &mockWishlistRepoForActivity{}}
		acts := NewWishlistActivities(repos, db, logger)

		_, err := acts.ApplyWishlistSurplusActivity(ctx, userUUID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
	_ = userRepo
}

type mockUserRepoWithBudget struct {
	core.UserRepository
	getUserByUUIDFn func(uuid.UUID) (*core.User, error)
}

func (m *mockUserRepoWithBudget) GetUserByUUID(id uuid.UUID) (*core.User, error) {
	if m.getUserByUUIDFn != nil {
		return m.getUserByUUIDFn(id)
	}
	return &core.User{UUID: id, MonthlyBudgetE5: 50000, SalaryDay: 1}, nil
}
