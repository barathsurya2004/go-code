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

func TestPgEnvelopeRepo_CreateEnvelope(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgEnvelopeRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Exec Error", func(t *testing.T) {
		now := time.Now()
		env := &core.Envelope{
			UserUUID:        userUUID,
			EnvelopeGroupID: uuid.New(),
			TargetAmountE5:  10000,
			Cadence:         "monthly",
			CountryISO:      "US",
			CreatedAt:       now,
			UpdatedAt:       now,
			IsSystem:        false,
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO envelope").
			WithArgs(
				env.UserUUID,
				env.EnvelopeGroupID,
				env.TargetAmountE5,
				env.Cadence,
				env.CountryISO,
				env.CreatedAt,
				env.UpdatedAt,
				env.IsSystem,
			).
			WillReturnError(errors.New("db error"))

		_, err := repo.CreateEnvelope(env, tx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		genID := uuid.New()
		env := &core.Envelope{
			UserUUID:        userUUID,
			EnvelopeGroupID: uuid.New(),
			TargetAmountE5:  10000,
			Cadence:         "monthly",
			CountryISO:      "US",
			CreatedAt:       now,
			UpdatedAt:       now,
			IsSystem:        false,
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO envelope").
			WithArgs(
				env.UserUUID,
				env.EnvelopeGroupID,
				env.TargetAmountE5,
				env.Cadence,
				env.CountryISO,
				env.CreatedAt,
				env.UpdatedAt,
				env.IsSystem,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(genID))

		id, err := repo.CreateEnvelope(env, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != genID {
			t.Errorf("expected ID %v, got %v", genID, id)
		}
	})
}

func TestPgEnvelopeRepo_GetEnvelopeByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgEnvelopeRepo(db)
	envID := uuid.New()
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, user_uuid, envelope_group_id, target_amount_e5, cadence, country_iso, created_at, updated_at, is_system").
			WithArgs(envID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetEnvelopeByID(envID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		groupID := uuid.New()
		rows := sqlmock.NewRows([]string{
			"id", "user_uuid", "envelope_group_id", "target_amount_e5", "cadence", "country_iso", "created_at", "updated_at", "is_system",
		}).AddRow(
			envID, userUUID, groupID, 50000.0, "monthly", "US", now, now, false,
		)

		mock.ExpectQuery("SELECT id, user_uuid, envelope_group_id, target_amount_e5, cadence, country_iso, created_at, updated_at, is_system").
			WithArgs(envID).
			WillReturnRows(rows)

		result, err := repo.GetEnvelopeByID(envID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.ID != envID {
			t.Errorf("expected ID %v, got %v", envID, result.ID)
		}
		if result.TargetAmountE5 != 50000.0 {
			t.Errorf("expected target amount 50000.0, got %v", result.TargetAmountE5)
		}
	})
}

func TestPgEnvelopeRepo_GetEnvelopesByUserUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgEnvelopeRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, user_uuid, envelope_group_id, target_amount_e5, cadence, country_iso, created_at, updated_at, is_system").
			WithArgs(userUUID).
			WillReturnError(errors.New("query failed"))

		_, err := repo.GetEnvelopesByUserUUID(userUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Scan Error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id"}).AddRow(123) // wrong types/columns
		mock.ExpectQuery("SELECT id, user_uuid, envelope_group_id, target_amount_e5, cadence, country_iso, created_at, updated_at, is_system").
			WithArgs(userUUID).
			WillReturnRows(rows)

		_, err := repo.GetEnvelopesByUserUUID(userUUID)
		if err == nil {
			t.Error("expected error scanning rows, got nil")
		}
	})

	t.Run("Row Iteration Error", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "user_uuid", "envelope_group_id", "target_amount_e5", "cadence", "country_iso", "created_at", "updated_at", "is_system",
		}).
			AddRow(uuid.New(), userUUID, uuid.New(), 1000.0, "monthly", "US", now, now, false).
			RowError(0, errors.New("row iteration error"))

		mock.ExpectQuery("SELECT id, user_uuid, envelope_group_id, target_amount_e5, cadence, country_iso, created_at, updated_at, is_system").
			WithArgs(userUUID).
			WillReturnRows(rows)

		_, err := repo.GetEnvelopesByUserUUID(userUUID)
		if err == nil {
			t.Error("expected rows.Err() error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		env1ID, env2ID := uuid.New(), uuid.New()
		group1ID, group2ID := uuid.New(), uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "user_uuid", "envelope_group_id", "target_amount_e5", "cadence", "country_iso", "created_at", "updated_at", "is_system",
		}).
			AddRow(env1ID, userUUID, group1ID, 1000.0, "monthly", "US", now, now, false).
			AddRow(env2ID, userUUID, group2ID, 2000.0, "weekly", "US", now, now, true)

		mock.ExpectQuery("SELECT id, user_uuid, envelope_group_id, target_amount_e5, cadence, country_iso, created_at, updated_at, is_system").
			WithArgs(userUUID).
			WillReturnRows(rows)

		results, err := repo.GetEnvelopesByUserUUID(userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 envelopes, got %d", len(results))
		}
		if results[0].ID != env1ID || results[1].ID != env2ID {
			t.Errorf("unexpected envelope IDs returned")
		}
	})
}

func TestPgEnvelopeRepo_UpdateEnvelope(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgEnvelopeRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Exec Error", func(t *testing.T) {
		now := time.Now()
		env := &core.Envelope{
			ID:              uuid.New(),
			UserUUID:        userUUID,
			EnvelopeGroupID: uuid.New(),
			TargetAmountE5:  15000,
			Cadence:         "monthly",
			CountryISO:      "US",
			UpdatedAt:       now,
			IsSystem:        false,
		}

		mock.ExpectExec("UPDATE envelope").
			WithArgs(
				env.ID,
				env.EnvelopeGroupID,
				env.TargetAmountE5,
				env.Cadence,
				env.CountryISO,
				env.UpdatedAt,
				env.IsSystem,
				env.UserUUID,
			).
			WillReturnError(errors.New("update error"))

		err := repo.UpdateEnvelope(env)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		env := &core.Envelope{
			ID:              uuid.New(),
			UserUUID:        userUUID,
			EnvelopeGroupID: uuid.New(),
			TargetAmountE5:  15000,
			Cadence:         "monthly",
			CountryISO:      "US",
			UpdatedAt:       now,
			IsSystem:        false,
		}

		mock.ExpectExec("UPDATE envelope").
			WithArgs(
				env.ID,
				env.EnvelopeGroupID,
				env.TargetAmountE5,
				env.Cadence,
				env.CountryISO,
				env.UpdatedAt,
				env.IsSystem,
				env.UserUUID,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateEnvelope(env)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestPgEnvelopeRepo_DeleteEnvelope(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgEnvelopeRepo(db)
	envID := uuid.New()

	t.Run("Exec Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM envelope").
			WithArgs(envID).
			WillReturnError(errors.New("delete error"))

		err := repo.DeleteEnvelope(envID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM envelope").
			WithArgs(envID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.DeleteEnvelope(envID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
