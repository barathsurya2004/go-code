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

func TestPgShortcutIntentRepo_CreateShortcutIntent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgShortcutIntentRepo(db)
	userUUID := uuid.New()
	envID := uuid.New()

	t.Run("Empty UserID Error", func(t *testing.T) {
		intent := &core.ShortcutIntent{
			UserID: uuid.Nil,
			Status: "pending",
		}
		_, err := repo.CreateShortcutIntent(intent, nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID is required error, got %v", err)
		}
	})

	t.Run("Empty Status Error", func(t *testing.T) {
		intent := &core.ShortcutIntent{
			UserID: userUUID,
			Status: "",
		}
		_, err := repo.CreateShortcutIntent(intent, nil)
		if err == nil || err.Error() != "shortcut intent status cannot be empty" {
			t.Errorf("expected shortcut intent status cannot be empty error, got %v", err)
		}
	})

	t.Run("Exec Error With Tx", func(t *testing.T) {
		intent := &core.ShortcutIntent{
			UserID:     userUUID,
			EnvelopeID: &envID,
			Latitude:   12.9716,
			Longitude:  77.5946,
			Status:     "pending",
			CreatedAt:  time.Now(),
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO shortcut_intent").
			WithArgs(intent.UserID, intent.EnvelopeID, intent.Latitude, intent.Longitude, intent.Status, sqlmock.AnyArg(), intent.TransactionID).
			WillReturnError(errors.New("db insert error"))

		_, err := repo.CreateShortcutIntent(intent, tx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success Without Tx and Zero CreatedAt", func(t *testing.T) {
		intent := &core.ShortcutIntent{
			UserID:    userUUID,
			Status:    "pending",
			Latitude:  12.9716,
			Longitude: 77.5946,
		}
		genID := uuid.New()
		mock.ExpectQuery("INSERT INTO shortcut_intent").
			WithArgs(intent.UserID, intent.EnvelopeID, intent.Latitude, intent.Longitude, intent.Status, sqlmock.AnyArg(), intent.TransactionID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(genID))

		id, err := repo.CreateShortcutIntent(intent, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != genID {
			t.Errorf("expected ID %v, got %v", genID, id)
		}
		if intent.ID != genID {
			t.Errorf("expected intent.ID to be set to %v, got %v", genID, intent.ID)
		}
		if intent.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set automatically")
		}
	})
}

func TestPgShortcutIntentRepo_GetShortcutIntentByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgShortcutIntentRepo(db)
	intentID := uuid.New()
	userUUID := uuid.New()
	envID := uuid.New()

	t.Run("Empty ID Error", func(t *testing.T) {
		_, err := repo.GetShortcutIntentByID(uuid.Nil)
		if err == nil || err.Error() != "shortcut intent UUID is required" {
			t.Errorf("expected shortcut intent UUID is required error, got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id FROM shortcut_intent WHERE id = \\$1").
			WithArgs(intentID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetShortcutIntentByID(intentID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "latitude", "longitude", "status", "created_at", "transaction_id"}).
			AddRow(intentID, userUUID, envID, 12.9716, 77.5946, "matched", now, nil)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id FROM shortcut_intent WHERE id = \\$1").
			WithArgs(intentID).
			WillReturnRows(rows)

		result, err := repo.GetShortcutIntentByID(intentID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.ID != intentID {
			t.Errorf("expected ID %v, got %v", intentID, result.ID)
		}
		if result.UserID != userUUID {
			t.Errorf("expected UserID %v, got %v", userUUID, result.UserID)
		}
		if result.Status != "matched" {
			t.Errorf("expected status 'matched', got %v", result.Status)
		}
	})
}

func TestPgShortcutIntentRepo_GetShortcutIntentsByUserUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgShortcutIntentRepo(db)
	userUUID := uuid.New()

	t.Run("Empty UserUUID Error", func(t *testing.T) {
		_, err := repo.GetShortcutIntentsByUserUUID(uuid.Nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID is required error, got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id FROM shortcut_intent WHERE user_id = \\$1").
			WithArgs(userUUID).
			WillReturnError(errors.New("query failed"))

		_, err := repo.GetShortcutIntentsByUserUUID(userUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Scan Error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id"}).AddRow("invalid-uuid")
		mock.ExpectQuery("SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id FROM shortcut_intent WHERE user_id = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		_, err := repo.GetShortcutIntentsByUserUUID(userUUID)
		if err == nil {
			t.Error("expected scan error, got nil")
		}
	})

	t.Run("Row Iteration Error", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "latitude", "longitude", "status", "created_at", "transaction_id"}).
			AddRow(uuid.New(), userUUID, uuid.New(), 12.9716, 77.5946, "pending", now, nil).
			RowError(0, errors.New("row iteration error"))

		mock.ExpectQuery("SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id FROM shortcut_intent WHERE user_id = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		_, err := repo.GetShortcutIntentsByUserUUID(userUUID)
		if err == nil {
			t.Error("expected row iteration error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		intent1ID, intent2ID := uuid.New(), uuid.New()
		env1ID, env2ID := uuid.New(), uuid.New()

		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "latitude", "longitude", "status", "created_at", "transaction_id"}).
			AddRow(intent1ID, userUUID, env1ID, 12.9716, 77.5946, "pending", now, nil).
			AddRow(intent2ID, userUUID, env2ID, 13.0827, 80.2707, "matched", now, nil)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id FROM shortcut_intent WHERE user_id = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		results, err := repo.GetShortcutIntentsByUserUUID(userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 shortcut intents, got %d", len(results))
		}
		if results[0].ID != intent1ID || results[1].ID != intent2ID {
			t.Errorf("unexpected shortcut intent IDs returned")
		}
	})
}

func TestPgShortcutIntentRepo_UpdateShortcutIntent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgShortcutIntentRepo(db)
	intentID := uuid.New()
	envID := uuid.New()

	t.Run("Empty ID Error", func(t *testing.T) {
		intent := &core.ShortcutIntent{
			ID:     uuid.Nil,
			Status: "pending",
		}
		err := repo.UpdateShortcutIntent(intent, nil)
		if err == nil || err.Error() != "shortcut intent UUID is required" {
			t.Errorf("expected shortcut intent UUID is required error, got %v", err)
		}
	})

	t.Run("Empty Status Error", func(t *testing.T) {
		intent := &core.ShortcutIntent{
			ID:     intentID,
			Status: "",
		}
		err := repo.UpdateShortcutIntent(intent, nil)
		if err == nil || err.Error() != "shortcut intent status cannot be empty" {
			t.Errorf("expected shortcut intent status cannot be empty error, got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		now := time.Now()
		intent := &core.ShortcutIntent{
			ID:         intentID,
			EnvelopeID: &envID,
			Latitude:   12.9716,
			Longitude:  77.5946,
			Status:     "matched",
			CreatedAt:  now,
		}

		mock.ExpectExec("UPDATE shortcut_intent").
			WithArgs(intent.EnvelopeID, intent.Latitude, intent.Longitude, intent.Status, intent.CreatedAt, intent.TransactionID, intent.ID).
			WillReturnError(errors.New("update error"))

		err := repo.UpdateShortcutIntent(intent, nil)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		intent := &core.ShortcutIntent{
			ID:         intentID,
			EnvelopeID: &envID,
			Latitude:   12.9716,
			Longitude:  77.5946,
			Status:     "matched",
			CreatedAt:  now,
		}

		mock.ExpectExec("UPDATE shortcut_intent").
			WithArgs(intent.EnvelopeID, intent.Latitude, intent.Longitude, intent.Status, intent.CreatedAt, intent.TransactionID, intent.ID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateShortcutIntent(intent, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Success With Tx", func(t *testing.T) {
		now := time.Now()
		intent := &core.ShortcutIntent{
			ID:         intentID,
			EnvelopeID: &envID,
			Latitude:   12.9716,
			Longitude:  77.5946,
			Status:     "matched",
			CreatedAt:  now,
		}

		mock.ExpectBegin()
		tx, _ := db.Begin()

		mock.ExpectExec("UPDATE shortcut_intent").
			WithArgs(intent.EnvelopeID, intent.Latitude, intent.Longitude, intent.Status, intent.CreatedAt, intent.TransactionID, intent.ID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateShortcutIntent(intent, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestPgShortcutIntentRepo_DeleteShortcutIntent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgShortcutIntentRepo(db)
	intentID := uuid.New()

	t.Run("Empty ID Error", func(t *testing.T) {
		err := repo.DeleteShortcutIntent(uuid.Nil)
		if err == nil || err.Error() != "shortcut intent UUID is required" {
			t.Errorf("expected shortcut intent UUID is required error, got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM shortcut_intent WHERE id = \\$1").
			WithArgs(intentID).
			WillReturnError(errors.New("delete error"))

		err := repo.DeleteShortcutIntent(intentID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM shortcut_intent WHERE id = \\$1").
			WithArgs(intentID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.DeleteShortcutIntent(intentID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestPgShortcutIntentRepo_GetPendingRecentShortcutIntent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgShortcutIntentRepo(db)
	userUUID := uuid.New()
	intentID := uuid.New()
	now := time.Now()
	tLower := now.Add(-10 * time.Minute)
	tUpper := now.Add(3 * time.Minute)

	t.Run("Empty UserUUID Error", func(t *testing.T) {
		_, err := repo.GetPendingRecentShortcutIntent(uuid.Nil, nil, tLower, tUpper)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID is required error, got %v", err)
		}
	})

	t.Run("Zero Time Error", func(t *testing.T) {
		_, err := repo.GetPendingRecentShortcutIntent(userUUID, nil, time.Time{}, tUpper)
		if err == nil || err.Error() != "time lowerbound and upperbound are required" {
			t.Errorf("expected time lowerbound and upperbound are required error, got %v", err)
		}
	})

	t.Run("Lowerbound After Upperbound Error", func(t *testing.T) {
		_, err := repo.GetPendingRecentShortcutIntent(userUUID, nil, tUpper, tLower)
		if err == nil || err.Error() != "time lowerbound cannot be after time upperbound" {
			t.Errorf("expected lowerbound cannot be after upperbound error, got %v", err)
		}
	})

	t.Run("Success With Tx", func(t *testing.T) {
		mock.ExpectBegin()
		tx, _ := db.Begin()
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "latitude", "longitude", "status", "created_at", "transaction_id"}).
			AddRow(intentID, userUUID, nil, 12.97, 77.59, "pending", now, nil)
		mock.ExpectQuery("SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id FROM shortcut_intent").
			WithArgs(userUUID, core.StatusPending, tLower, tUpper).
			WillReturnRows(rows)

		res, err := repo.GetPendingRecentShortcutIntent(userUUID, tx, tLower, tUpper)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.ID != intentID {
			t.Errorf("expected intent ID %v, got %v", intentID, res.ID)
		}
	})

	t.Run("Success Without Tx", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "latitude", "longitude", "status", "created_at", "transaction_id"}).
			AddRow(intentID, userUUID, nil, 12.97, 77.59, "pending", now, nil)
		mock.ExpectQuery("SELECT id, user_id, envelope_id, latitude, longitude, status, created_at, transaction_id FROM shortcut_intent").
			WithArgs(userUUID, core.StatusPending, tLower, tUpper).
			WillReturnRows(rows)

		res, err := repo.GetPendingRecentShortcutIntent(userUUID, nil, tLower, tUpper)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.ID != intentID {
			t.Errorf("expected intent ID %v, got %v", intentID, res.ID)
		}
	})
}
