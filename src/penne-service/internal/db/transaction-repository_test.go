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

func TestPgTransactionRowsRepo_CreateTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Negative Amount", func(t *testing.T) {
		txn := &core.Transaction{AmountE5: -100, CountryISO: "US", Category: "Food", Type: "debit"}
		_, err := repo.CreateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction amount cannot be negative" {
			t.Errorf("expected negative amount error, got %v", err)
		}
	})

	t.Run("Empty CountryISO", func(t *testing.T) {
		txn := &core.Transaction{AmountE5: 100, CountryISO: "", Category: "Food", Type: "debit"}
		_, err := repo.CreateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction country ISO cannot be empty" {
			t.Errorf("expected empty country ISO error, got %v", err)
		}
	})

	t.Run("Empty Category", func(t *testing.T) {
		txn := &core.Transaction{AmountE5: 100, CountryISO: "US", Category: "", Type: "debit"}
		_, err := repo.CreateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction category cannot be empty" {
			t.Errorf("expected empty category error, got %v", err)
		}
	})

	t.Run("Empty Type", func(t *testing.T) {
		txn := &core.Transaction{AmountE5: 100, CountryISO: "US", Category: "Food", Type: ""}
		_, err := repo.CreateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction type cannot be empty" {
			t.Errorf("expected empty type error, got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		txn := &core.Transaction{
			AmountE5:   100,
			UserUUID:   userUUID,
			CountryISO: "US",
			Category:   "Food",
			BankName:   "Chase",
			Type:       "debit",
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO transactionrows").
			WithArgs(txn.AmountE5, txn.UserUUID, txn.CountryISO, txn.Category, txn.BankName, txn.Type).
			WillReturnError(errors.New("db error"))

		_, err := repo.CreateTransaction(txn, tx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		genUUID := uuid.New()
		txn := &core.Transaction{
			AmountE5:   100,
			UserUUID:   userUUID,
			CountryISO: "US",
			Category:   "Food",
			BankName:   "Chase",
			Type:       "debit",
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO transactionrows").
			WithArgs(txn.AmountE5, txn.UserUUID, txn.CountryISO, txn.Category, txn.BankName, txn.Type).
			WillReturnRows(sqlmock.NewRows([]string{"uuid"}).AddRow(genUUID))

		id, err := repo.CreateTransaction(txn, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != genUUID {
			t.Errorf("expected UUID %v, got %v", genUUID, id)
		}
	})
}

func TestPgTransactionRowsRepo_GetTransactionByUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174001")

	t.Run("Empty UUID", func(t *testing.T) {
		_, err := repo.GetTransactionByUUID(uuid.Nil)
		if err == nil || err.Error() != "transaction UUID is required" {
			t.Errorf("expected 'transaction UUID is required', got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE uuid = \\$1").
			WithArgs(validUUID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetTransactionByUUID(validUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"uuid", "amount_e5", "user_uuid", "country_iso2", "category", "bank_name", "txn_type", "created_at", "updated_at"}).
			AddRow(validUUID, float64(500), userUUID, "US", "Food", "Chase", "debit", now, now)

		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE uuid = \\$1").
			WithArgs(validUUID).
			WillReturnRows(rows)

		txn, err := repo.GetTransactionByUUID(validUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if txn.UUID != validUUID || txn.AmountE5 != 500 {
			t.Errorf("unexpected transaction: %+v", txn)
		}
	})
}

func TestPgTransactionRowsRepo_GetTransactionsByUserUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	now := time.Now()

	t.Run("Empty User UUID", func(t *testing.T) {
		_, err := repo.GetTransactionsByUserUUID(uuid.Nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected 'user UUID is required', got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE user_uuid = \\$1").
			WithArgs(userUUID).
			WillReturnError(errors.New("query failed"))

		_, err := repo.GetTransactionsByUserUUID(userUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Scan Error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"uuid", "amount_e5", "user_uuid", "country_iso2", "category", "bank_name", "txn_type", "created_at", "updated_at"}).
			AddRow("invalid_uuid", "invalid_number", userUUID, "US", "Food", "Chase", "debit", now, now)

		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE user_uuid = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		_, err := repo.GetTransactionsByUserUUID(userUUID)
		if err == nil {
			t.Error("expected scan error, got nil")
		}
	})

	t.Run("Rows Err", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"uuid", "amount_e5", "user_uuid", "country_iso2", "category", "bank_name", "txn_type", "created_at", "updated_at"}).
			AddRow(uuid.New(), float64(100), userUUID, "US", "Food", "Chase", "debit", now, now).
			RowError(0, errors.New("row error"))

		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE user_uuid = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		_, err := repo.GetTransactionsByUserUUID(userUUID)
		if err == nil {
			t.Error("expected rows error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		txn1UUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174001")
		txn2UUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174002")
		rows := sqlmock.NewRows([]string{"uuid", "amount_e5", "user_uuid", "country_iso2", "category", "bank_name", "txn_type", "created_at", "updated_at"}).
			AddRow(txn1UUID, float64(100), userUUID, "US", "Food", "Chase", "debit", now, now).
			AddRow(txn2UUID, float64(200), userUUID, "US", "Tech", "Citi", "credit", now, now)

		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE user_uuid = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		txns, err := repo.GetTransactionsByUserUUID(userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(txns) != 2 {
			t.Errorf("expected 2 transactions, got %d", len(txns))
		}
	})
}

func TestPgTransactionRowsRepo_UpdateTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Empty UUID", func(t *testing.T) {
		txn := &core.Transaction{UUID: uuid.Nil}
		err := repo.UpdateTransaction(txn)
		if err == nil || err.Error() != "transaction UUID is required" {
			t.Errorf("expected empty UUID error, got %v", err)
		}
	})

	t.Run("Negative Amount", func(t *testing.T) {
		txn := &core.Transaction{UUID: validUUID, AmountE5: -5}
		err := repo.UpdateTransaction(txn)
		if err == nil || err.Error() != "transaction amount cannot be negative" {
			t.Errorf("expected negative amount error, got %v", err)
		}
	})

	t.Run("Empty CountryISO", func(t *testing.T) {
		txn := &core.Transaction{UUID: validUUID, AmountE5: 5, CountryISO: ""}
		err := repo.UpdateTransaction(txn)
		if err == nil || err.Error() != "transaction country ISO cannot be empty" {
			t.Errorf("expected empty country ISO error, got %v", err)
		}
	})

	t.Run("Empty Category", func(t *testing.T) {
		txn := &core.Transaction{UUID: validUUID, AmountE5: 5, CountryISO: "US", Category: ""}
		err := repo.UpdateTransaction(txn)
		if err == nil || err.Error() != "transaction category cannot be empty" {
			t.Errorf("expected empty category error, got %v", err)
		}
	})

	t.Run("Empty Type", func(t *testing.T) {
		txn := &core.Transaction{UUID: validUUID, AmountE5: 5, CountryISO: "US", Category: "Food", Type: ""}
		err := repo.UpdateTransaction(txn)
		if err == nil || err.Error() != "transaction type cannot be empty" {
			t.Errorf("expected empty type error, got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		txn := &core.Transaction{UUID: validUUID, AmountE5: 5, CountryISO: "US", Category: "Food", BankName: "Chase", Type: "debit"}
		mock.ExpectExec("UPDATE transactionrows").
			WithArgs(txn.AmountE5, txn.CountryISO, txn.Category, txn.BankName, txn.Type, txn.UUID).
			WillReturnError(errors.New("update failed"))

		err := repo.UpdateTransaction(txn)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		txn := &core.Transaction{UUID: validUUID, AmountE5: 5, CountryISO: "US", Category: "Food", BankName: "Chase", Type: "debit"}
		mock.ExpectExec("UPDATE transactionrows").
			WithArgs(txn.AmountE5, txn.CountryISO, txn.Category, txn.BankName, txn.Type, txn.UUID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateTransaction(txn)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestPgTransactionRowsRepo_DeleteTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)
	validUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Empty UUID", func(t *testing.T) {
		err := repo.DeleteTransaction(uuid.Nil)
		if err == nil || err.Error() != "transaction UUID is required" {
			t.Errorf("expected empty UUID error, got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM transactionrows WHERE uuid = \\$1").
			WithArgs(validUUID).
			WillReturnError(errors.New("delete failed"))

		err := repo.DeleteTransaction(validUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM transactionrows WHERE uuid = \\$1").
			WithArgs(validUUID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.DeleteTransaction(validUUID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
