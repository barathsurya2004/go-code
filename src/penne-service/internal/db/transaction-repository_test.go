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
			WithArgs(txn.UserID, txn.EnvelopeID, txn.AmountE5, txn.CountryISO, txn.PaymentMethod, txn.Type, sqlmock.AnyArg(), txn.ShortcutIntentID, txn.Description, txn.WishlistItemID).
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
			WithArgs(txn.UserID, txn.EnvelopeID, txn.AmountE5, txn.CountryISO, txn.PaymentMethod, txn.Type, sqlmock.AnyArg(), txn.ShortcutIntentID, txn.Description, txn.WishlistItemID).
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
			WithArgs(txn.UserID, txn.EnvelopeID, txn.AmountE5, txn.CountryISO, txn.PaymentMethod, txn.Type, sqlmock.AnyArg(), txn.ShortcutIntentID, txn.Description, txn.WishlistItemID).
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
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "shortcut_intent_id", "description", "wishlist_item_id"}).
			AddRow(validUUID, userUUID, nil, int64(500), "US", "Chase", "debit", now, nil, "Coffee", nil)

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
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "description", "wishlist_item_id"}).
			AddRow(txn1UUID, userUUID, nil, int64(100), "US", "Chase", "debit", now, "Item 1", nil).
			AddRow(txn2UUID, userUUID, nil, int64(200), "US", "Citi", "credit", now, "Item 2", nil)

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
		err := repo.UpdateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction UUID is required" {
			t.Errorf("expected empty UUID error, got %v", err)
		}
	})

	t.Run("Negative Amount", func(t *testing.T) {
		txn := &core.Transaction{ID: validUUID, AmountE5: -5}
		err := repo.UpdateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction amount cannot be negative" {
			t.Errorf("expected negative amount error, got %v", err)
		}
	})

	t.Run("Empty CountryISO", func(t *testing.T) {
		txn := &core.Transaction{ID: validUUID, AmountE5: 5, CountryISO: ""}
		err := repo.UpdateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction country ISO cannot be empty" {
			t.Errorf("expected empty country ISO error, got %v", err)
		}
	})

	t.Run("Empty Type", func(t *testing.T) {
		txn := &core.Transaction{ID: validUUID, AmountE5: 5, CountryISO: "US", Type: ""}
		err := repo.UpdateTransaction(txn, nil)
		if err == nil || err.Error() != "transaction type cannot be empty" {
			t.Errorf("expected empty type error, got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		txn := &core.Transaction{ID: validUUID, AmountE5: 5, CountryISO: "US", PaymentMethod: "Chase", Type: "debit"}
		mock.ExpectExec("UPDATE transactionrows").
			WithArgs(txn.EnvelopeID, txn.AmountE5, txn.CountryISO, txn.PaymentMethod, txn.Type, txn.ShortcutIntentID, txn.Description, txn.WishlistItemID, txn.ID).
			WillReturnError(errors.New("update failed"))

		err := repo.UpdateTransaction(txn, nil)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		txn := &core.Transaction{ID: validUUID, AmountE5: 5, CountryISO: "US", PaymentMethod: "Chase", Type: "debit"}
		mock.ExpectExec("UPDATE transactionrows").
			WithArgs(txn.EnvelopeID, txn.AmountE5, txn.CountryISO, txn.PaymentMethod, txn.Type, txn.ShortcutIntentID, txn.Description, txn.WishlistItemID, txn.ID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateTransaction(txn, nil)
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

func TestPgTransactionRowsRepo_GetTransactionByTime(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)
	now := time.Now()

	t.Run("Zero Time Range", func(t *testing.T) {
		_, err := repo.GetTransactionByTime(time.Time{}, now, nil)
		if err == nil || err.Error() != "time range is required" {
			t.Errorf("expected time range required error, got %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		txnID := uuid.New()
		userID := uuid.New()
		t1 := now.Add(-time.Hour)
		t2 := now

		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "shortcut_intent_id"}).
			AddRow(txnID, userID, nil, 500, "US", "Card", "debit", now, nil)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, shortcut_intent_id FROM transactionrows").
			WithArgs(t1, t2).
			WillReturnRows(rows)

		txn, err := repo.GetTransactionByTime(t1, t2, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if txn == nil || txn.ID != txnID {
			t.Errorf("expected transaction %v, got %v", txnID, txn)
		}
	})

	t.Run("Success With Tx", func(t *testing.T) {
		txnID := uuid.New()
		userID := uuid.New()
		t1 := now.Add(-time.Hour)
		t2 := now

		mock.ExpectBegin()
		tx, _ := db.Begin()

		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "shortcut_intent_id"}).
			AddRow(txnID, userID, nil, 500, "US", "Card", "debit", now, nil)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, shortcut_intent_id FROM transactionrows").
			WithArgs(t1, t2).
			WillReturnRows(rows)

		txn, err := repo.GetTransactionByTime(t1, t2, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if txn == nil || txn.ID != txnID {
			t.Errorf("expected transaction %v, got %v", txnID, txn)
		}
	})
}

func TestPgTransactionRowsRepo_GetDashboardSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)

	t.Run("Empty User UUID", func(t *testing.T) {
		_, err := repo.GetDashboardSummary(uuid.Nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID is required error, got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		userUUID := uuid.New()
		mock.ExpectQuery("SELECT").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("query error"))

		_, err := repo.GetDashboardSummary(userUUID)
		if err == nil {
			t.Errorf("expected query error, got nil")
		}
	})

	t.Run("ErrNoRows", func(t *testing.T) {
		userUUID := uuid.New()
		mock.ExpectQuery("SELECT").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		summary, err := repo.GetDashboardSummary(userUUID)
		if err != nil {
			t.Errorf("expected no error on ErrNoRows, got %v", err)
		}
		if summary == nil || summary.TotalIncomeE5 != 0 || summary.TotalExpenseE5 != 0 {
			t.Errorf("expected empty summary, got %+v", summary)
		}
	})

	t.Run("Zero Values (No Transactions Found via COALESCE)", func(t *testing.T) {
		userUUID := uuid.New()
		rows := sqlmock.NewRows([]string{"prev_income_e5", "curr_income_e5", "total_expense_e5", "card_spent_e5", "bank_spent_e5", "fallback_budget_e5"}).
			AddRow(0, 0, 0, 0, 0, 0)

		mock.ExpectQuery("SELECT").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		summary, err := repo.GetDashboardSummary(userUUID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if summary == nil || summary.TotalIncomeE5 != 0 || summary.TotalExpenseE5 != 0 || summary.CardSpentE5 != 0 || summary.BankSpentE5 != 0 {
			t.Errorf("unexpected summary result: %+v", summary)
		}
	})

	t.Run("Standard Case: Last Month Income Powers Current Month Budget", func(t *testing.T) {
		userUUID := uuid.New()
		// prev_income: 10000, curr_income (paycheck on 25th): 12000, total_expense: 4000, card: 3000, bank: 1000, fallback: 0
		rows := sqlmock.NewRows([]string{"prev_income_e5", "curr_income_e5", "total_expense_e5", "card_spent_e5", "bank_spent_e5", "fallback_budget_e5"}).
			AddRow(10000, 12000, 4000, 3000, 1000, 0)

		mock.ExpectQuery("SELECT").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		summary, err := repo.GetDashboardSummary(userUUID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if summary == nil {
			t.Fatalf("expected summary, got nil")
		}
		if summary.BaseIncomeE5 != 10000 {
			t.Errorf("expected BaseIncomeE5 10000, got %d", summary.BaseIncomeE5)
		}
		if summary.BufferedIncomeE5 != 12000 {
			t.Errorf("expected BufferedIncomeE5 12000, got %d", summary.BufferedIncomeE5)
		}
		if summary.BufferedUsedE5 != 0 {
			t.Errorf("expected BufferedUsedE5 0, got %d", summary.BufferedUsedE5)
		}
		if summary.BufferedRemainingE5 != 12000 {
			t.Errorf("expected BufferedRemainingE5 12000, got %d", summary.BufferedRemainingE5)
		}
		if summary.TotalExpenseE5 != 4000 {
			t.Errorf("expected TotalExpenseE5 4000, got %d", summary.TotalExpenseE5)
		}
		if summary.TotalRemainingE5 != 6000 {
			t.Errorf("expected TotalRemainingE5 6000, got %d", summary.TotalRemainingE5)
		}
		if summary.TotalIncomeE5 != 10000 {
			t.Errorf("expected TotalIncomeE5 10000, got %d", summary.TotalIncomeE5)
		}
	})

	t.Run("Edge Case: Expenses Exceed Last Month Income But Covered by Buffered Salary", func(t *testing.T) {
		userUUID := uuid.New()
		// prev_income: 10000, curr_income: 12000, total_expense: 11000 (1000 over prev_income), card: 8000, bank: 3000
		rows := sqlmock.NewRows([]string{"prev_income_e5", "curr_income_e5", "total_expense_e5", "card_spent_e5", "bank_spent_e5", "fallback_budget_e5"}).
			AddRow(10000, 12000, 11000, 8000, 3000, 0)

		mock.ExpectQuery("SELECT").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		summary, err := repo.GetDashboardSummary(userUUID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if summary.BaseIncomeE5 != 10000 {
			t.Errorf("expected BaseIncomeE5 10000, got %d", summary.BaseIncomeE5)
		}
		if summary.BufferedUsedE5 != 1000 {
			t.Errorf("expected BufferedUsedE5 1000, got %d", summary.BufferedUsedE5)
		}
		if summary.BufferedRemainingE5 != 11000 {
			t.Errorf("expected BufferedRemainingE5 11000, got %d", summary.BufferedRemainingE5)
		}
		if summary.TotalRemainingE5 != 0 {
			t.Errorf("expected TotalRemainingE5 0, got %d", summary.TotalRemainingE5)
		}
		if summary.TotalIncomeE5 != 11000 {
			t.Errorf("expected TotalIncomeE5 11000, got %d", summary.TotalIncomeE5)
		}
	})

	t.Run("Edge Case: Catastrophic Overspend Exceeding Buffered Salary", func(t *testing.T) {
		userUUID := uuid.New()
		// prev_income: 10000, curr_income: 2000, total_expense: 15000, deficit: 5000, buffer covers 2000, uncovered: 3000
		rows := sqlmock.NewRows([]string{"prev_income_e5", "curr_income_e5", "total_expense_e5", "card_spent_e5", "bank_spent_e5", "fallback_budget_e5"}).
			AddRow(10000, 2000, 15000, 10000, 5000, 0)

		mock.ExpectQuery("SELECT").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		summary, err := repo.GetDashboardSummary(userUUID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if summary.BufferedUsedE5 != 2000 {
			t.Errorf("expected BufferedUsedE5 2000, got %d", summary.BufferedUsedE5)
		}
		if summary.BufferedRemainingE5 != 0 {
			t.Errorf("expected BufferedRemainingE5 0, got %d", summary.BufferedRemainingE5)
		}
		if summary.TotalRemainingE5 != -3000 {
			t.Errorf("expected TotalRemainingE5 -3000, got %d", summary.TotalRemainingE5)
		}
		if summary.TotalIncomeE5 != 12000 {
			t.Errorf("expected TotalIncomeE5 12000, got %d", summary.TotalIncomeE5)
		}
	})

	t.Run("Edge Case: New User Bootstrapping with Fallback Budget", func(t *testing.T) {
		userUUID := uuid.New()
		// prev_income: 0, curr_income: 0, total_expense: 2000, card: 1500, bank: 500, fallback_budget: 8000
		rows := sqlmock.NewRows([]string{"prev_income_e5", "curr_income_e5", "total_expense_e5", "card_spent_e5", "bank_spent_e5", "fallback_budget_e5"}).
			AddRow(0, 0, 2000, 1500, 500, 8000)

		mock.ExpectQuery("SELECT").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		summary, err := repo.GetDashboardSummary(userUUID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if summary.BaseIncomeE5 != 8000 {
			t.Errorf("expected BaseIncomeE5 8000, got %d", summary.BaseIncomeE5)
		}
		if summary.TotalRemainingE5 != 6000 {
			t.Errorf("expected TotalRemainingE5 6000, got %d", summary.TotalRemainingE5)
		}
		if summary.TotalIncomeE5 != 8000 {
			t.Errorf("expected TotalIncomeE5 8000, got %d", summary.TotalIncomeE5)
		}
	})
}

func TestPgTransactionRowsRepo_GetTransactionByUserUUIDPaginated(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)
	now := time.Now()

	t.Run("Empty User UUID", func(t *testing.T) {
		_, err := repo.GetTransactionByUserUUIDPaginated(uuid.Nil, time.Time{}, uuid.Nil, 10)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID is required error, got %v", err)
		}
	})

	t.Run("Invalid Limit", func(t *testing.T) {
		_, err := repo.GetTransactionByUserUUIDPaginated(uuid.New(), time.Time{}, uuid.Nil, 0)
		if err == nil || err.Error() != "limit must be positive" {
			t.Errorf("expected limit must be positive error, got %v", err)
		}
	})

	t.Run("First Page (Nil Cursor)", func(t *testing.T) {
		userUUID := uuid.New()
		txnID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "description", "wishlist_item_id"}).
			AddRow(txnID, userUUID, nil, 1000, "US", "Card", "debit", now, "Item", nil)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE\\(description, ''\\), wishlist_item_id FROM transactionrows").
			WithArgs(userUUID, nil, nil, 10).
			WillReturnRows(rows)

		txns, err := repo.GetTransactionByUserUUIDPaginated(userUUID, time.Time{}, uuid.Nil, 10)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(txns) != 1 || txns[0].ID != txnID {
			t.Errorf("expected 1 transaction with ID %v, got %v", txnID, txns)
		}
	})

	t.Run("Subsequent Page (With Cursor)", func(t *testing.T) {
		userUUID := uuid.New()
		lastTxnID := uuid.New()
		lastTime := now.Add(-time.Hour)
		newTxnID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "description", "wishlist_item_id"}).
			AddRow(newTxnID, userUUID, nil, 2000, "US", "Card", "debit", lastTime, "Item 2", nil)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE\\(description, ''\\), wishlist_item_id FROM transactionrows").
			WithArgs(userUUID, lastTime, lastTxnID, 10).
			WillReturnRows(rows)

		txns, err := repo.GetTransactionByUserUUIDPaginated(userUUID, lastTime, lastTxnID, 10)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(txns) != 1 || txns[0].ID != newTxnID {
			t.Errorf("expected 1 transaction with ID %v, got %v", newTxnID, txns)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		userUUID := uuid.New()
		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE\\(description, ''\\), wishlist_item_id FROM transactionrows").
			WithArgs(userUUID, nil, nil, 10).
			WillReturnError(errors.New("db error"))

		_, err := repo.GetTransactionByUserUUIDPaginated(userUUID, time.Time{}, uuid.Nil, 10)
		if err == nil {
			t.Errorf("expected query error, got nil")
		}
	})
}

func TestPgTransactionRowsRepo_GetTransactionByAmountAndTime(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)
	userUUID := uuid.New()
	now := time.Now()
	t1 := now.Add(-5 * time.Minute)
	t2 := now.Add(5 * time.Minute)

	t.Run("Empty User UUID", func(t *testing.T) {
		_, err := repo.GetTransactionByAmountAndTime(uuid.Nil, 500, t1, t2, nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID is required error, got %v", err)
		}
	})

	t.Run("Zero Amount", func(t *testing.T) {
		_, err := repo.GetTransactionByAmountAndTime(userUUID, 0, t1, t2, nil)
		if err == nil || err.Error() != "amount must be greater than zero" {
			t.Errorf("expected amount must be greater than zero error, got %v", err)
		}
	})

	t.Run("Negative Amount", func(t *testing.T) {
		_, err := repo.GetTransactionByAmountAndTime(userUUID, -100, t1, t2, nil)
		if err == nil || err.Error() != "amount must be greater than zero" {
			t.Errorf("expected amount must be greater than zero error, got %v", err)
		}
	})

	t.Run("Zero Time Range", func(t *testing.T) {
		_, err := repo.GetTransactionByAmountAndTime(userUUID, 500, time.Time{}, t2, nil)
		if err == nil || err.Error() != "time range is required" {
			t.Errorf("expected time range is required error, got %v", err)
		}
	})

	t.Run("Invalid Time Range Bounds", func(t *testing.T) {
		_, err := repo.GetTransactionByAmountAndTime(userUUID, 500, t2, t1, nil)
		if err == nil || err.Error() != "time lowerbound cannot be after time upperbound" {
			t.Errorf("expected lowerbound after upperbound error, got %v", err)
		}
	})

	t.Run("Success without Tx", func(t *testing.T) {
		txnID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "shortcut_intent_id"}).
			AddRow(txnID, userUUID, nil, int64(500), "US", "bank_account", "debit", now, nil)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, shortcut_intent_id FROM transactionrows").
			WithArgs(userUUID, int64(500), t1, t2).
			WillReturnRows(rows)

		txn, err := repo.GetTransactionByAmountAndTime(userUUID, 500, t1, t2, nil)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if txn == nil || txn.ID != txnID {
			t.Errorf("expected txn ID %v, got %v", txnID, txn)
		}
	})

	t.Run("Success with Tx", func(t *testing.T) {
		txnID := uuid.New()
		mock.ExpectBegin()
		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}

		rows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "shortcut_intent_id"}).
			AddRow(txnID, userUUID, nil, int64(500), "US", "bank_account", "debit", now, nil)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, shortcut_intent_id FROM transactionrows").
			WithArgs(userUUID, int64(500), t1, t2).
			WillReturnRows(rows)

		txn, err := repo.GetTransactionByAmountAndTime(userUUID, 500, t1, t2, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if txn == nil || txn.ID != txnID {
			t.Errorf("expected txn ID %v, got %v", txnID, txn)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, shortcut_intent_id FROM transactionrows").
			WithArgs(userUUID, int64(500), t1, t2).
			WillReturnError(errors.New("db error"))

		_, err := repo.GetTransactionByAmountAndTime(userUUID, 500, t1, t2, nil)
		if err == nil {
			t.Errorf("expected query error, got nil")
		}
	})
}

func TestPgTransactionRowsRepo_GetMonthlyInsights(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgTransactionRowsRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Validation Errors", func(t *testing.T) {
		_, err := repo.GetMonthlyInsights(uuid.Nil, 2026, 10)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID error, got %v", err)
		}

		_, err = repo.GetMonthlyInsights(userUUID, 1999, 10)
		if err == nil || err.Error() != "invalid year" {
			t.Errorf("expected invalid year error, got %v", err)
		}

		_, err = repo.GetMonthlyInsights(userUUID, 2101, 10)
		if err == nil || err.Error() != "invalid year" {
			t.Errorf("expected invalid year error, got %v", err)
		}

		_, err = repo.GetMonthlyInsights(userUUID, 2026, 0)
		if err == nil || err.Error() != "invalid month" {
			t.Errorf("expected invalid month error, got %v", err)
		}

		_, err = repo.GetMonthlyInsights(userUUID, 2026, 13)
		if err == nil || err.Error() != "invalid month" {
			t.Errorf("expected invalid month error, got %v", err)
		}
	})

	t.Run("Envelope Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT e.id, COALESCE\\(e.name, ''\\), COALESCE\\(eg.name, ''\\) FROM envelope e").
			WithArgs(userUUID).
			WillReturnError(errors.New("envelope query failed"))

		_, err := repo.GetMonthlyInsights(userUUID, 2026, 10)
		if err == nil || err.Error() != "envelope query failed" {
			t.Errorf("expected envelope query error, got %v", err)
		}
	})

	t.Run("Transaction Query Error", func(t *testing.T) {
		envRows := sqlmock.NewRows([]string{"id", "name", "group_name"})
		mock.ExpectQuery("SELECT e.id, COALESCE\\(e.name, ''\\), COALESCE\\(eg.name, ''\\) FROM envelope e").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		subRows := sqlmock.NewRows([]string{"last_transaction_id", "name", "merchant_pattern"})
		mock.ExpectQuery("SELECT last_transaction_id, name, COALESCE\\(merchant_pattern, ''\\) FROM subscriptions").
			WithArgs(userUUID).
			WillReturnRows(subRows)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE\\(description, ''\\), wishlist_item_id FROM transactionrows").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("txn query failed"))

		_, err := repo.GetMonthlyInsights(userUUID, 2026, 10)
		if err == nil || err.Error() != "txn query failed" {
			t.Errorf("expected txn query error, got %v", err)
		}
	})

	t.Run("Empty Month Success", func(t *testing.T) {
		envRows := sqlmock.NewRows([]string{"id", "name", "group_name"})
		mock.ExpectQuery("SELECT e.id, COALESCE\\(e.name, ''\\), COALESCE\\(eg.name, ''\\) FROM envelope e").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		subRows := sqlmock.NewRows([]string{"last_transaction_id", "name", "merchant_pattern"})
		mock.ExpectQuery("SELECT last_transaction_id, name, COALESCE\\(merchant_pattern, ''\\) FROM subscriptions").
			WithArgs(userUUID).
			WillReturnRows(subRows)

		txnRows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "description", "wishlist_item_id"})
		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE\\(description, ''\\), wishlist_item_id FROM transactionrows").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(txnRows)

		prevRows := sqlmock.NewRows([]string{"sum"}).AddRow(int64(0))
		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount_e5\\), 0\\) FROM transactionrows").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(prevRows)

		report, err := repo.GetMonthlyInsights(userUUID, 2026, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.TotalIncomeE5 != 0 || report.TotalExpenseE5 != 0 {
			t.Errorf("expected 0 totals, got income %d, expense %d", report.TotalIncomeE5, report.TotalExpenseE5)
		}
		if report.DaysInMonth != 31 {
			t.Errorf("expected 31 days, got %d", report.DaysInMonth)
		}
		if len(report.DailyHeatmap) != 31 {
			t.Errorf("expected 31 heatmap days, got %d", len(report.DailyHeatmap))
		}
	})

	t.Run("Full Month with Subscriptions and Discretionary", func(t *testing.T) {
		envID := uuid.New()
		envRows := sqlmock.NewRows([]string{"id", "name", "group_name"}).
			AddRow(envID, "Groceries", "Food")
		mock.ExpectQuery("SELECT e.id, COALESCE\\(e.name, ''\\), COALESCE\\(eg.name, ''\\) FROM envelope e").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		subTxnID := uuid.New()
		subRows := sqlmock.NewRows([]string{"last_transaction_id", "name", "merchant_pattern"}).
			AddRow(&subTxnID, "Netflix", "netflix.com")
		mock.ExpectQuery("SELECT last_transaction_id, name, COALESCE\\(merchant_pattern, ''\\) FROM subscriptions").
			WithArgs(userUUID).
			WillReturnRows(subRows)

		t1 := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
		t2 := time.Date(2026, 10, 5, 15, 0, 0, 0, time.UTC)
		t3 := time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)
		t4 := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)

		txnRows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "description", "wishlist_item_id"}).
			AddRow(uuid.New(), userUUID, nil, int64(10000000), "IN", "bank_account", "credit", t1, "Salary", nil).
			AddRow(subTxnID, userUUID, nil, int64(50000), "IN", "bank_card", "debit", t2, "Subscription: Netflix Standard", nil).
			AddRow(uuid.New(), userUUID, &envID, int64(150000), "IN", "upi", "debit", t3, "Supermarket shopping", nil).
			AddRow(uuid.New(), userUUID, nil, int64(250000), "IN", "bank_card", "debit", t4, "Fancy Electronics", nil)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE\\(description, ''\\), wishlist_item_id FROM transactionrows").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(txnRows)

		prevRows := sqlmock.NewRows([]string{"sum"}).AddRow(int64(400000))
		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount_e5\\), 0\\) FROM transactionrows").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(prevRows)

		report, err := repo.GetMonthlyInsights(userUUID, 2026, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.TotalIncomeE5 != 10000000 {
			t.Errorf("expected income 10000000, got %d", report.TotalIncomeE5)
		}
		if report.TotalExpenseE5 != 450000 {
			t.Errorf("expected expense 450000, got %d", report.TotalExpenseE5)
		}
		if report.SubscriptionExpenseE5 != 50000 {
			t.Errorf("expected sub expense 50000, got %d", report.SubscriptionExpenseE5)
		}
		if report.DiscretionaryExpenseE5 != 400000 {
			t.Errorf("expected discretionary expense 400000, got %d", report.DiscretionaryExpenseE5)
		}
		if report.PeakDay == nil || report.PeakDay.Date != "2026-10-15" {
			t.Errorf("expected peak day 2026-10-15, got %v", report.PeakDay)
		}
		if report.LargestTransaction == nil || report.LargestTransaction.AmountE5 != 250000 {
			t.Errorf("expected largest txn 250000, got %v", report.LargestTransaction)
		}
		if report.PreviousMonth == nil || report.PreviousMonth.TotalExpenseE5 != 400000 {
			t.Errorf("expected previous month 400000, got %v", report.PreviousMonth)
		}
		if len(report.CategorySplits) != 2 {
			t.Errorf("expected 2 category splits, got %d", len(report.CategorySplits))
		}
	})

	t.Run("Current Month And Pattern Matches", func(t *testing.T) {
		now := time.Now().UTC()
		envRows := sqlmock.NewRows([]string{"id", "name", "group_name"})
		mock.ExpectQuery("SELECT e.id, COALESCE\\(e.name, ''\\), COALESCE\\(eg.name, ''\\) FROM envelope e").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		subRows := sqlmock.NewRows([]string{"last_transaction_id", "name", "merchant_pattern"}).
			AddRow(nil, "Spotify", "spotify.com")
		mock.ExpectQuery("SELECT last_transaction_id, name, COALESCE\\(merchant_pattern, ''\\) FROM subscriptions").
			WithArgs(userUUID).
			WillReturnRows(subRows)

		tNow := time.Date(now.Year(), now.Month(), 1, 10, 0, 0, 0, time.UTC)
		txnRows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "description", "wishlist_item_id"}).
			AddRow(uuid.New(), userUUID, nil, int64(11900), "IN", "card", "debit", tNow, "spotify premium student", nil)
		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE\\(description, ''\\), wishlist_item_id FROM transactionrows").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(txnRows)

		prevRows := sqlmock.NewRows([]string{"sum"}).AddRow(int64(0))
		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount_e5\\), 0\\) FROM transactionrows").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(prevRows)

		report, err := repo.GetMonthlyInsights(userUUID, now.Year(), int(now.Month()))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.SubscriptionExpenseE5 != 11900 {
			t.Errorf("expected 11900 sub expense, got %d", report.SubscriptionExpenseE5)
		}
	})

	t.Run("Txn Row Scan Error", func(t *testing.T) {
		envRows := sqlmock.NewRows([]string{"id", "name", "group_name"})
		mock.ExpectQuery("SELECT e.id, COALESCE\\(e.name, ''\\), COALESCE\\(eg.name, ''\\) FROM envelope e").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		subRows := sqlmock.NewRows([]string{"last_transaction_id", "name", "merchant_pattern"})
		mock.ExpectQuery("SELECT last_transaction_id, name, COALESCE\\(merchant_pattern, ''\\) FROM subscriptions").
			WithArgs(userUUID).
			WillReturnRows(subRows)

		txnRows := sqlmock.NewRows([]string{"id"}).AddRow("invalid")
		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE\\(description, ''\\), wishlist_item_id FROM transactionrows").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(txnRows)

		_, err := repo.GetMonthlyInsights(userUUID, 2026, 10)
		if err == nil {
			t.Errorf("expected scan error, got nil")
		}
	})

	t.Run("Subscription Query Error and All Intensity Levels", func(t *testing.T) {
		emptyEnvID := uuid.New()
		envRows := sqlmock.NewRows([]string{"id", "name", "group_name"}).
			AddRow(emptyEnvID, "", "") // empty name covers Custom Category fallback
		mock.ExpectQuery("SELECT e.id, COALESCE\\(e.name, ''\\), COALESCE\\(eg.name, ''\\) FROM envelope e").
			WithArgs(userUUID).
			WillReturnRows(envRows)

		// Subscription query returns error
		mock.ExpectQuery("SELECT last_transaction_id, name, COALESCE\\(merchant_pattern, ''\\) FROM subscriptions").
			WithArgs(userUUID).
			WillReturnError(errors.New("sub db error"))

		d1 := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
		d2 := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
		d3 := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
		d4 := time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)

		// max will be 1000000
		// d1: 100000 (ratio 0.1 <= 0.2 -> level 1)
		// d2: 400000 (ratio 0.4 <= 0.45 -> level 2)
		// d3: 700000 (ratio 0.7 <= 0.75 -> level 3)
		// d4: 1000000 (ratio 1.0 > 0.75 -> level 4)
		txnRows := sqlmock.NewRows([]string{"id", "user_id", "envelope_id", "amount_e5", "country_iso2", "payment_method", "txn_type", "created_at", "description", "wishlist_item_id"}).
			AddRow(uuid.New(), userUUID, &emptyEnvID, int64(100000), "IN", "custom_crypto", "debit", d1, "", nil).
			AddRow(uuid.New(), userUUID, &emptyEnvID, int64(400000), "IN", "upi", "debit", d2, "", nil).
			AddRow(uuid.New(), userUUID, &emptyEnvID, int64(700000), "IN", "bank_account", "debit", d3, "", nil).
			AddRow(uuid.New(), userUUID, &emptyEnvID, int64(1000000), "IN", "bank_card", "debit", d4, "", nil)

		mock.ExpectQuery("SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE\\(description, ''\\), wishlist_item_id FROM transactionrows").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(txnRows)

		prevRows := sqlmock.NewRows([]string{"sum"}).AddRow(int64(0))
		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(amount_e5\\), 0\\) FROM transactionrows").
			WithArgs(userUUID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(prevRows)

		report, err := repo.GetMonthlyInsights(userUUID, 2026, 9)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.TotalExpenseE5 != 2200000 {
			t.Errorf("expected 2200000, got %d", report.TotalExpenseE5)
		}
		if report.DailyHeatmap[0].IntensityLevel != 1 {
			t.Errorf("expected level 1, got %d", report.DailyHeatmap[0].IntensityLevel)
		}
		if report.DailyHeatmap[1].IntensityLevel != 2 {
			t.Errorf("expected level 2, got %d", report.DailyHeatmap[1].IntensityLevel)
		}
		if report.DailyHeatmap[2].IntensityLevel != 3 {
			t.Errorf("expected level 3, got %d", report.DailyHeatmap[2].IntensityLevel)
		}
		if report.DailyHeatmap[3].IntensityLevel != 4 {
			t.Errorf("expected level 4, got %d", report.DailyHeatmap[3].IntensityLevel)
		}
		if report.CategorySplits[0].EnvelopeName != "Custom Category" {
			t.Errorf("expected Custom Category, got %s", report.CategorySplits[0].EnvelopeName)
		}
	})
}


