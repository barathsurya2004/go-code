package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
)

type pgWishlistRepo struct {
	db *sql.DB
}

func NewPgWishlistRepo(db *sql.DB) core.WishlistRepository {
	return &pgWishlistRepo{
		db: db,
	}
}

func (r *pgWishlistRepo) CreateWishlistItem(item *core.WishlistItem, Tx *sql.Tx) (uuid.UUID, error) {
	if item.UserUUID == uuid.Nil {
		return uuid.Nil, errors.New("user UUID is required")
	}
	if item.Title == "" {
		return uuid.Nil, errors.New("title is required")
	}
	if item.TargetAmountE5 <= 0 {
		return uuid.Nil, errors.New("target amount must be greater than zero")
	}
	if item.Priority < 1 || item.Priority > 5 {
		item.Priority = 3
	}
	if item.Urgency < 1 || item.Urgency > 5 {
		item.Urgency = 3
	}
	if item.ItemType == "" {
		item.ItemType = "purchase"
	}
	if item.Status == "" {
		item.Status = "active"
	}

	now := utils.NowUTC()
	item.CreatedAt = now
	item.UpdatedAt = now

	query := `
		INSERT INTO wishlist_items (
			user_uuid, title, target_amount_e5, saved_amount_e5,
			priority, urgency, item_type, status, target_date, notes,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var id uuid.UUID
	var err error
	if Tx != nil {
		err = Tx.QueryRowContext(context.Background(), query,
			item.UserUUID, item.Title, item.TargetAmountE5, item.SavedAmountE5,
			item.Priority, item.Urgency, item.ItemType, item.Status, item.TargetDate, item.Notes,
			item.CreatedAt, item.UpdatedAt,
		).Scan(&id)
	} else {
		err = r.db.QueryRowContext(context.Background(), query,
			item.UserUUID, item.Title, item.TargetAmountE5, item.SavedAmountE5,
			item.Priority, item.Urgency, item.ItemType, item.Status, item.TargetDate, item.Notes,
			item.CreatedAt, item.UpdatedAt,
		).Scan(&id)
	}
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *pgWishlistRepo) GetWishlistItemByID(id uuid.UUID) (*core.WishlistItem, error) {
	if id == uuid.Nil {
		return nil, errors.New("wishlist item ID is required")
	}

	query := `
		SELECT id, user_uuid, title, target_amount_e5, saved_amount_e5,
		       priority, urgency, item_type, status, target_date, notes,
		       created_at, updated_at
		FROM wishlist_items
		WHERE id = $1
	`

	item := &core.WishlistItem{}
	err := r.db.QueryRowContext(context.Background(), query, id).Scan(
		&item.ID, &item.UserUUID, &item.Title, &item.TargetAmountE5, &item.SavedAmountE5,
		&item.Priority, &item.Urgency, &item.ItemType, &item.Status, &item.TargetDate, &item.Notes,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *pgWishlistRepo) GetWishlistItemsByUserUUID(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}

	query := `
		SELECT id, user_uuid, title, target_amount_e5, saved_amount_e5,
		       priority, urgency, item_type, status, target_date, notes,
		       created_at, updated_at
		FROM wishlist_items
		WHERE user_uuid = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(context.Background(), query, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*core.WishlistItem
	for rows.Next() {
		item := &core.WishlistItem{}
		if err := rows.Scan(
			&item.ID, &item.UserUUID, &item.Title, &item.TargetAmountE5, &item.SavedAmountE5,
			&item.Priority, &item.Urgency, &item.ItemType, &item.Status, &item.TargetDate, &item.Notes,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *pgWishlistRepo) GetActiveWishlistItemsByUserUUID(userUUID uuid.UUID) ([]*core.WishlistItem, error) {
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}

	query := `
		SELECT id, user_uuid, title, target_amount_e5, saved_amount_e5,
		       priority, urgency, item_type, status, target_date, notes,
		       created_at, updated_at
		FROM wishlist_items
		WHERE user_uuid = $1 AND status = 'active'
		ORDER BY (priority * urgency) DESC, created_at ASC
	`

	rows, err := r.db.QueryContext(context.Background(), query, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*core.WishlistItem
	for rows.Next() {
		item := &core.WishlistItem{}
		if err := rows.Scan(
			&item.ID, &item.UserUUID, &item.Title, &item.TargetAmountE5, &item.SavedAmountE5,
			&item.Priority, &item.Urgency, &item.ItemType, &item.Status, &item.TargetDate, &item.Notes,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *pgWishlistRepo) UpdateWishlistItem(item *core.WishlistItem, Tx *sql.Tx) error {
	if item.ID == uuid.Nil {
		return errors.New("wishlist item ID is required")
	}
	if item.Title == "" {
		return errors.New("title is required")
	}
	if item.TargetAmountE5 <= 0 {
		return errors.New("target amount must be greater than zero")
	}
	if item.Priority < 1 || item.Priority > 5 {
		return errors.New("priority must be between 1 and 5")
	}
	if item.Urgency < 1 || item.Urgency > 5 {
		return errors.New("urgency must be between 1 and 5")
	}

	query := `
		UPDATE wishlist_items
		SET title = $1, target_amount_e5 = $2, saved_amount_e5 = $3,
		    priority = $4, urgency = $5, item_type = $6, status = $7,
		    target_date = $8, notes = $9, updated_at = NOW()
		WHERE id = $10
	`

	var err error
	if Tx != nil {
		_, err = Tx.ExecContext(context.Background(), query,
			item.Title, item.TargetAmountE5, item.SavedAmountE5,
			item.Priority, item.Urgency, item.ItemType, item.Status,
			item.TargetDate, item.Notes, item.ID,
		)
	} else {
		_, err = r.db.ExecContext(context.Background(), query,
			item.Title, item.TargetAmountE5, item.SavedAmountE5,
			item.Priority, item.Urgency, item.ItemType, item.Status,
			item.TargetDate, item.Notes, item.ID,
		)
	}
	return err
}

func (r *pgWishlistRepo) DeleteWishlistItem(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("wishlist item ID is required")
	}

	query := `DELETE FROM wishlist_items WHERE id = $1`
	_, err := r.db.ExecContext(context.Background(), query, id)
	return err
}

func (r *pgWishlistRepo) CreateWishlistAllocation(alloc *core.WishlistAllocation, Tx *sql.Tx) (uuid.UUID, error) {
	if alloc.WishlistItemID == uuid.Nil {
		return uuid.Nil, errors.New("wishlist item ID is required")
	}
	if alloc.UserUUID == uuid.Nil {
		return uuid.Nil, errors.New("user UUID is required")
	}
	if alloc.AmountE5 <= 0 {
		return uuid.Nil, errors.New("allocation amount must be greater than zero")
	}
	if alloc.SourceType == "" {
		alloc.SourceType = "cycle_surplus"
	}
	if alloc.CycleDate.IsZero() {
		alloc.CycleDate = utils.NowUTC()
	}
	alloc.CreatedAt = utils.NowUTC()

	query := `
		INSERT INTO wishlist_allocations (
			wishlist_item_id, user_uuid, amount_e5, source_type, cycle_date, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	var err error
	if Tx != nil {
		err = Tx.QueryRowContext(context.Background(), query,
			alloc.WishlistItemID, alloc.UserUUID, alloc.AmountE5, alloc.SourceType, alloc.CycleDate, alloc.CreatedAt,
		).Scan(&id)
	} else {
		err = r.db.QueryRowContext(context.Background(), query,
			alloc.WishlistItemID, alloc.UserUUID, alloc.AmountE5, alloc.SourceType, alloc.CycleDate, alloc.CreatedAt,
		).Scan(&id)
	}
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *pgWishlistRepo) GetWishlistAllocationsByItemID(itemID uuid.UUID) ([]*core.WishlistAllocation, error) {
	if itemID == uuid.Nil {
		return nil, errors.New("wishlist item ID is required")
	}

	query := `
		SELECT id, wishlist_item_id, user_uuid, amount_e5, source_type, cycle_date, created_at
		FROM wishlist_allocations
		WHERE wishlist_item_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(context.Background(), query, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allocs []*core.WishlistAllocation
	for rows.Next() {
		alloc := &core.WishlistAllocation{}
		if err := rows.Scan(
			&alloc.ID, &alloc.WishlistItemID, &alloc.UserUUID, &alloc.AmountE5,
			&alloc.SourceType, &alloc.CycleDate, &alloc.CreatedAt,
		); err != nil {
			return nil, err
		}
		allocs = append(allocs, alloc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return allocs, nil
}

func (r *pgWishlistRepo) GetWishlistAllocationsByUserUUID(userUUID uuid.UUID) ([]*core.WishlistAllocation, error) {
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}

	query := `
		SELECT id, wishlist_item_id, user_uuid, amount_e5, source_type, cycle_date, created_at
		FROM wishlist_allocations
		WHERE user_uuid = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(context.Background(), query, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allocs []*core.WishlistAllocation
	for rows.Next() {
		alloc := &core.WishlistAllocation{}
		if err := rows.Scan(
			&alloc.ID, &alloc.WishlistItemID, &alloc.UserUUID, &alloc.AmountE5,
			&alloc.SourceType, &alloc.CycleDate, &alloc.CreatedAt,
		); err != nil {
			return nil, err
		}
		allocs = append(allocs, alloc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return allocs, nil
}
