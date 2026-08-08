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
			CreatedAt:         now,
			UpdatedAt:         now,
			StartDate:         &now,
			EndDate:           &now,
		}

		mock.ExpectExec("INSERT INTO allocation").
			WithArgs(alloc.EnvelopeID, alloc.AllocatedAmountE5, alloc.CreatedAt, alloc.UpdatedAt, alloc.StartDate, alloc.EndDate).
			WillReturnError(errors.New("db error"))

		err := repo.CreateAllocation(alloc)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		alloc := &core.Allocation{
			EnvelopeID:        uuid.New(),
			AllocatedAmountE5: 150000,
			CreatedAt:         now,
			UpdatedAt:         now,
			StartDate:         &now,
			EndDate:           &now,
		}

		mock.ExpectExec("INSERT INTO allocation").
			WithArgs(alloc.EnvelopeID, alloc.AllocatedAmountE5, alloc.CreatedAt, alloc.UpdatedAt, alloc.StartDate, alloc.EndDate).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreateAllocation(alloc)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
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
		mock.ExpectQuery("SELECT id, envelope_id, amount_e5, created_at, updated_at, start_date, end_date").
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

		rows := sqlmock.NewRows([]string{"id", "envelope_id", "amount_e5", "created_at", "updated_at", "start_date", "end_date"}).
			AddRow(allocID, envelopeID, 250000.0, now, now, now, now)

		mock.ExpectQuery("SELECT id, envelope_id, amount_e5, created_at, updated_at, start_date, end_date").
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
		mock.ExpectQuery("SELECT id, envelope_id, amount_e5, created_at, updated_at, start_date, end_date").
			WithArgs(envelopeID).
			WillReturnError(errors.New("query error"))

		_, err := repo.GetAllocationsByEnvelopeID(envelopeID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Scan Error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id"}).AddRow("invalid")
		mock.ExpectQuery("SELECT id, envelope_id, amount_e5, created_at, updated_at, start_date, end_date").
			WithArgs(envelopeID).
			WillReturnRows(rows)

		_, err := repo.GetAllocationsByEnvelopeID(envelopeID)
		if err == nil {
			t.Error("expected scan error, got nil")
		}
	})

	t.Run("Row Iteration Error", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "envelope_id", "amount_e5", "created_at", "updated_at", "start_date", "end_date"}).
			AddRow(uuid.New(), envelopeID, 100000.0, now, now, now, now).
			RowError(0, errors.New("iteration error"))

		mock.ExpectQuery("SELECT id, envelope_id, amount_e5, created_at, updated_at, start_date, end_date").
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

		rows := sqlmock.NewRows([]string{"id", "envelope_id", "amount_e5", "created_at", "updated_at", "start_date", "end_date"}).
			AddRow(alloc1ID, envelopeID, 100000.0, now, now, now, now).
			AddRow(alloc2ID, envelopeID, 200000.0, now, now, now, now)

		mock.ExpectQuery("SELECT id, envelope_id, amount_e5, created_at, updated_at, start_date, end_date").
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
			CreatedAt:         now,
			UpdatedAt:         now,
			StartDate:         &now,
			EndDate:           &now,
		}

		mock.ExpectExec("UPDATE allocation").
			WithArgs(alloc.ID, alloc.EnvelopeID, alloc.AllocatedAmountE5, alloc.CreatedAt, alloc.UpdatedAt, alloc.StartDate, alloc.EndDate).
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
			CreatedAt:         now,
			UpdatedAt:         now,
			StartDate:         &now,
			EndDate:           &now,
		}

		mock.ExpectExec("UPDATE allocation").
			WithArgs(alloc.ID, alloc.EnvelopeID, alloc.AllocatedAmountE5, alloc.CreatedAt, alloc.UpdatedAt, alloc.StartDate, alloc.EndDate).
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
