package db

import (
	"database/sql"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
)

type pgEnvelopeRepo struct {
	db *sql.DB
}

func NewPgEnvelopeRepo(db *sql.DB) core.EnvelopeRepository {
	return &pgEnvelopeRepo{
		db: db,
	}
}

func (r *pgEnvelopeRepo) CreateEnvelope(envelope *core.Envelope, Tx *sql.Tx) (uuid.UUID, error) {
	query := `
		INSERT INTO envelope (
			user_uuid,
			name,
			envelope_group_id,
			target_amount_e5,
			cadence,
			country_iso,
			created_at,
			updated_at,
			is_system
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8,$9) RETURNING id
	`
	var row *sql.Row
	if Tx != nil {
		row = Tx.QueryRow(
			query,
			envelope.UserUUID,
			envelope.Name,
			envelope.EnvelopeGroupID,
			envelope.TargetAmountE5,
			envelope.Cadence,
			envelope.CountryISO,
			envelope.CreatedAt,
			envelope.UpdatedAt,
			envelope.IsSystem,
		)
	} else {
		row = r.db.QueryRow(
			query,
			envelope.UserUUID,
			envelope.Name,
			envelope.EnvelopeGroupID,
			envelope.TargetAmountE5,
			envelope.Cadence,
			envelope.CountryISO,
			envelope.CreatedAt,
			envelope.UpdatedAt,
			envelope.IsSystem,
		)
	}
	if err := row.Scan(&envelope.ID); err != nil {
		return uuid.Nil, err
	}
	return envelope.ID, nil
}

func (r *pgEnvelopeRepo) GetEnvelopeByID(id uuid.UUID) (*core.Envelope, error) {
	query := `
		SELECT id, user_uuid,name, envelope_group_id, target_amount_e5, cadence, country_iso, created_at, updated_at, is_system
		FROM envelope
		WHERE id = $1
	`
	envelope := &core.Envelope{}
	err := r.db.QueryRow(query, id).Scan(
		&envelope.ID,
		&envelope.UserUUID,
		&envelope.Name,
		&envelope.EnvelopeGroupID,
		&envelope.TargetAmountE5,
		&envelope.Cadence,
		&envelope.CountryISO,
		&envelope.CreatedAt,
		&envelope.UpdatedAt,
		&envelope.IsSystem,
	)
	if err != nil {
		return nil, err
	}
	return envelope, nil
}

func (r *pgEnvelopeRepo) GetEnvelopesByUserUUID(userUUID uuid.UUID) ([]*core.Envelope, error) {
	query := `
		SELECT id, user_uuid,name, envelope_group_id, target_amount_e5, cadence, country_iso, created_at, updated_at, is_system
		FROM envelope
		WHERE user_uuid = $1
	`
	rows, err := r.db.Query(query, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	envelopes := []*core.Envelope{}
	for rows.Next() {
		envelope := &core.Envelope{}
		err := rows.Scan(
			&envelope.ID,
			&envelope.UserUUID,
			&envelope.Name,
			&envelope.EnvelopeGroupID,
			&envelope.TargetAmountE5,
			&envelope.Cadence,
			&envelope.CountryISO,
			&envelope.CreatedAt,
			&envelope.UpdatedAt,
			&envelope.IsSystem,
		)
		if err != nil {
			return nil, err
		}
		envelopes = append(envelopes, envelope)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return envelopes, nil
}

func (r *pgEnvelopeRepo) UpdateEnvelope(envelope *core.Envelope) error {
	query := `
		UPDATE envelope
		SET 
			envelope_group_id = $2,
			target_amount_e5 = $3,
			cadence = $4,
			country_iso = $5,
			updated_at = $6,
			is_system = $7,
			name = $8
		WHERE id = $1 AND user_uuid = $9
	`
	_, err := r.db.Exec(
		query,
		envelope.ID,
		envelope.EnvelopeGroupID,
		envelope.TargetAmountE5,
		envelope.Cadence,
		envelope.CountryISO,
		envelope.UpdatedAt,
		envelope.IsSystem,
		envelope.Name,
		envelope.UserUUID,
	)
	return err
}

func (r *pgEnvelopeRepo) DeleteEnvelope(id uuid.UUID) error {
	query := `
		DELETE FROM envelope
		WHERE id = $1
	`
	_, err := r.db.Exec(query, id)
	return err
}
