package db

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
)

func TestPgTransactionRowsRepo_CreateTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)

	t.Run("Negative Amount", func(t *testing.T) {
		txn := &core.Transaction{AmountE5: -100, CountryISO: "US", Category: "Food", Type: "debit"}
		err := repo.CreateTransaction(txn)
		if err == nil || err.Error() != "transaction amount cannot be negative" {
			t.Errorf("expected negative amount error, got %v", err)
		}
	})

	t.Run("Empty CountryISO", func(t *testing.T) {
		txn := &core.Transaction{AmountE5: 100, CountryISO: "", Category: "Food", Type: "debit"}
		err := repo.CreateTransaction(txn)
		if err == nil || err.Error() != "transaction country ISO cannot be empty" {
			t.Errorf("expected empty country ISO error, got %v", err)
		}
	})

	t.Run("Empty Category", func(t *testing.T) {
		txn := &core.Transaction{AmountE5: 100, CountryISO: "US", Category: "", Type: "debit"}
		err := repo.CreateTransaction(txn)
		if err == nil || err.Error() != "transaction category cannot be empty" {
			t.Errorf("expected empty category error, got %v", err)
		}
	})

	t.Run("Empty Type", func(t *testing.T) {
		txn := &core.Transaction{AmountE5: 100, CountryISO: "US", Category: "Food", Type: ""}
		err := repo.CreateTransaction(txn)
		if err == nil || err.Error() != "transaction type cannot be empty" {
			t.Errorf("expected empty type error, got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		txn := &core.Transaction{
			AmountE5:   100,
			UserUUID:   "123e4567-e89b-12d3-a456-426614174000",
			CountryISO: "US",
			Category:   "Food",
			BankName:   "Chase",
			Type:       "debit",
		}
		mock.ExpectExec("INSERT INTO transactionrows").
			WithArgs(sqlmock.AnyArg(), txn.AmountE5, txn.UserUUID, txn.CountryISO, txn.Category, txn.BankName, txn.Type).
			WillReturnError(errors.New("db error"))

		err := repo.CreateTransaction(txn)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		txn := &core.Transaction{
			AmountE5:   100,
			UserUUID:   "123e4567-e89b-12d3-a456-426614174000",
			CountryISO: "US",
			Category:   "Food",
			BankName:   "Chase",
			Type:       "debit",
		}
		mock.ExpectExec("INSERT INTO transactionrows").
			WithArgs(sqlmock.AnyArg(), txn.AmountE5, txn.UserUUID, txn.CountryISO, txn.Category, txn.BankName, txn.Type).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.CreateTransaction(txn)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if txn.UUID == "" {
			t.Error("expected generated UUID")
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

	t.Run("Empty UUID", func(t *testing.T) {
		_, err := repo.GetTransactionByUUID("")
		if err == nil || err.Error() != "transaction UUID is required" {
			t.Errorf("expected 'transaction UUID is required', got %v", err)
		}
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		_, err := repo.GetTransactionByUUID("invalid")
		if err == nil || err.Error() != "transaction UUID is invalid" {
			t.Errorf("expected 'transaction UUID is invalid', got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		validUUID := "123e4567-e89b-12d3-a456-426614174000"
		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE uuid = \\$1").
			WithArgs(validUUID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetTransactionByUUID(validUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		validUUID := "123e4567-e89b-12d3-a456-426614174000"
		rows := sqlmock.NewRows([]string{"uuid", "amount_e5", "user_uuid", "country_iso2", "category", "bank_name", "txn_type", "created_at", "updated_at"}).
			AddRow(validUUID, float64(500), "user-1", "US", "Food", "Chase", "debit", "2026-01-01", "2026-01-01")

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
	userUUID := "123e4567-e89b-12d3-a456-426614174000"

	t.Run("Empty User UUID", func(t *testing.T) {
		_, err := repo.GetTransactionsByUserUUID("")
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected 'user UUID is required', got %v", err)
		}
	})

	t.Run("Invalid User UUID", func(t *testing.T) {
		_, err := repo.GetTransactionsByUserUUID("invalid")
		if err == nil || err.Error() != "user UUID is invalid" {
			t.Errorf("expected 'user UUID is invalid', got %v", err)
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
			AddRow("txn-1", "invalid_number", "user-1", "US", "Food", "Chase", "debit", "2026-01-01", "2026-01-01")

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
			AddRow("txn-1", float64(100), "user-1", "US", "Food", "Chase", "debit", "2026-01-01", "2026-01-01").
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
		rows := sqlmock.NewRows([]string{"uuid", "amount_e5", "user_uuid", "country_iso2", "category", "bank_name", "txn_type", "created_at", "updated_at"}).
			AddRow("123e4567-e89b-12d3-a456-426614174001", float64(100), userUUID, "US", "Food", "Chase", "debit", "2026-01-01", "2026-01-01").
			AddRow("123e4567-e89b-12d3-a456-426614174002", float64(200), userUUID, "US", "Tech", "Citi", "credit", "2026-01-02", "2026-01-02")

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
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	t.Run("Empty UUID", func(t *testing.T) {
		txn := &core.Transaction{UUID: ""}
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

	t.Run("Invalid UUID", func(t *testing.T) {
		txn := &core.Transaction{UUID: "invalid", AmountE5: 5, CountryISO: "US", Category: "Food", Type: "debit"}
		err := repo.UpdateTransaction(txn)
		if err == nil || err.Error() != "transaction UUID is invalid" {
			t.Errorf("expected invalid UUID error, got %v", err)
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
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	t.Run("Empty UUID", func(t *testing.T) {
		err := repo.DeleteTransaction("")
		if err == nil || err.Error() != "transaction UUID is required" {
			t.Errorf("expected empty UUID error, got %v", err)
		}
	})

	t.Run("Invalid UUID", func(t *testing.T) {
		err := repo.DeleteTransaction("invalid")
		if err == nil || err.Error() != "transaction UUID is invalid" {
			t.Errorf("expected invalid UUID error, got %v", err)
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
