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

func TestPgUserRepo_CreateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgUserRepo(db)
	genUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Empty Name", func(t *testing.T) {
		_, err := repo.CreateUser(&core.User{Name: "   ", Email: "alice@example.com", PasswordHash: "hashed"}, nil)
		if err == nil || err.Error() != "user name is required" {
			t.Errorf("expected 'user name is required', got %v", err)
		}
	})

	t.Run("Empty Email", func(t *testing.T) {
		_, err := repo.CreateUser(&core.User{Name: "Alice", Email: "", PasswordHash: "hashed"}, nil)
		if err == nil || err.Error() != "user email is required" {
			t.Errorf("expected 'user email is required', got %v", err)
		}
	})

	t.Run("Empty PasswordHash", func(t *testing.T) {
		_, err := repo.CreateUser(&core.User{Name: "Alice", Email: "alice@example.com", PasswordHash: ""}, nil)
		if err == nil || err.Error() != "user password hash is required" {
			t.Errorf("expected 'user password hash is required', got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		user := &core.User{
			Name:         "Alice",
			Email:        "alice@example.com",
			PasswordHash: "hashed",
		}
		// mock transaction
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO users").
			WithArgs(user.Name, user.Email, user.PasswordHash).
			WillReturnError(errors.New("db error"))

		_, err := repo.CreateUser(user, tx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		user := &core.User{
			Name:         "Alice",
			Email:        "alice@example.com",
			PasswordHash: "hashed",
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO users").
			WithArgs(user.Name, user.Email, user.PasswordHash).
			WillReturnRows(sqlmock.NewRows([]string{"uuid"}).AddRow(genUUID))

		id, err := repo.CreateUser(user, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != genUUID {
			t.Errorf("expected ID %v, got %v", genUUID, id)
		}
	})
}

func TestPgUserRepo_GetUserByUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgUserRepo(db)
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Empty UUID", func(t *testing.T) {
		_, err := repo.GetUserByUUID(uuid.Nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected 'user UUID is required', got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT uuid, name, created_at, updated_at FROM users").
			WithArgs(validUUID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetUserByUUID(validUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()

		rows := sqlmock.NewRows([]string{"uuid", "name", "created_at", "updated_at"}).
			AddRow(validUUID, "Alice", now, now)

		mock.ExpectQuery("SELECT uuid, name, created_at, updated_at FROM users").
			WithArgs(validUUID).
			WillReturnRows(rows)

		user, err := repo.GetUserByUUID(validUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.UUID != validUUID || user.Name != "Alice" {
			t.Errorf("unexpected user returned: %+v", user)
		}
	})
}

func TestPgUserRepo_GetUserByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgUserRepo(db)
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	email := "alice@example.com"

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT uuid, name, created_at, updated_at, password_hash, email").
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetUserByEmail(email)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()

		rows := sqlmock.NewRows([]string{"uuid", "name", "created_at", "updated_at", "password_hash", "email"}).
			AddRow(validUUID, "Alice", now, now, "hashed", email)

		mock.ExpectQuery("SELECT uuid, name, created_at, updated_at, password_hash, email").
			WithArgs(email).
			WillReturnRows(rows)

		user, err := repo.GetUserByEmail(email)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.UUID != validUUID || user.Name != "Alice" || user.Email != email || user.PasswordHash != "hashed" {
			t.Errorf("unexpected user returned: %+v", user)
		}
	})
}
