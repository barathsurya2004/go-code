package db

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
)

type EnvelopeGroupRepository struct {
	db *sql.DB
}

func NewEnvelopeGroupRepository(db *sql.DB) *EnvelopeGroupRepository {
	return &EnvelopeGroupRepository{db: db}
}

func (r *EnvelopeGroupRepository) CreateEnvelopeGroup(envelopeGroup *core.EnvelopeGroup) error {
	query := `
		INSERT INTO envelope_groups (user_uuid, name, is_system)
		VALUES ($1, $2, $3)
	`

	if strings.TrimSpace(envelopeGroup.Name) == "" {
		return errors.New("envelope group name is required")
	}

	_, err := r.db.Exec(query, envelopeGroup.UserUUID, envelopeGroup.Name, envelopeGroup.IsSystem)
	return err
}

func (r *EnvelopeGroupRepository) GetEnvelopeGroupByID(id uuid.UUID) (*core.EnvelopeGroup, error) {
	query := `
		SELECT id, user_uuid, name, is_system
		FROM envelope_groups
		WHERE id = $1
	`

	if err := uuid.Validate(id.String()); err != nil {
		return nil, errors.New("envelope group ID is invalid")
	}

	envelopeGroup := &core.EnvelopeGroup{}
	err := r.db.QueryRow(query, id).Scan(
		&envelopeGroup.ID,
		&envelopeGroup.UserUUID,
		&envelopeGroup.Name,
		&envelopeGroup.IsSystem,
		&envelopeGroup.CreatedAt,
		&envelopeGroup.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return envelopeGroup, nil
}

func (r *EnvelopeGroupRepository) GetEnvelopeGroupsByUserUUID(userUUID string) ([]*core.EnvelopeGroup, error) {
	query := `
		SELECT id, user_uuid, name, is_system
		FROM envelope_groups
		WHERE user_uuid = $1
	`

	if err := uuid.Validate(userUUID); err != nil {
		return nil, errors.New("user UUID is invalid")
	}

	rows, err := r.db.Query(query, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var envelopeGroups []*core.EnvelopeGroup
	for rows.Next() {
		envelopeGroup := &core.EnvelopeGroup{}
		if err := rows.Scan(
			&envelopeGroup.ID,
			&envelopeGroup.UserUUID,
			&envelopeGroup.Name,
			&envelopeGroup.IsSystem,
			&envelopeGroup.CreatedAt,
			&envelopeGroup.UpdatedAt,
		); err != nil {
			return nil, err
		}
		envelopeGroups = append(envelopeGroups, envelopeGroup)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return envelopeGroups, nil
}

func (r *EnvelopeGroupRepository) UpdateEnvelopeGroup(envelopeGroup *core.EnvelopeGroup) error {
	query := `
		UPDATE envelope_groups
		SET name = $1, is_system = $2
		WHERE id = $3
	`

	if strings.TrimSpace(envelopeGroup.Name) == "" {
		return errors.New("envelope group name is required")
	}

	if err := uuid.Validate(envelopeGroup.ID.String()); err != nil {
		return errors.New("envelope group ID is invalid")
	}

	_, err := r.db.Exec(query, envelopeGroup.Name, envelopeGroup.IsSystem, envelopeGroup.ID)
	return err
}

func (r *EnvelopeGroupRepository) DeleteEnvelopeGroup(id uuid.UUID) error {
	query := `
		DELETE FROM envelope_groups
		WHERE id = $1
	`

	if err := uuid.Validate(id.String()); err != nil {
		return errors.New("envelope group ID is invalid")
	}

	_, err := r.db.Exec(query, id)
	return err
}
