package db

import (
	"database/sql"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
)

type pgAllocationRepo struct {
	db *sql.DB
}

func NewPgAllocationRepo(db *sql.DB) core.AllocationRepository {
	return &pgAllocationRepo{
		db: db,
	}
}

func (r *pgAllocationRepo) CreateAllocation(allocation *core.Allocation) error {
	query := `
		INSERT INTO allocation (envelope_id, amount_e5, created_at, updated_at, start_date, end_date)
		VALUES ($1, $2, COALESCE($3, NOW()), COALESCE($4, NOW()), $5, $6)
	`
	_, err := r.db.Exec(query, allocation.EnvelopeID, allocation.AllocatedAmountE5, allocation.CreatedAt, allocation.UpdatedAt, allocation.StartDate, allocation.EndDate)
	return err
}

func (r *pgAllocationRepo) GetAllocationByID(id uuid.UUID) (*core.Allocation, error) {
	query := `
		SELECT id, envelope_id, amount_e5, created_at, updated_at, start_date, end_date
		FROM allocation
		WHERE id = $1
	`
	allocation := &core.Allocation{}
	err := r.db.QueryRow(query, id).Scan(&allocation.ID, &allocation.EnvelopeID, &allocation.AllocatedAmountE5, &allocation.CreatedAt, &allocation.UpdatedAt, &allocation.StartDate, &allocation.EndDate)
	return allocation, err
}

func (r *pgAllocationRepo) GetAllocationsByEnvelopeID(envelopeID uuid.UUID) ([]*core.Allocation, error) {
	query := `
		SELECT id, envelope_id, amount_e5, created_at, updated_at, start_date, end_date
		FROM allocation
		WHERE envelope_id = $1
	`
	rows, err := r.db.Query(query, envelopeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	allocations := []*core.Allocation{}
	for rows.Next() {
		allocation := &core.Allocation{}
		err := rows.Scan(&allocation.ID, &allocation.EnvelopeID, &allocation.AllocatedAmountE5, &allocation.CreatedAt, &allocation.UpdatedAt, &allocation.StartDate, &allocation.EndDate)
		if err != nil {
			return nil, err
		}
		allocations = append(allocations, allocation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return allocations, nil
}

func (r *pgAllocationRepo) UpdateAllocation(allocation *core.Allocation) error {
	query := `
		UPDATE allocation
		SET envelope_id = $2, amount_e5 = $3, created_at = $4, updated_at = $5, start_date = $6, end_date = $7
		WHERE id = $1
	`
	_, err := r.db.Exec(query, allocation.ID, allocation.EnvelopeID, allocation.AllocatedAmountE5, allocation.CreatedAt, allocation.UpdatedAt, allocation.StartDate, allocation.EndDate)
	return err
}

func (r *pgAllocationRepo) DeleteAllocation(id uuid.UUID) error {
	query := `
		DELETE FROM allocation
		WHERE id = $1
	`
	_, err := r.db.Exec(query, id)
	return err
}
