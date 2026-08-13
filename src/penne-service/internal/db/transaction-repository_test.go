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
		txn := &core.Transaction{AmountE5: -100, CountryISO: "US", Type: "debit"}
		_, err := repo.CreateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction amount cannot be negative" {
			t.Errorf("expected negative amount error, got %v", err)
		}
	})

	t.Run("Empty CountryISO", func(t *testing.T) {
		txn := &core.Transaction{AmountE5: 100, CountryISO: "", Type: "debit"}
		_, err := repo.CreateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction country ISO cannot be empty" {
			t.Errorf("expected empty country ISO error, got %v", err)
		}
	})

	t.Run("Empty Type", func(t *testing.T) {
		txn := &core.Transaction{AmountE5: 100, CountryISO: "US", Type: ""}
		_, err := repo.CreateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction type cannot be empty" {
			t.Errorf("expected empty type error, got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		txn := &core.Transaction{
			AmountE5:      100,
			UserID:        userUUID,
			CountryISO:    "US",
			PaymentMethod: "Chase",
			Type:          "debit",
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO transactionrows").
			WithArgs(txn.UserID, txn.EnvelopeID, txn.AmountE5, txn.CountryISO, txn.PaymentMethod, txn.Type, sqlmock.AnyArg()).
			WillReturnError(errors.New("db error"))

		_, err := repo.CreateTransaction(txn, tx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		genUUID := uuid.New()
		txn := &core.Transaction{
			AmountE5:      100,
			UserID:        userUUID,
			CountryISO:    "US",
			PaymentMethod: "Chase",
			Type:          "debit",
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectQuery("INSERT INTO transactionrows").
			WithArgs(txn.UserID, txn.EnvelopeID, txn.AmountE5, txn.CountryISO, txn.PaymentMethod, txn.Type, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(genUUID))

		id, err := repo.CreateTransaction(txn, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id != genUUID {
			t.Errorf("expected UUID %v, got %v", genUUID, id)
		}
	})

	t.Run("Success Without Tx", func(t *testing.T) {
		genUUID := uuid.New()
		txn := &core.Transaction{
			AmountE5:      100,
			UserID:        userUUID,
			CountryISO:    "US",
			PaymentMethod: "Chase",
			Type:          "debit",
		}
		mock.ExpectQuery("INSERT INTO transactionrows").
			WithArgs(txn.UserID, txn.EnvelopeID, txn.AmountE5, txn.CountryISO, txn.PaymentMethod, txn.Type, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(genUUID))

		id, err := repo.CreateTransaction(txn, nil)
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
		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE id = \\$1").
			WithArgs(validUUID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetTransactionByUUID(validUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at"}).
			AddRow(validUUID, userUUID, nil, int64(500), "US", "Chase", "debit", now)

		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE id = \\$1").
			WithArgs(validUUID).
			WillReturnRows(rows)

		txn, err := repo.GetTransactionByUUID(validUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if txn.ID != validUUID || txn.AmountE5 != 500 {
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
		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE user_id = \\$1").
			WithArgs(userUUID).
			WillReturnError(errors.New("query failed"))

		_, err := repo.GetTransactionsByUserUUID(userUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Scan Error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at"}).
			AddRow("invalid_uuid", userUUID, nil, "invalid_number", "US", "Chase", "debit", now)

		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE user_id = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		_, err := repo.GetTransactionsByUserUUID(userUUID)
		if err == nil {
			t.Error("expected scan error, got nil")
		}
	})

	t.Run("Rows Err", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at"}).
			AddRow(uuid.New(), userUUID, nil, int64(100), "US", "Chase", "debit", now).
			RowError(0, errors.New("row error"))

		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE user_id = \\$1").
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
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at"}).
			AddRow(txn1UUID, userUUID, nil, int64(100), "US", "Chase", "debit", now).
			AddRow(txn2UUID, userUUID, nil, int64(200), "US", "Citi", "credit", now)

		mock.ExpectQuery("SELECT (.+) FROM transactionrows WHERE user_id = \\$1").
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
		txn := &core.Transaction{ID: uuid.Nil}
		err := repo.UpdateTransaction(txn)
		if err == nil || err.Error() != "transaction UUID is required" {
			t.Errorf("expected empty UUID error, got %v", err)
		}
	})

	t.Run("Negative Amount", func(t *testing.T) {
		txn := &core.Transaction{ID: validUUID, AmountE5: -5}
		err := repo.UpdateTransaction(txn)
		if err == nil || err.Error() != "transaction amount cannot be negative" {
			t.Errorf("expected negative amount error, got %v", err)
		}
	})

	t.Run("Empty CountryISO", func(t *testing.T) {
		txn := &core.Transaction{ID: validUUID, AmountE5: 5, CountryISO: ""}
		err := repo.UpdateTransaction(txn)
		if err == nil || err.Error() != "transaction country ISO cannot be empty" {
			t.Errorf("expected empty country ISO error, got %v", err)
		}
	})

	t.Run("Empty Type", func(t *testing.T) {
		txn := &core.Transaction{ID: validUUID, AmountE5: 5, CountryISO: "US", Type: ""}
		err := repo.UpdateTransaction(txn)
		if err == nil || err.Error() != "transaction type cannot be empty" {
			t.Errorf("expected empty type error, got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		txn := &core.Transaction{ID: validUUID, AmountE5: 5, CountryISO: "US", PaymentMethod: "Chase", Type: "debit"}
		mock.ExpectExec("UPDATE transactionrows").
			WithArgs(txn.EnvelopeID, txn.AmountE5, txn.CountryISO, txn.PaymentMethod, txn.Type, txn.ID).
			WillReturnError(errors.New("update failed"))

		err := repo.UpdateTransaction(txn)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		txn := &core.Transaction{ID: validUUID, AmountE5: 5, CountryISO: "US", PaymentMethod: "Chase", Type: "debit"}
		mock.ExpectExec("UPDATE transactionrows").
			WithArgs(txn.EnvelopeID, txn.AmountE5, txn.CountryISO, txn.PaymentMethod, txn.Type, txn.ID).
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
		mock.ExpectExec("DELETE FROM transactionrows WHERE id = \\$1").
			WithArgs(validUUID).
			WillReturnError(errors.New("delete failed"))

		err := repo.DeleteTransaction(validUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM transactionrows WHERE id = \\$1").
			WithArgs(validUUID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.DeleteTransaction(validUUID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
