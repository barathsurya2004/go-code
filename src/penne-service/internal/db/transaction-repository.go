package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
)

type pgTransactionRowsRepo struct {
	db *sql.DB
}

func NewPgTransactionRowsRepo(db *sql.DB) core.TransactionRepository {
	return &pgTransactionRowsRepo{
		db: db,
	}
}

func (r *pgTransactionRowsRepo) CreateTransaction(txn *core.Transaction, Tx *sql.Tx) (uuid.UUID, error) {
	query := `
		INSERT INTO transactionrows (user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type,created_at,shortcut_intent_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)RETURNING id
	`

	// validation checks
	if txn.AmountE5 < 0 {
		return uuid.Nil, errors.New("transaction amount cannot be negative")
	}
	if txn.CountryISO == "" {
		return uuid.Nil, errors.New("transaction country ISO cannot be empty")
	}
	if txn.Type == "" {
		return uuid.Nil, errors.New("transaction type cannot be empty")
	}
	if txn.CreatedAt.IsZero() {
		txn.CreatedAt = utils.NowUTC()
	} else {
		txn.CreatedAt = txn.CreatedAt.UTC()
	}
	var txnID uuid.UUID
	var row *sql.Row
	if Tx != nil {
		row = Tx.QueryRow(query,
			txn.UserID,
			txn.EnvelopeID,
			txn.AmountE5,
			txn.CountryISO,
			txn.PaymentMethod,
			txn.Type,
			txn.CreatedAt,
			txn.ShortcutIntentID,
		)
	} else {
		row = r.db.QueryRow(query,
			txn.UserID,
			txn.EnvelopeID,
			txn.AmountE5,
			txn.CountryISO,
			txn.PaymentMethod,
			txn.Type,
			txn.CreatedAt,
			txn.ShortcutIntentID,
		)
	}
	if err := row.Scan(&txnID); err != nil {
		return uuid.Nil, err
	}
	txn.ID = txnID
	return txnID, nil
}

func (r *pgTransactionRowsRepo) GetTransactionByUUID(id uuid.UUID) (*core.Transaction, error) {
	query := `
		SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, shortcut_intent_id
		FROM transactionrows
		WHERE id = $1
	`
	// validation checks
	if id == uuid.Nil {
		return nil, errors.New("transaction UUID is required")
	}

	txn := &core.Transaction{}
	err := r.db.QueryRowContext(context.Background(), query, id).Scan(
		&txn.ID,
		&txn.UserID,
		&txn.EnvelopeID,
		&txn.AmountE5,
		&txn.CountryISO,
		&txn.PaymentMethod,
		&txn.Type,
		&txn.CreatedAt,
		&txn.ShortcutIntentID,
	)
	if err != nil {
		return nil, err
	}
	return txn, nil
}

func (r *pgTransactionRowsRepo) GetTransactionsByUserUUID(userID uuid.UUID) ([]*core.Transaction, error) {
	query := `
		SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at
		FROM transactionrows
		WHERE user_id = $1
	`
	// validation checks
	if userID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}

	rows, err := r.db.QueryContext(context.Background(), query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*core.Transaction
	for rows.Next() {
		txn := &core.Transaction{}
		if err := rows.Scan(
			&txn.ID,
			&txn.UserID,
			&txn.EnvelopeID,
			&txn.AmountE5,
			&txn.CountryISO,
			&txn.PaymentMethod,
			&txn.Type,
			&txn.CreatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, txn)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *pgTransactionRowsRepo) GetTransactionByUserUUIDPaginated(userID uuid.UUID, lastTransactionCreatedAt time.Time, lastTransactionID uuid.UUID, limit int) ([]*core.Transaction, error) {
	if userID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}
	if limit <= 0 {
		return nil, errors.New("limit must be positive")
	}

	query := `
		SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at
		FROM transactionrows
		WHERE user_id = $1
		  AND ($2::timestamptz IS NULL OR $3::uuid IS NULL OR (created_at, id) < ($2, $3))
		ORDER BY created_at DESC, id DESC
		LIMIT $4;
	`

	var createdAtArg interface{} = lastTransactionCreatedAt
	if lastTransactionCreatedAt.IsZero() {
		createdAtArg = nil
	}

	var idArg interface{} = lastTransactionID
	if lastTransactionID == uuid.Nil {
		idArg = nil
	}

	rows, err := r.db.QueryContext(context.Background(), query, userID, createdAtArg, idArg, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var transactions []*core.Transaction
	for rows.Next() {
		txn := &core.Transaction{}
		if err := rows.Scan(
			&txn.ID,
			&txn.UserID,
			&txn.EnvelopeID,
			&txn.AmountE5,
			&txn.CountryISO,
			&txn.PaymentMethod,
			&txn.Type,
			&txn.CreatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, txn)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *pgTransactionRowsRepo) UpdateTransaction(txn *core.Transaction, Tx *sql.Tx) error {
	query := `
		UPDATE transactionrows
		SET envelope_id = $1, amount_e5 = $2, country_iso2 = $3, payment_method = $4, txn_type = $5, shortcut_intent_id = $6
		WHERE id = $7
	`

	// validation checks
	if txn.ID == uuid.Nil {
		return errors.New("transaction UUID is required")
	}
	if txn.AmountE5 < 0 {
		return errors.New("transaction amount cannot be negative")
	}
	if txn.CountryISO == "" {
		return errors.New("transaction country ISO cannot be empty")
	}
	if txn.Type == "" {
		return errors.New("transaction type cannot be empty")
	}
	var err error
	if Tx != nil {
		_, err = Tx.ExecContext(context.Background(), query,
			txn.EnvelopeID,
			txn.AmountE5,
			txn.CountryISO,
			txn.PaymentMethod,
			txn.Type,
			txn.ShortcutIntentID,
			txn.ID,
		)
	} else {
		_, err = r.db.ExecContext(context.Background(), query,
			txn.EnvelopeID,
			txn.AmountE5,
			txn.CountryISO,
			txn.PaymentMethod,
			txn.Type,
			txn.ShortcutIntentID,
			txn.ID,
		)
	}
	return err
}

func (r *pgTransactionRowsRepo) DeleteTransaction(id uuid.UUID) error {
	query := `
		DELETE FROM transactionrows
		WHERE id = $1
	`
	// validation checks
	if id == uuid.Nil {
		return errors.New("transaction UUID is required")
	}

	_, err := r.db.ExecContext(context.Background(), query, id)
	return err
}

func (r *pgTransactionRowsRepo) GetTransactionByTime(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
	// validation checks
	if time_lowerbound.IsZero() || time_upperbound.IsZero() {
		return nil, errors.New("time range is required")
	}
	query := `
		SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, shortcut_intent_id
		FROM transactionrows
		WHERE created_at BETWEEN $1 AND $2 AND shortcut_intent_id IS NULL
		ORDER BY created_at DESC LIMIT 1
	`
	txn := &core.Transaction{}
	var row *sql.Row
	if Tx != nil {
		row = Tx.QueryRowContext(context.Background(), query, time_lowerbound, time_upperbound)
	} else {
		row = r.db.QueryRowContext(context.Background(), query, time_lowerbound, time_upperbound)
	}
	err := row.Scan(
		&txn.ID,
		&txn.UserID,
		&txn.EnvelopeID,
		&txn.AmountE5,
		&txn.CountryISO,
		&txn.PaymentMethod,
		&txn.Type,
		&txn.CreatedAt,
		&txn.ShortcutIntentID,
	)
	if err != nil {
		return nil, err
	}
	return txn, nil
}

func (r *pgTransactionRowsRepo) GetTransactionByAmountAndTime(userUUID uuid.UUID, amountE5 int64, time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*core.Transaction, error) {
	// validation checks
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}
	if amountE5 <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if time_lowerbound.IsZero() || time_upperbound.IsZero() {
		return nil, errors.New("time range is required")
	}
	if time_lowerbound.After(time_upperbound) {
		return nil, errors.New("time lowerbound cannot be after time upperbound")
	}
	query := `
		SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, shortcut_intent_id
		FROM transactionrows
		WHERE user_id = $1 AND amount_e5 = $2 AND created_at BETWEEN $3 AND $4
		ORDER BY created_at DESC LIMIT 1
	`
	txn := &core.Transaction{}
	var row *sql.Row
	if Tx != nil {
		row = Tx.QueryRowContext(context.Background(), query, userUUID, amountE5, time_lowerbound, time_upperbound)
	} else {
		row = r.db.QueryRowContext(context.Background(), query, userUUID, amountE5, time_lowerbound, time_upperbound)
	}
	err := row.Scan(
		&txn.ID,
		&txn.UserID,
		&txn.EnvelopeID,
		&txn.AmountE5,
		&txn.CountryISO,
		&txn.PaymentMethod,
		&txn.Type,
		&txn.CreatedAt,
		&txn.ShortcutIntentID,
	)
	if err != nil {
		return nil, err
	}
	return txn, nil
}

func (r *pgTransactionRowsRepo) GetDashboardSummary(userUUID uuid.UUID) (*core.DashboardSummary, error) {
	// validation checks
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}

	now := utils.NowUTC()
	prevStart, prevEnd, err := utils.GetPreviousCadenceStartAndEndTime(core.MonthlyCadence, now)
	if err != nil {
		return nil, err
	}
	currStart, currEnd, err := utils.GetCadenceStartAndEndTime(core.MonthlyCadence, now)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT 
			COALESCE((
				SELECT SUM(amount_e5)
				FROM transactionrows
				WHERE user_id = $1 AND txn_type = 'credit' AND created_at BETWEEN $2 AND $3
			), 0) as prev_income_e5,
			COALESCE((
				SELECT SUM(amount_e5)
				FROM transactionrows
				WHERE user_id = $1 AND txn_type = 'credit' AND created_at BETWEEN $4 AND $5
			), 0) as curr_income_e5,
			COALESCE((
				SELECT SUM(amount_e5)
				FROM transactionrows
				WHERE user_id = $1 AND txn_type = 'debit' AND created_at BETWEEN $4 AND $5
			), 0) as total_expense_e5,
			COALESCE((
				SELECT SUM(amount_e5)
				FROM transactionrows
				WHERE user_id = $1 AND txn_type = 'debit' AND payment_method = 'bank_card' AND created_at BETWEEN $4 AND $5
			), 0) as card_spent_e5,
			COALESCE((
				SELECT SUM(amount_e5)
				FROM transactionrows
				WHERE user_id = $1 AND txn_type = 'debit' AND payment_method = 'bank_account' AND created_at BETWEEN $4 AND $5
			), 0) as bank_spent_e5,
			COALESCE((
				SELECT monthly_budget_e5
				FROM users
				WHERE uuid = $1
			), 0) as fallback_budget_e5
	`

	dashboardSummary := &core.DashboardSummary{}
	var prevIncomeE5, currIncomeE5, totalExpenseE5, cardSpentE5, bankSpentE5, fallbackBudgetE5 int64

	err = r.db.QueryRowContext(context.Background(), query, userUUID, prevStart, prevEnd, currStart, currEnd).Scan(
		&prevIncomeE5,
		&currIncomeE5,
		&totalExpenseE5,
		&cardSpentE5,
		&bankSpentE5,
		&fallbackBudgetE5,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return dashboardSummary, nil
		}
		return nil, err
	}

	baseIncomeE5 := prevIncomeE5
	if baseIncomeE5 == 0 && fallbackBudgetE5 > 0 {
		baseIncomeE5 = fallbackBudgetE5
	}

	bufferedIncomeE5 := currIncomeE5
	var bufferedUsedE5 int64 = 0
	var totalRemainingE5 int64 = 0

	if totalExpenseE5 <= baseIncomeE5 {
		totalRemainingE5 = baseIncomeE5 - totalExpenseE5
		bufferedUsedE5 = 0
	} else {
		deficit := totalExpenseE5 - baseIncomeE5
		if deficit <= bufferedIncomeE5 {
			bufferedUsedE5 = deficit
			totalRemainingE5 = 0
		} else {
			bufferedUsedE5 = bufferedIncomeE5
			uncoveredDeficit := deficit - bufferedIncomeE5
			totalRemainingE5 = -uncoveredDeficit
		}
	}

	bufferedRemainingE5 := bufferedIncomeE5 - bufferedUsedE5
	effectiveIncomeE5 := baseIncomeE5 + bufferedUsedE5

	dashboardSummary.TotalIncomeE5 = effectiveIncomeE5
	dashboardSummary.BaseIncomeE5 = baseIncomeE5
	dashboardSummary.BufferedIncomeE5 = bufferedIncomeE5
	dashboardSummary.BufferedUsedE5 = bufferedUsedE5
	dashboardSummary.BufferedRemainingE5 = bufferedRemainingE5
	dashboardSummary.TotalExpenseE5 = totalExpenseE5
	dashboardSummary.TotalRemainingE5 = totalRemainingE5
	dashboardSummary.CardSpentE5 = cardSpentE5
	dashboardSummary.BankSpentE5 = bankSpentE5

	return dashboardSummary, nil
}
