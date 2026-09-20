package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
)

func TestPgWishlistRepo_WishlistItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgWishlistRepo(db)
	userUUID := uuid.New()
	itemUUID := uuid.New()

	t.Run("CreateWishlistItem - Validation Errors", func(t *testing.T) {
		// Missing user UUID
		_, err := repo.CreateWishlistItem(&core.WishlistItem{Title: "Laptop", TargetAmountE5: 1000}, nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID error, got %v", err)
		}

		// Missing title
		_, err = repo.CreateWishlistItem(&core.WishlistItem{UserUUID: userUUID, Title: "", TargetAmountE5: 1000}, nil)
		if err == nil || err.Error() != "title is required" {
			t.Errorf("expected title error, got %v", err)
		}

		// Target amount <= 0
		_, err = repo.CreateWishlistItem(&core.WishlistItem{UserUUID: userUUID, Title: "Laptop", TargetAmountE5: 0}, nil)
		if err == nil || err.Error() != "target amount must be greater than zero" {
			t.Errorf("expected target amount error, got %v", err)
		}
	})

	t.Run("CreateWishlistItem - Success", func(t *testing.T) {
		item := &core.WishlistItem{
			UserUUID:       userUUID,
			Title:          "MacBook Pro",
			TargetAmountE5: 20000000000,
			Priority:       4,
			Urgency:        5,
		}

		mock.ExpectQuery("INSERT INTO wishlist_items").
			WithArgs(
				item.UserUUID, item.Title, item.TargetAmountE5, int64(0),
				4, 5, "purchase", "active", nil, "",
				sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(itemUUID))

		id, err := repo.CreateWishlistItem(item, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if id != itemUUID {
			t.Errorf("expected ID %v, got %v", itemUUID, id)
		}
	})

	t.Run("CreateWishlistItem - With Tx", func(t *testing.T) {
		item := &core.WishlistItem{
			UserUUID:       userUUID,
			Title:          "MacBook Pro",
			TargetAmountE5: 20000000000,
		}

		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO wishlist_items").
			WithArgs(
				item.UserUUID, item.Title, item.TargetAmountE5, int64(0),
				3, 3, "purchase", "active", nil, "",
				sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(itemUUID))

		id, err := repo.CreateWishlistItem(item, tx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if id != itemUUID {
			t.Errorf("expected ID %v, got %v", itemUUID, id)
		}
	})

	t.Run("GetWishlistItemByID - Validation and Query", func(t *testing.T) {
		_, err := repo.GetWishlistItemByID(uuid.Nil)
		if err == nil || err.Error() != "wishlist item ID is required" {
			t.Errorf("expected wishlist item ID error, got %v", err)
		}

		mock.ExpectQuery("SELECT (.+) FROM wishlist_items WHERE id = \\$1").
			WithArgs(itemUUID).
			WillReturnError(sql.ErrNoRows)

		_, err = repo.GetWishlistItemByID(itemUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "user_uuid", "title", "target_amount_e5", "saved_amount_e5",
			"priority", "urgency", "item_type", "status", "target_date", "notes",
			"created_at", "updated_at",
		}).AddRow(
			itemUUID, userUUID, "MacBook Pro", int64(20000000), int64(5000000),
			4, 5, "purchase", "active", nil, "notes", now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM wishlist_items WHERE id = \\$1").
			WithArgs(itemUUID).
			WillReturnRows(rows)

		item, err := repo.GetWishlistItemByID(itemUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if item.Title != "MacBook Pro" || item.SavedAmountE5 != 5000000 {
			t.Errorf("unexpected item: %+v", item)
		}
	})

	t.Run("GetWishlistItemsByUserUUID - Success", func(t *testing.T) {
		_, err := repo.GetWishlistItemsByUserUUID(uuid.Nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID error, got %v", err)
		}

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "user_uuid", "title", "target_amount_e5", "saved_amount_e5",
			"priority", "urgency", "item_type", "status", "target_date", "notes",
			"created_at", "updated_at",
		}).
			AddRow(itemUUID, userUUID, "Item 1", int64(100), int64(0), 3, 3, "purchase", "active", nil, "", now, now).
			AddRow(uuid.New(), userUUID, "Item 2", int64(200), int64(50), 4, 4, "purchase", "active", nil, "", now, now)

		mock.ExpectQuery("SELECT (.+) FROM wishlist_items WHERE user_uuid = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		items, err := repo.GetWishlistItemsByUserUUID(userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(items) != 2 {
			t.Errorf("expected 2 items, got %d", len(items))
		}
	})

	t.Run("GetActiveWishlistItemsByUserUUID - Success", func(t *testing.T) {
		_, err := repo.GetActiveWishlistItemsByUserUUID(uuid.Nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID error, got %v", err)
		}

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "user_uuid", "title", "target_amount_e5", "saved_amount_e5",
			"priority", "urgency", "item_type", "status", "target_date", "notes",
			"created_at", "updated_at",
		}).
			AddRow(itemUUID, userUUID, "Active Item", int64(100), int64(0), 5, 5, "purchase", "active", nil, "", now, now)

		mock.ExpectQuery("SELECT (.+) FROM wishlist_items WHERE user_uuid = \\$1 AND status = 'active'").
			WithArgs(userUUID).
			WillReturnRows(rows)

		items, err := repo.GetActiveWishlistItemsByUserUUID(userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}
	})

	t.Run("UpdateWishlistItem - Validations and Success", func(t *testing.T) {
		// Missing ID
		err := repo.UpdateWishlistItem(&core.WishlistItem{Title: "Title", TargetAmountE5: 100, Priority: 3, Urgency: 3}, nil)
		if err == nil || err.Error() != "wishlist item ID is required" {
			t.Errorf("expected wishlist item ID error, got %v", err)
		}

		// Empty Title
		err = repo.UpdateWishlistItem(&core.WishlistItem{ID: itemUUID, Title: "", TargetAmountE5: 100, Priority: 3, Urgency: 3}, nil)
		if err == nil || err.Error() != "title is required" {
			t.Errorf("expected title error, got %v", err)
		}

		// TargetAmount <= 0
		err = repo.UpdateWishlistItem(&core.WishlistItem{ID: itemUUID, Title: "Title", TargetAmountE5: 0, Priority: 3, Urgency: 3}, nil)
		if err == nil || err.Error() != "target amount must be greater than zero" {
			t.Errorf("expected target amount error, got %v", err)
		}

		// Priority out of bounds
		err = repo.UpdateWishlistItem(&core.WishlistItem{ID: itemUUID, Title: "Title", TargetAmountE5: 100, Priority: 6, Urgency: 3}, nil)
		if err == nil || err.Error() != "priority must be between 1 and 5" {
			t.Errorf("expected priority error, got %v", err)
		}

		// Urgency out of bounds
		err = repo.UpdateWishlistItem(&core.WishlistItem{ID: itemUUID, Title: "Title", TargetAmountE5: 100, Priority: 3, Urgency: 0}, nil)
		if err == nil || err.Error() != "urgency must be between 1 and 5" {
			t.Errorf("expected urgency error, got %v", err)
		}

		item := &core.WishlistItem{
			ID:             itemUUID,
			Title:          "Updated Title",
			TargetAmountE5: 500000,
			SavedAmountE5:  100000,
			Priority:       4,
			Urgency:        4,
			ItemType:       "purchase",
			Status:         "active",
		}

		mock.ExpectExec("UPDATE wishlist_items SET").
			WithArgs(
				item.Title, item.TargetAmountE5, item.SavedAmountE5,
				item.Priority, item.Urgency, item.ItemType, item.Status,
				nil, "", item.ID,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.UpdateWishlistItem(item, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("DeleteWishlistItem - Success", func(t *testing.T) {
		err := repo.DeleteWishlistItem(uuid.Nil)
		if err == nil || err.Error() != "wishlist item ID is required" {
			t.Errorf("expected wishlist item ID error, got %v", err)
		}

		mock.ExpectExec("DELETE FROM wishlist_items WHERE id = \\$1").
			WithArgs(itemUUID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.DeleteWishlistItem(itemUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestPgWishlistRepo_WishlistAllocations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgWishlistRepo(db)
	userUUID := uuid.New()
	itemUUID := uuid.New()
	allocUUID := uuid.New()

	t.Run("CreateWishlistAllocation - Validation", func(t *testing.T) {
		// Missing item ID
		_, err := repo.CreateWishlistAllocation(&core.WishlistAllocation{UserUUID: userUUID, AmountE5: 100}, nil)
		if err == nil || err.Error() != "wishlist item ID is required" {
			t.Errorf("expected item ID error, got %v", err)
		}

		// Missing user UUID
		_, err = repo.CreateWishlistAllocation(&core.WishlistAllocation{WishlistItemID: itemUUID, AmountE5: 100}, nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID error, got %v", err)
		}

		// Amount <= 0
		_, err = repo.CreateWishlistAllocation(&core.WishlistAllocation{WishlistItemID: itemUUID, UserUUID: userUUID, AmountE5: 0}, nil)
		if err == nil || err.Error() != "allocation amount must be greater than zero" {
			t.Errorf("expected allocation amount error, got %v", err)
		}
	})

	t.Run("CreateWishlistAllocation - Success", func(t *testing.T) {
		alloc := &core.WishlistAllocation{
			WishlistItemID: itemUUID,
			UserUUID:       userUUID,
			AmountE5:       500000,
		}

		mock.ExpectQuery("INSERT INTO wishlist_allocations").
			WithArgs(
				alloc.WishlistItemID, alloc.UserUUID, alloc.AmountE5, "cycle_surplus",
				sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(allocUUID))

		id, err := repo.CreateWishlistAllocation(alloc, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if id != allocUUID {
			t.Errorf("expected ID %v, got %v", allocUUID, id)
		}
	})

	t.Run("GetWishlistAllocationsByItemID - Success", func(t *testing.T) {
		_, err := repo.GetWishlistAllocationsByItemID(uuid.Nil)
		if err == nil || err.Error() != "wishlist item ID is required" {
			t.Errorf("expected wishlist item ID error, got %v", err)
		}

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "wishlist_item_id", "user_uuid", "amount_e5", "source_type", "cycle_date", "created_at",
		}).AddRow(allocUUID, itemUUID, userUUID, int64(500000), "cycle_surplus", now, now)

		mock.ExpectQuery("SELECT (.+) FROM wishlist_allocations WHERE wishlist_item_id = \\$1").
			WithArgs(itemUUID).
			WillReturnRows(rows)

		allocs, err := repo.GetWishlistAllocationsByItemID(itemUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(allocs) != 1 {
			t.Errorf("expected 1 allocation, got %d", len(allocs))
		}
	})

	t.Run("GetWishlistAllocationsByUserUUID - Success", func(t *testing.T) {
		_, err := repo.GetWishlistAllocationsByUserUUID(uuid.Nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID error, got %v", err)
		}

		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "wishlist_item_id", "user_uuid", "amount_e5", "source_type", "cycle_date", "created_at",
		}).AddRow(allocUUID, itemUUID, userUUID, int64(500000), "cycle_surplus", now, now)

		mock.ExpectQuery("SELECT (.+) FROM wishlist_allocations WHERE user_uuid = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		allocs, err := repo.GetWishlistAllocationsByUserUUID(userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(allocs) != 1 {
			t.Errorf("expected 1 allocation, got %d", len(allocs))
		}
	})
}
