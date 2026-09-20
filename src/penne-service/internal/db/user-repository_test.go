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
			Name:            "Alice",
			Email:           "alice@example.com",
			PasswordHash:    "hashed",
			MonthlyBudgetE5: 10000000,
			SalaryDay:       25,
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO users").
			WithArgs(user.Name, user.Email, user.PasswordHash, user.MonthlyBudgetE5, user.SalaryDay).
			WillReturnError(errors.New("db error"))

		_, err := repo.CreateUser(user, tx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		user := &core.User{
			Name:            "Alice",
			Email:           "alice@example.com",
			PasswordHash:    "hashed",
			MonthlyBudgetE5: 10000000,
			SalaryDay:       25,
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO users").
			WithArgs(user.Name, user.Email, user.PasswordHash, user.MonthlyBudgetE5, user.SalaryDay).
			WillReturnRows(sqlmock.NewRows([]string{"uuid"}).AddRow(genUUID))

		id, err := repo.CreateUser(user, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != genUUID {
			t.Errorf("expected UUID %v, got %v", genUUID, id)
		}
	})

	t.Run("Success Without Tx", func(t *testing.T) {
		user := &core.User{
			Name:         "Alice",
			Email:        "alice@example.com",
			PasswordHash: "hashed",
		}
		mock.ExpectQuery("INSERT INTO users").
			WithArgs(user.Name, user.Email, user.PasswordHash, int64(0), 1).
			WillReturnRows(sqlmock.NewRows([]string{"uuid"}).AddRow(genUUID))

		id, err := repo.CreateUser(user, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != genUUID {
			t.Errorf("expected UUID %v, got %v", genUUID, id)
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
		mock.ExpectQuery("SELECT uuid, name, created_at, updated_at, monthly_budget_e5, salary_day FROM users").
			WithArgs(validUUID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetUserByUUID(validUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()

		rows := sqlmock.NewRows([]string{"uuid", "name", "created_at", "updated_at", "monthly_budget_e5", "salary_day"}).
			AddRow(validUUID, "Alice", now, now, int64(5000000), 25)

		mock.ExpectQuery("SELECT uuid, name, created_at, updated_at, monthly_budget_e5, salary_day FROM users").
			WithArgs(validUUID).
			WillReturnRows(rows)

		user, err := repo.GetUserByUUID(validUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.UUID != validUUID || user.Name != "Alice" || user.MonthlyBudgetE5 != 5000000 || user.SalaryDay != 25 {
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
		mock.ExpectQuery("SELECT uuid, name, created_at, updated_at, password_hash, email, monthly_budget_e5, salary_day").
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetUserByEmail(email)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()

		rows := sqlmock.NewRows([]string{"uuid", "name", "created_at", "updated_at", "password_hash", "email", "monthly_budget_e5", "salary_day"}).
			AddRow(validUUID, "Alice", now, now, "hashed", email, int64(5000000), 25)

		mock.ExpectQuery("SELECT uuid, name, created_at, updated_at, password_hash, email, monthly_budget_e5, salary_day").
			WithArgs(email).
			WillReturnRows(rows)

		user, err := repo.GetUserByEmail(email)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.UUID != validUUID || user.Name != "Alice" || user.Email != email || user.PasswordHash != "hashed" || user.MonthlyBudgetE5 != 5000000 || user.SalaryDay != 25 {
			t.Errorf("unexpected user returned: %+v", user)
		}
	})
}

func TestPgUserRepo_UpdateBudgetSettings(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgUserRepo(db)
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Empty UUID", func(t *testing.T) {
		err := repo.UpdateBudgetSettings(uuid.Nil, 5000000, 25, nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected 'user UUID is required', got %v", err)
		}
	})

	t.Run("Negative Monthly Budget", func(t *testing.T) {
		err := repo.UpdateBudgetSettings(validUUID, -1, 25, nil)
		if err == nil || err.Error() != "monthly budget cannot be negative" {
			t.Errorf("expected 'monthly budget cannot be negative', got %v", err)
		}
	})

	t.Run("Invalid Salary Day Low", func(t *testing.T) {
		err := repo.UpdateBudgetSettings(validUUID, 5000000, 0, nil)
		if err == nil || err.Error() != "salary day must be between 1 and 31" {
			t.Errorf("expected 'salary day must be between 1 and 31', got %v", err)
		}
	})

	t.Run("Invalid Salary Day High", func(t *testing.T) {
		err := repo.UpdateBudgetSettings(validUUID, 5000000, 32, nil)
		if err == nil || err.Error() != "salary day must be between 1 and 31" {
			t.Errorf("expected 'salary day must be between 1 and 31', got %v", err)
		}
	})

	t.Run("Success With Tx", func(t *testing.T) {
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectExec("UPDATE users").
			WithArgs(int64(5000000), 25, validUUID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateBudgetSettings(validUUID, 5000000, 25, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Success Without Tx", func(t *testing.T) {
		mock.ExpectExec("UPDATE users").
			WithArgs(int64(5000000), 25, validUUID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateBudgetSettings(validUUID, 5000000, 25, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
