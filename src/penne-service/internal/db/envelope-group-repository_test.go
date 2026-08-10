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

func TestEnvelopeGroupRepository_CreateEnvelopeGroup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewEnvelopeGroupRepository(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Empty Name Error", func(t *testing.T) {
		group := &core.EnvelopeGroup{
			UserUUID: userUUID,
			Name:     "   ",
		}
		_, err := repo.CreateEnvelopeGroup(group, nil)
		if err == nil || err.Error() != "envelope group name is required" {
			t.Errorf("expected 'envelope group name is required', got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		group := &core.EnvelopeGroup{
			UserUUID: userUUID,
			Name:     "Savings",
			IsSystem: false,
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO envelope_group").
			WithArgs(group.UserUUID, group.Name, group.IsSystem).
			WillReturnError(errors.New("db error"))

		_, err := repo.CreateEnvelopeGroup(group, tx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		group := &core.EnvelopeGroup{
			UserUUID: userUUID,
			Name:     "Savings",
			IsSystem: false,
		}
		genID := uuid.New()
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO envelope_group").
			WithArgs(group.UserUUID, group.Name, group.IsSystem).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(genID))

		id, err := repo.CreateEnvelopeGroup(group, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != genID {
			t.Errorf("expected ID %v, got %v", genID, id)
		}
	})

	t.Run("Success Without Tx", func(t *testing.T) {
		group := &core.EnvelopeGroup{
			UserUUID: userUUID,
			Name:     "Savings",
			IsSystem: false,
		}
		genID := uuid.New()
		mock.ExpectQuery("INSERT INTO envelope_group").
			WithArgs(group.UserUUID, group.Name, group.IsSystem).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(genID))

		id, err := repo.CreateEnvelopeGroup(group, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != genID {
			t.Errorf("expected ID %v, got %v", genID, id)
		}
	})
}

func TestEnvelopeGroupRepository_GetEnvelopeGroupByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewEnvelopeGroupRepository(db)
	validID := uuid.New()
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Invalid ID", func(t *testing.T) {
		_, err := repo.GetEnvelopeGroupByID(uuid.Nil)
		if err == nil || err.Error() != "envelope group ID is invalid" {
			t.Errorf("expected 'envelope group ID is invalid', got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, user_uuid, name, is_system").
			WithArgs(validID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetEnvelopeGroupByID(validID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "user_uuid", "name", "is_system", "created_at", "updated_at"}).
			AddRow(validID, userUUID, "Bills", true, now, now)

		mock.ExpectQuery("SELECT id, user_uuid, name, is_system").
			WithArgs(validID).
			WillReturnRows(rows)

		res, err := repo.GetEnvelopeGroupByID(validID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.ID != validID {
			t.Errorf("expected ID %v, got %v", validID, res.ID)
		}
		if res.Name != "Bills" {
			t.Errorf("expected name Bills, got %s", res.Name)
		}
	})
}

func TestEnvelopeGroupRepository_GetEnvelopeGroupsByUserUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewEnvelopeGroupRepository(db)
	validUserUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Invalid User UUID", func(t *testing.T) {
		_, err := repo.GetEnvelopeGroupsByUserUUID(uuid.Nil)
		if err == nil || err.Error() != "user UUID is invalid" {
			t.Errorf("expected 'user UUID is invalid', got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, user_uuid, name, is_system").
			WithArgs(validUserUUID).
			WillReturnError(errors.New("query failed"))

		_, err := repo.GetEnvelopeGroupsByUserUUID(validUserUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Scan Error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id"}).AddRow("invalid")
		mock.ExpectQuery("SELECT id, user_uuid, name, is_system").
			WithArgs(validUserUUID).
			WillReturnRows(rows)

		_, err := repo.GetEnvelopeGroupsByUserUUID(validUserUUID)
		if err == nil {
			t.Error("expected error scanning rows, got nil")
		}
	})

	t.Run("Row Iteration Error", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "user_uuid", "name", "is_system", "created_at", "updated_at"}).
			AddRow(uuid.New(), validUserUUID, "Investments", false, now, now).
			RowError(0, errors.New("row iteration error"))

		mock.ExpectQuery("SELECT id, user_uuid, name, is_system").
			WithArgs(validUserUUID).
			WillReturnRows(rows)

		_, err := repo.GetEnvelopeGroupsByUserUUID(validUserUUID)
		if err == nil {
			t.Error("expected rows.Err() error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		id1, id2 := uuid.New(), uuid.New()
		rows := sqlmock.NewRows([]string{"id", "user_uuid", "name", "is_system", "created_at", "updated_at"}).
			AddRow(id1, validUserUUID, "Emergency Fund", false, now, now).
			AddRow(id2, validUserUUID, "Vacation", false, now, now)

		mock.ExpectQuery("SELECT id, user_uuid, name, is_system").
			WithArgs(validUserUUID).
			WillReturnRows(rows)

		results, err := repo.GetEnvelopeGroupsByUserUUID(validUserUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 envelope groups, got %d", len(results))
		}
	})
}

func TestEnvelopeGroupRepository_UpdateEnvelopeGroup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewEnvelopeGroupRepository(db)

	t.Run("Empty Name Error", func(t *testing.T) {
		group := &core.EnvelopeGroup{
			ID:   uuid.New(),
			Name: "",
		}
		err := repo.UpdateEnvelopeGroup(group)
		if err == nil || err.Error() != "envelope group name is required" {
			t.Errorf("expected 'envelope group name is required', got %v", err)
		}
	})

	t.Run("Invalid ID", func(t *testing.T) {
		group := &core.EnvelopeGroup{
			ID:   uuid.Nil,
			Name: "Valid Name",
		}
		err := repo.UpdateEnvelopeGroup(group)
		if err == nil || err.Error() != "envelope group ID is invalid" {
			t.Errorf("expected 'envelope group ID is invalid', got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		group := &core.EnvelopeGroup{
			ID:       uuid.New(),
			Name:     "New Name",
			IsSystem: false,
		}

		mock.ExpectExec("UPDATE envelope_group").
			WithArgs(group.Name, group.IsSystem, group.ID).
			WillReturnError(errors.New("update failed"))

		err := repo.UpdateEnvelopeGroup(group)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		group := &core.EnvelopeGroup{
			ID:       uuid.New(),
			Name:     "New Name",
			IsSystem: false,
		}

		mock.ExpectExec("UPDATE envelope_group").
			WithArgs(group.Name, group.IsSystem, group.ID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateEnvelopeGroup(group)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestEnvelopeGroupRepository_DeleteEnvelopeGroup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewEnvelopeGroupRepository(db)
	validID := uuid.New()

	t.Run("Invalid ID", func(t *testing.T) {
		err := repo.DeleteEnvelopeGroup(uuid.Nil)
		if err == nil || err.Error() != "envelope group ID is invalid" {
			t.Errorf("expected 'envelope group ID is invalid', got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM envelope_group").
			WithArgs(validID).
			WillReturnError(errors.New("delete failed"))

		err := repo.DeleteEnvelopeGroup(validID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM envelope_group").
			WithArgs(validID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.DeleteEnvelopeGroup(validID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
