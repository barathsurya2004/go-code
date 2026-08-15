package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
)

type pgShortcutIntentRepo struct {
	db *sql.DB
}

func NewPgShortcutIntentRepo(db *sql.DB) core.ShortcutIntentRepository {
	return &pgShortcutIntentRepo{
		db: db,
	}
}

func (r *pgShortcutIntentRepo) CreateShortcutIntent(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error) {
	query := `
		INSERT INTO shortcut_intent (user_id, envelope_id, latitude, longitude, status, created_at, transaction_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`

	// validation checks
	if shortcutIntent.UserID == uuid.Nil {
		return uuid.Nil, errors.New("user UUID is required")
	}
	if shortcutIntent.Status == "" {
		return uuid.Nil, errors.New("shortcut intent status cannot be empty")
	}
	if shortcutIntent.CreatedAt.IsZero() {
		shortcutIntent.CreatedAt = time.Now()
	}
	var shortcutIntentID uuid.UUID
	var row *sql.Row
	if Tx != nil {
		row = Tx.QueryRow(query,
			shortcutIntent.UserID,
			shortcutIntent.EnvelopeID,
			shortcutIntent.Latitude,
			shortcutIntent.Longitude,
			shortcutIntent.Status,
			shortcutIntent.CreatedAt,
			shortcutIntent.TransactionID,
		)
	} else {
		row = r.db.QueryRowContext(context.Background(), query,
			shortcutIntent.UserID,
			shortcutIntent.EnvelopeID,
			shortcutIntent.Latitude,
			shortcutIntent.Longitude,
			shortcutIntent.Status,
			shortcutIntent.CreatedAt,
			shortcutIntent.TransactionID,
		)
	}
	if err := row.Scan(&shortcutIntentID); err != nil {
		return uuid.Nil, err
	}
	shortcutIntent.ID = shortcutIntentID
	return shortcutIntentID, nil
}

func (r *pgShortcutIntentRepo) GetShortcutIntentByID(id uuid.UUID) (*core.ShortcutIntent, error) {
	query := `
		SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id
		FROM shortcut_intent
		WHERE id = $1
	`

	// validation checks
	if id == uuid.Nil {
		return nil, errors.New("shortcut intent UUID is required")
	}

	shortcutIntent := &core.ShortcutIntent{}
	err := r.db.QueryRowContext(context.Background(), query, id).Scan(
		&shortcutIntent.ID,
		&shortcutIntent.UserID,
		&shortcutIntent.EnvelopeID,
		&shortcutIntent.Latitude,
		&shortcutIntent.Longitude,
		&shortcutIntent.Status,
		&shortcutIntent.CreatedAt,
		&shortcutIntent.TransactionID,
	)
	if err != nil {
		return nil, err
	}
	return shortcutIntent, nil
}

func (r *pgShortcutIntentRepo) GetShortcutIntentsByUserUUID(userUUID uuid.UUID) ([]*core.ShortcutIntent, error) {
	query := `
		SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id
		FROM shortcut_intent
		WHERE user_id = $1
	`

	// validation checks
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}

	rows, err := r.db.QueryContext(context.Background(), query, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shortcutIntents []*core.ShortcutIntent
	for rows.Next() {
		shortcutIntent := &core.ShortcutIntent{}
		if err := rows.Scan(
			&shortcutIntent.ID,
			&shortcutIntent.UserID,
			&shortcutIntent.EnvelopeID,
			&shortcutIntent.Latitude,
			&shortcutIntent.Longitude,
			&shortcutIntent.Status,
			&shortcutIntent.CreatedAt,
			&shortcutIntent.TransactionID,
		); err != nil {
			return nil, err
		}
		shortcutIntents = append(shortcutIntents, shortcutIntent)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return shortcutIntents, nil
}

func (r *pgShortcutIntentRepo) UpdateShortcutIntent(shortcutIntent *core.ShortcutIntent, Tx *sql.Tx) error {
	query := `
		UPDATE shortcut_intent
		SET envelope_id = $1, latitude = $2, longitude = $3, status = $4, created_at = $5, transaction_id = $6
		WHERE id = $7
	`

	// validation checks
	if shortcutIntent.ID == uuid.Nil {
		return errors.New("shortcut intent UUID is required")
	}
	if shortcutIntent.Status == "" {
		return errors.New("shortcut intent status cannot be empty")
	}

	if Tx != nil {
		_, err := Tx.Exec(query,
			shortcutIntent.EnvelopeID,
			shortcutIntent.Latitude,
			shortcutIntent.Longitude,
			shortcutIntent.Status,
			shortcutIntent.CreatedAt,
			shortcutIntent.TransactionID,
			shortcutIntent.ID,
		)
		return err
	}

	_, err := r.db.ExecContext(context.Background(), query,
		shortcutIntent.EnvelopeID,
		shortcutIntent.Latitude,
		shortcutIntent.Longitude,
		shortcutIntent.Status,
		shortcutIntent.CreatedAt,
		shortcutIntent.TransactionID,
		shortcutIntent.ID,
	)
	return err
}

func (r *pgShortcutIntentRepo) DeleteShortcutIntent(id uuid.UUID) error {
	query := `
		DELETE FROM shortcut_intent
		WHERE id = $1
	`

	// validation checks
	if id == uuid.Nil {
		return errors.New("shortcut intent UUID is required")
	}

	_, err := r.db.ExecContext(context.Background(), query, id)
	return err
}

func (r *pgShortcutIntentRepo) GetPendingRecentShortcutIntent(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*core.ShortcutIntent, error) {
	query := `
		SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id
		FROM shortcut_intent
		WHERE user_id = $1 AND status = $2 AND created_at BETWEEN $3 AND $4
	`

	// validation checks
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}
	if time_lowerbound.IsZero() || time_upperbound.IsZero() {
		return nil, errors.New("time lowerbound and upperbound are required")
	}
	if time_lowerbound.After(time_upperbound) {
		return nil, errors.New("time lowerbound cannot be after time upperbound")
	}

	shortcutIntent := &core.ShortcutIntent{}
	var err error
	if Tx != nil {
		err = Tx.QueryRow(query, userUUID, core.StatusPending, time_lowerbound, time_upperbound).Scan(
			&shortcutIntent.ID,
			&shortcutIntent.UserID,
			&shortcutIntent.EnvelopeID,
			&shortcutIntent.Latitude,
			&shortcutIntent.Longitude,
			&shortcutIntent.Status,
			&shortcutIntent.CreatedAt,
			&shortcutIntent.TransactionID,
		)
	} else {
		err = r.db.QueryRowContext(context.Background(), query, userUUID, core.StatusPending, time_lowerbound, time_upperbound).Scan(
			&shortcutIntent.ID,
			&shortcutIntent.UserID,
			&shortcutIntent.EnvelopeID,
			&shortcutIntent.Latitude,
			&shortcutIntent.Longitude,
			&shortcutIntent.Status,
			&shortcutIntent.CreatedAt,
			&shortcutIntent.TransactionID,
		)
	}
	if err != nil {
		return nil, err
	}
	return shortcutIntent, nil
}
