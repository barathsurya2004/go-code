package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
)

func TestPgAllocationRepo_CreateAllocation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgAllocationRepo(db)

	t.Run("Exec Error", func(t *testing.T) {
		now := time.Now()
		alloc := &core.Allocation{
			EnvelopeID:        uuid.New(),
			AllocatedAmountE5: 150000,
			SpentAmountE5:     0,
			CreatedAt:         now,
			UpdatedAt:         now,
			StartDate:         &now,
			EndDate:           &now,
		}

		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date FROM allocation WHERE envelope_id = \\$1").
			WithArgs(alloc.EnvelopeID, alloc.StartDate, alloc.EndDate).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("INSERT INTO allocation").
			WithArgs(alloc.EnvelopeID, alloc.AllocatedAmountE5, alloc.SpentAmountE5, alloc.CreatedAt, alloc.UpdatedAt, alloc.StartDate, alloc.EndDate).
			WillReturnError(errors.New("db error"))

		_, err := repo.CreateAllocation(alloc, tx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		genID := uuid.New()
		alloc := &core.Allocation{
			EnvelopeID:        uuid.New(),
			AllocatedAmountE5: 150000,
			SpentAmountE5:     0,
			CreatedAt:         now,
			UpdatedAt:         now,
			StartDate:         &now,
			EndDate:           &now,
		}

		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date FROM allocation WHERE envelope_id = \\$1").
			WithArgs(alloc.EnvelopeID, alloc.StartDate, alloc.EndDate).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("INSERT INTO allocation").
			WithArgs(alloc.EnvelopeID, alloc.AllocatedAmountE5, alloc.SpentAmountE5, alloc.CreatedAt, alloc.UpdatedAt, alloc.StartDate, alloc.EndDate).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(genID))

		id, err := repo.CreateAllocation(alloc, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != genID {
			t.Errorf("expected ID %v, got %v", genID, id)
		}
	})

	t.Run("Already Present", func(t *testing.T) {
		now := time.Now()
		existingID := uuid.New()
		alloc := &core.Allocation{
			EnvelopeID:        uuid.New(),
			AllocatedAmountE5: 150000,
			SpentAmountE5:     0,
			CreatedAt:         now,
			UpdatedAt:         now,
			StartDate:         &now,
			EndDate:           &now,
		}

		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date FROM allocation WHERE envelope_id = \\$1").
			WithArgs(alloc.EnvelopeID, alloc.StartDate, alloc.EndDate).
			WillReturnRows(sqlmock.NewRows([]string{"id", "envelope_id", "allocated_amount_e5", "spent_amount_e5", "created_at", "updated_at", "start_date", "end_date"}).
				AddRow(existingID, alloc.EnvelopeID, alloc.AllocatedAmountE5, 0, now, now, now, now))

		id, err := repo.CreateAllocation(alloc, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != existingID {
			t.Errorf("expected ID %v, got %v", existingID, id)
		}
	})
}

func TestPgAllocationRepo_GetAllocationByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgAllocationRepo(db)
	allocID := uuid.New()

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date").
			WithArgs(allocID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetAllocationByID(allocID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		envelopeID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "envelope_id", "allocated_amount_e5", "spent_amount_e5", "created_at", "updated_at", "start_date", "end_date"}).
			AddRow(allocID, envelopeID, 250000.0, 50000, now, now, now, now)

		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date").
			WithArgs(allocID).
			WillReturnRows(rows)

		result, err := repo.GetAllocationByID(allocID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.ID != allocID {
			t.Errorf("expected allocation ID %v, got %v", allocID, result.ID)
		}
		if result.EnvelopeID != envelopeID {
			t.Errorf("expected envelope ID %v, got %v", envelopeID, result.EnvelopeID)
		}
		if result.AllocatedAmountE5 != 250000.0 {
			t.Errorf("expected amount 250000.0, got %v", result.AllocatedAmountE5)
		}
		if result.SpentAmountE5 != 50000 {
			t.Errorf("expected spent amount 50000, got %v", result.SpentAmountE5)
		}
	})
}

func TestPgAllocationRepo_GetAllocationsByEnvelopeID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgAllocationRepo(db)
	envelopeID := uuid.New()

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date").
			WithArgs(envelopeID).
			WillReturnError(errors.New("query error"))

		_, err := repo.GetAllocationsByEnvelopeID(envelopeID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Scan Error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id"}).AddRow("invalid")
		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date").
			WithArgs(envelopeID).
			WillReturnRows(rows)

		_, err := repo.GetAllocationsByEnvelopeID(envelopeID)
		if err == nil {
			t.Error("expected scan error, got nil")
		}
	})

	t.Run("Row Iteration Error", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "envelope_id", "allocated_amount_e5", "spent_amount_e5", "created_at", "updated_at", "start_date", "end_date"}).
			AddRow(uuid.New(), envelopeID, 100000.0, 0, now, now, now, now).
			RowError(0, errors.New("iteration error"))

		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date").
			WithArgs(envelopeID).
			WillReturnRows(rows)

		_, err := repo.GetAllocationsByEnvelopeID(envelopeID)
		if err == nil {
			t.Error("expected rows.Err() error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		alloc1ID, alloc2ID := uuid.New(), uuid.New()

		rows := sqlmock.NewRows([]string{"id", "envelope_id", "allocated_amount_e5", "spent_amount_e5", "created_at", "updated_at", "start_date", "end_date"}).
			AddRow(alloc1ID, envelopeID, 100000.0, 10000, now, now, now, now).
			AddRow(alloc2ID, envelopeID, 200000.0, 20000, now, now, now, now)

		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date").
			WithArgs(envelopeID).
			WillReturnRows(rows)

		results, err := repo.GetAllocationsByEnvelopeID(envelopeID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 allocations, got %d", len(results))
		}
	})
}

func TestPgAllocationRepo_GetActiveAllocationsByUserUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgAllocationRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	now := time.Now()

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, envelope_group_id, user_uuid, name, target_amount_e5, cadence, country_iso, is_system FROM envelope").
			WithArgs(userUUID).
			WillReturnError(errors.New("query error"))

		_, err := repo.GetActiveAllocationsByUserUUID(userUUID, now, nil)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Scan Error", func(t *testing.T) {
		envRows := sqlmock.NewRows([]string{"id", "envelope_group_id", "user_uuid", "name", "target_amount_e5", "cadence", "country_iso", "is_system"}).
			AddRow(uuid.New(), uuid.New(), userUUID, "Food", 100.0, "monthly", "US", false)

		mock.ExpectQuery("SELECT id, envelope_group_id, user_uuid, name, target_amount_e5, cadence, country_iso, is_system FROM envelope").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		rows := sqlmock.NewRows([]string{"id"}).AddRow("invalid")
		mock.ExpectQuery("SELECT a.id, a.envelope_id, a.allocated_amount_e5, a.spent_amount_e5, a.created_at, a.updated_at, a.start_date, a.end_date").
			WithArgs(userUUID, now).
			WillReturnRows(rows)

		_, err := repo.GetActiveAllocationsByUserUUID(userUUID, now, nil)
		if err == nil {
			t.Error("expected scan error, got nil")
		}
	})

	t.Run("Row Iteration Error", func(t *testing.T) {
		envRows := sqlmock.NewRows([]string{"id", "envelope_group_id", "user_uuid", "name", "target_amount_e5", "cadence", "country_iso", "is_system"}).
			AddRow(uuid.New(), uuid.New(), userUUID, "Food", 100.0, "monthly", "US", false)

		mock.ExpectQuery("SELECT id, envelope_group_id, user_uuid, name, target_amount_e5, cadence, country_iso, is_system FROM envelope").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		rows := sqlmock.NewRows([]string{"id", "envelope_id", "allocated_amount_e5", "spent_amount_e5", "created_at", "updated_at", "start_date", "end_date"}).
			AddRow(uuid.New(), uuid.New(), 100000.0, 0, now, now, now, now).
			RowError(0, errors.New("iteration error"))

		mock.ExpectQuery("SELECT a.id, a.envelope_id, a.allocated_amount_e5, a.spent_amount_e5, a.created_at, a.updated_at, a.start_date, a.end_date").
			WithArgs(userUUID, now).
			WillReturnRows(rows)

		_, err := repo.GetActiveAllocationsByUserUUID(userUUID, now, nil)
		if err == nil {
			t.Error("expected rows.Err() error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		allocID := uuid.New()
		envelopeID := uuid.New()

		envRows := sqlmock.NewRows([]string{"id", "envelope_group_id", "user_uuid", "name", "target_amount_e5", "cadence", "country_iso", "is_system"}).
			AddRow(envelopeID, uuid.New(), userUUID, "Food", 100.0, "monthly", "US", false)

		mock.ExpectQuery("SELECT id, envelope_group_id, user_uuid, name, target_amount_e5, cadence, country_iso, is_system FROM envelope").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		rows := sqlmock.NewRows([]string{"id", "envelope_id", "allocated_amount_e5", "spent_amount_e5", "created_at", "updated_at", "start_date", "end_date"}).
			AddRow(allocID, envelopeID, 150000.0, 20000, now, now, now, now)

		mock.ExpectQuery("SELECT a.id, a.envelope_id, a.allocated_amount_e5, a.spent_amount_e5, a.created_at, a.updated_at, a.start_date, a.end_date").
			WithArgs(userUUID, now).
			WillReturnRows(rows)

		results, err := repo.GetActiveAllocationsByUserUUID(userUUID, now, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 allocation, got %d", len(results))
		}
		if results[0].SpentAmountE5 != 20000 {
			t.Errorf("expected spent amount 20000, got %d", results[0].SpentAmountE5)
		}
	})

	t.Run("Success With Tx", func(t *testing.T) {
		allocID := uuid.New()
		envelopeID := uuid.New()

		mock.ExpectBegin()
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		envRows := sqlmock.NewRows([]string{"id", "envelope_group_id", "user_uuid", "name", "target_amount_e5", "cadence", "country_iso", "is_system"}).
			AddRow(envelopeID, uuid.New(), userUUID, "Food", 100.0, "monthly", "US", false)

		mock.ExpectQuery("SELECT id, envelope_group_id, user_uuid, name, target_amount_e5, cadence, country_iso, is_system FROM envelope").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		rows := sqlmock.NewRows([]string{"id", "envelope_id", "allocated_amount_e5", "spent_amount_e5", "created_at", "updated_at", "start_date", "end_date"}).
			AddRow(allocID, envelopeID, 150000.0, 0, now, now, now, now)

		mock.ExpectQuery("SELECT a.id, a.envelope_id, a.allocated_amount_e5, a.spent_amount_e5, a.created_at, a.updated_at, a.start_date, a.end_date").
			WithArgs(userUUID, now).
			WillReturnRows(rows)

		results, err := repo.GetActiveAllocationsByUserUUID(userUUID, now, tx)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 allocation, got %d", len(results))
		}
	})

	t.Run("Auto Create Allocation Success", func(t *testing.T) {
		envelopeID := uuid.New()
		allocID := uuid.New()

		envRows := sqlmock.NewRows([]string{"id", "envelope_group_id", "user_uuid", "name", "target_amount_e5", "cadence", "country_iso", "is_system"}).
			AddRow(envelopeID, uuid.New(), userUUID, "Food", 100.0, "monthly", "US", false)

		mock.ExpectQuery("SELECT id, envelope_group_id, user_uuid, name, target_amount_e5, cadence, country_iso, is_system FROM envelope").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		rows := sqlmock.NewRows([]string{"id", "envelope_id", "allocated_amount_e5", "spent_amount_e5", "created_at", "updated_at", "start_date", "end_date"})

		mock.ExpectQuery("SELECT a.id, a.envelope_id, a.allocated_amount_e5, a.spent_amount_e5, a.created_at, a.updated_at, a.start_date, a.end_date").
			WithArgs(userUUID, now).
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date FROM allocation WHERE envelope_id = \\$1").
			WithArgs(envelopeID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery("INSERT INTO allocation").
			WithArgs(envelopeID, 100.0, int64(0), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(allocID))

		results, err := repo.GetActiveAllocationsByUserUUID(userUUID, now, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 allocation created, got %d", len(results))
		}
	})

	t.Run("Auto Create Allocation Cadence Error", func(t *testing.T) {
		envelopeID := uuid.New()

		envRows := sqlmock.NewRows([]string{"id", "envelope_group_id", "user_uuid", "name", "target_amount_e5", "cadence", "country_iso", "is_system"}).
			AddRow(envelopeID, uuid.New(), userUUID, "Food", 100.0, "invalid_cadence", "US", false)

		mock.ExpectQuery("SELECT id, envelope_group_id, user_uuid, name, target_amount_e5, cadence, country_iso, is_system FROM envelope").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		rows := sqlmock.NewRows([]string{"id", "envelope_id", "allocated_amount_e5", "spent_amount_e5", "created_at", "updated_at", "start_date", "end_date"})

		mock.ExpectQuery("SELECT a.id, a.envelope_id, a.allocated_amount_e5, a.spent_amount_e5, a.created_at, a.updated_at, a.start_date, a.end_date").
			WithArgs(userUUID, now).
			WillReturnRows(rows)

		_, err := repo.GetActiveAllocationsByUserUUID(userUUID, now, nil)
		if err == nil {
			t.Error("expected error for invalid cadence, got nil")
		}
	})

	t.Run("Auto Create Allocation DB Error", func(t *testing.T) {
		envelopeID := uuid.New()

		envRows := sqlmock.NewRows([]string{"id", "envelope_group_id", "user_uuid", "name", "target_amount_e5", "cadence", "country_iso", "is_system"}).
			AddRow(envelopeID, uuid.New(), userUUID, "Food", 100.0, "monthly", "US", false)

		mock.ExpectQuery("SELECT id, envelope_group_id, user_uuid, name, target_amount_e5, cadence, country_iso, is_system FROM envelope").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		rows := sqlmock.NewRows([]string{"id", "envelope_id", "allocated_amount_e5", "spent_amount_e5", "created_at", "updated_at", "start_date", "end_date"})

		mock.ExpectQuery("SELECT a.id, a.envelope_id, a.allocated_amount_e5, a.spent_amount_e5, a.created_at, a.updated_at, a.start_date, a.end_date").
			WithArgs(userUUID, now).
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date FROM allocation WHERE envelope_id = \\$1").
			WithArgs(envelopeID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery("INSERT INTO allocation").
			WithArgs(envelopeID, 100.0, int64(0), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("creation failed"))

		_, err := repo.GetActiveAllocationsByUserUUID(userUUID, now, nil)
		if err == nil {
			t.Error("expected error for CreateAllocation failure, got nil")
		}
	})

	t.Run("Allocations Query Error", func(t *testing.T) {
		envelopeID := uuid.New()

		envRows := sqlmock.NewRows([]string{"id", "envelope_group_id", "user_uuid", "name", "target_amount_e5", "cadence", "country_iso", "is_system"}).
			AddRow(envelopeID, uuid.New(), userUUID, "Food", 100.0, "monthly", "US", false)

		mock.ExpectQuery("SELECT id, envelope_group_id, user_uuid, name, target_amount_e5, cadence, country_iso, is_system FROM envelope").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		mock.ExpectQuery("SELECT a.id, a.envelope_id, a.allocated_amount_e5, a.spent_amount_e5, a.created_at, a.updated_at, a.start_date, a.end_date").
			WithArgs(userUUID, now).
			WillReturnError(errors.New("allocations query error"))

		_, err := repo.GetActiveAllocationsByUserUUID(userUUID, now, nil)
		if err == nil {
			t.Error("expected error for allocations query error, got nil")
		}
	})
}

func TestPgAllocationRepo_UpdateAllocation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgAllocationRepo(db)

	t.Run("Exec Error", func(t *testing.T) {
		now := time.Now()
		alloc := &core.Allocation{
			ID:                uuid.New(),
			EnvelopeID:        uuid.New(),
			AllocatedAmountE5: 300000,
			SpentAmountE5:     10000,
			CreatedAt:         now,
			UpdatedAt:         now,
			StartDate:         &now,
			EndDate:           &now,
		}

		mock.ExpectExec("UPDATE allocation").
			WithArgs(alloc.ID, alloc.EnvelopeID, alloc.AllocatedAmountE5, alloc.SpentAmountE5, alloc.CreatedAt, alloc.UpdatedAt, alloc.StartDate, alloc.EndDate).
			WillReturnError(errors.New("update error"))

		err := repo.UpdateAllocation(alloc)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		alloc := &core.Allocation{
			ID:                uuid.New(),
			EnvelopeID:        uuid.New(),
			AllocatedAmountE5: 300000,
			SpentAmountE5:     10000,
			CreatedAt:         now,
			UpdatedAt:         now,
			StartDate:         &now,
			EndDate:           &now,
		}

		mock.ExpectExec("UPDATE allocation").
			WithArgs(alloc.ID, alloc.EnvelopeID, alloc.AllocatedAmountE5, alloc.SpentAmountE5, alloc.CreatedAt, alloc.UpdatedAt, alloc.StartDate, alloc.EndDate).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateAllocation(alloc)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestPgAllocationRepo_DeleteAllocation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgAllocationRepo(db)
	allocID := uuid.New()

	t.Run("Exec Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM allocation").
			WithArgs(allocID).
			WillReturnError(errors.New("delete error"))

		err := repo.DeleteAllocation(allocID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM allocation").
			WithArgs(allocID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.DeleteAllocation(allocID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestPgAllocationRepo_UpdateSpentAmount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgAllocationRepo(db)
	envelopeID := uuid.New()
	targetDate := time.Now()

	t.Run("Update Existing Success", func(t *testing.T) {
		allocID := uuid.New()
		mock.ExpectQuery("UPDATE allocation SET spent_amount_e5 = GREATEST").
			WithArgs(int64(50000), envelopeID, targetDate).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(allocID))

		err := repo.UpdateSpentAmount(envelopeID, targetDate, 50000, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Update Existing Not Found - Create New Success", func(t *testing.T) {
		newAllocID := uuid.New()
		// 1. UPDATE returns ErrNoRows
		mock.ExpectQuery("UPDATE allocation SET spent_amount_e5 = GREATEST").
			WithArgs(int64(25000), envelopeID, targetDate).
			WillReturnError(sql.ErrNoRows)

		// 2. Select envelope
		mock.ExpectQuery("SELECT cadence, target_amount_e5 FROM envelope WHERE id = \\$1").
			WithArgs(envelopeID).
			WillReturnRows(sqlmock.NewRows([]string{"cadence", "target_amount_e5"}).AddRow("monthly", 100000.0))

		// 3. Check existing allocation in CreateAllocation
		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date FROM allocation WHERE envelope_id = \\$1").
			WithArgs(envelopeID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		// 4. Insert allocation
		mock.ExpectQuery("INSERT INTO allocation").
			WithArgs(envelopeID, 100000.0, int64(25000), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(newAllocID))

		err := repo.UpdateSpentAmount(envelopeID, targetDate, 25000, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Update DB Error", func(t *testing.T) {
		mock.ExpectQuery("UPDATE allocation SET spent_amount_e5 = GREATEST").
			WithArgs(int64(10000), envelopeID, targetDate).
			WillReturnError(errors.New("db error"))

		err := repo.UpdateSpentAmount(envelopeID, targetDate, 10000, nil)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Validation Errors", func(t *testing.T) {
		if err := repo.UpdateSpentAmount(uuid.Nil, targetDate, 100, nil); err == nil {
			t.Error("expected error for nil envelope ID")
		}
		if err := repo.UpdateSpentAmount(envelopeID, time.Time{}, 100, nil); err == nil {
			t.Error("expected error for zero target date")
		}
		if _, err := repo.CreateAllocation(nil, nil); err == nil {
			t.Error("expected error for nil allocation")
		}
		if _, err := repo.CreateAllocation(&core.Allocation{}, nil); err == nil {
			t.Error("expected error for empty allocation envelope ID")
		}
		if err := repo.UpdateAllocation(nil); err == nil {
			t.Error("expected error for nil allocation in update")
		}
		if err := repo.UpdateAllocation(&core.Allocation{}); err == nil {
			t.Error("expected error for empty envelope ID in update")
		}
	})

	t.Run("Fallback GetEnvelope Error", func(t *testing.T) {
		mock.ExpectQuery("UPDATE allocation SET spent_amount_e5 = GREATEST").
			WithArgs(int64(25000), envelopeID, targetDate).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery("SELECT cadence, target_amount_e5 FROM envelope WHERE id = \\$1").
			WithArgs(envelopeID).
			WillReturnError(errors.New("envelope fetch error"))

		err := repo.UpdateSpentAmount(envelopeID, targetDate, 25000, nil)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Fallback CreateAllocation Error", func(t *testing.T) {
		mock.ExpectQuery("UPDATE allocation SET spent_amount_e5 = GREATEST").
			WithArgs(int64(25000), envelopeID, targetDate).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery("SELECT cadence, target_amount_e5 FROM envelope WHERE id = \\$1").
			WithArgs(envelopeID).
			WillReturnRows(sqlmock.NewRows([]string{"cadence", "target_amount_e5"}).AddRow("monthly", 100000.0))

		mock.ExpectQuery("SELECT id, envelope_id, allocated_amount_e5, spent_amount_e5, created_at, updated_at, start_date, end_date FROM allocation WHERE envelope_id = \\$1").
			WithArgs(envelopeID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery("INSERT INTO allocation").
			WithArgs(envelopeID, 100000.0, int64(25000), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("insert failed"))

		err := repo.UpdateSpentAmount(envelopeID, targetDate, 25000, nil)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

