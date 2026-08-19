package db

import (
	"database/sql"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
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

func (r *pgAllocationRepo) CreateAllocation(allocation *core.Allocation, Tx *sql.Tx) (uuid.UUID, error) {
	checkQuery := `
		SELECT id, envelope_id, allocated_amount_e5, created_at, updated_at, start_date, end_date
		FROM allocation
		WHERE envelope_id = $1
		  AND ($2::date IS NULL OR $3::date IS NULL OR (start_date <= $3 AND end_date >= $2))
		LIMIT 1
	`
	var checkRow *sql.Row
	if Tx != nil {
		checkRow = Tx.QueryRow(checkQuery, allocation.EnvelopeID, allocation.StartDate, allocation.EndDate)
	} else {
		checkRow = r.db.QueryRow(checkQuery, allocation.EnvelopeID, allocation.StartDate, allocation.EndDate)
	}

	var existingAllocation core.Allocation
	err := checkRow.Scan(
		&existingAllocation.ID,
		&existingAllocation.EnvelopeID,
		&existingAllocation.AllocatedAmountE5,
		&existingAllocation.CreatedAt,
		&existingAllocation.UpdatedAt,
		&existingAllocation.StartDate,
		&existingAllocation.EndDate,
	)

	if err == nil {
		*allocation = existingAllocation
		return allocation.ID, nil
	} else if err != sql.ErrNoRows {
		return uuid.Nil, err
	}

	query := `
		INSERT INTO allocation (envelope_id, allocated_amount_e5, created_at, updated_at, start_date, end_date)
		VALUES ($1, $2, COALESCE($3, NOW()), COALESCE($4, NOW()), $5, $6) RETURNING id
	`
	var row *sql.Row
	if Tx != nil {
		row = Tx.QueryRow(query, allocation.EnvelopeID, allocation.AllocatedAmountE5, allocation.CreatedAt, allocation.UpdatedAt, allocation.StartDate, allocation.EndDate)
	} else {
		row = r.db.QueryRow(query, allocation.EnvelopeID, allocation.AllocatedAmountE5, allocation.CreatedAt, allocation.UpdatedAt, allocation.StartDate, allocation.EndDate)
	}
	if err := row.Scan(&allocation.ID); err != nil {
		return uuid.Nil, err
	}
	return allocation.ID, nil
}

func (r *pgAllocationRepo) GetAllocationByID(id uuid.UUID) (*core.Allocation, error) {
	query := `
		SELECT id, envelope_id, allocated_amount_e5, created_at, updated_at, start_date, end_date
		FROM allocation
		WHERE id = $1
	`
	allocation := &core.Allocation{}
	err := r.db.QueryRow(query, id).Scan(&allocation.ID, &allocation.EnvelopeID, &allocation.AllocatedAmountE5, &allocation.CreatedAt, &allocation.UpdatedAt, &allocation.StartDate, &allocation.EndDate)
	return allocation, err
}

func (r *pgAllocationRepo) GetAllocationsByEnvelopeID(envelopeID uuid.UUID) ([]*core.Allocation, error) {
	query := `
		SELECT id, envelope_id, allocated_amount_e5, created_at, updated_at, start_date, end_date
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

func (r *pgAllocationRepo) GetActiveAllocationsByUserUUID(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*core.Allocation, error) {
	queryForEnv := `
		SELECT id, envelope_group_id, user_uuid, name, target_amount_e5, cadence, country_iso, is_system
		FROM envelope WHERE user_uuid = $1
	`

	var rowsForEnv *sql.Rows
	var err error
	if Tx != nil {
		rowsForEnv, err = Tx.Query(queryForEnv, userUUID)
	} else {
		rowsForEnv, err = r.db.Query(queryForEnv, userUUID)
	}
	if err != nil {
		return nil, err
	}
	defer rowsForEnv.Close()
	envelopeMap := make(map[uuid.UUID]core.Envelope)
	for rowsForEnv.Next() {
		envelope := &core.Envelope{}
		err := rowsForEnv.Scan(&envelope.ID, &envelope.EnvelopeGroupID, &envelope.UserUUID, &envelope.Name, &envelope.TargetAmountE5, &envelope.Cadence, &envelope.CountryISO, &envelope.IsSystem)
		if err != nil {
			return nil, err
		}
		envelopeMap[envelope.ID] = *envelope
	}
	if err := rowsForEnv.Err(); err != nil {
		return nil, err
	}

	query := `
		SELECT a.id, a.envelope_id, a.allocated_amount_e5, a.created_at, a.updated_at, a.start_date, a.end_date
		FROM allocation a
		JOIN envelope e ON a.envelope_id = e.id
		WHERE e.user_uuid = $1
		  AND $2::date BETWEEN a.start_date AND a.end_date
	`
	var rows *sql.Rows
	if Tx != nil {
		rows, err = Tx.Query(query, userUUID, targetDate)
	} else {
		rows, err = r.db.Query(query, userUUID, targetDate)
	}
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
		delete(envelopeMap, allocation.EnvelopeID)
	}

	for _, envelope := range envelopeMap {
		startDate, endDate, err := utils.GetCadenceStartAndEndTime(envelope.Cadence, targetDate)
		if err != nil {
			return nil, err
		}

		allocation := &core.Allocation{
			EnvelopeID:        envelope.ID,
			AllocatedAmountE5: envelope.TargetAmountE5,
			CreatedAt:         utils.NowUTC(),
			UpdatedAt:         utils.NowUTC(),
			StartDate:         &startDate,
			EndDate:           &endDate,
		}

		allocation.ID, err = r.CreateAllocation(allocation, Tx)
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
		SET envelope_id = $2, allocated_amount_e5 = $3, created_at = $4, updated_at = $5, start_date = $6, end_date = $7
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
