package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
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
		INSERT INTO transactionrows (user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, shortcut_intent_id, description, wishlist_item_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id
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
			txn.Description,
			txn.WishlistItemID,
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
			txn.Description,
			txn.WishlistItemID,
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
		SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, shortcut_intent_id, COALESCE(description, ''), wishlist_item_id
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
		&txn.Description,
		&txn.WishlistItemID,
	)
	if err != nil {
		return nil, err
	}
	return txn, nil
}

func (r *pgTransactionRowsRepo) GetTransactionsByUserUUID(userID uuid.UUID) ([]*core.Transaction, error) {
	query := `
		SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE(description, ''), wishlist_item_id
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
			&txn.Description,
			&txn.WishlistItemID,
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
		SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE(description, ''), wishlist_item_id
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
			&txn.Description,
			&txn.WishlistItemID,
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
		SET envelope_id = $1, amount_e5 = $2, country_iso2 = $3, payment_method = $4, txn_type = $5, shortcut_intent_id = $6, description = $7, wishlist_item_id = $8
		WHERE id = $9
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
			txn.Description,
			txn.WishlistItemID,
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
			txn.Description,
			txn.WishlistItemID,
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

func (r *pgTransactionRowsRepo) GetMonthlyInsights(userUUID uuid.UUID, year int, month int) (*core.MonthlyInsightsReport, error) {
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}
	if year < 2000 || year > 2100 {
		return nil, errors.New("invalid year")
	}
	if month < 1 || month > 12 {
		return nil, errors.New("invalid month")
	}

	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	daysInMonth := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()

	now := utils.NowUTC()
	isCurrentMonth := now.Year() == year && int(now.Month()) == month
	daysElapsed := daysInMonth
	if isCurrentMonth {
		if now.Day() < daysInMonth {
			daysElapsed = now.Day()
		}
	}

	// 1. Fetch envelopes and groups for user
	type envInfo struct {
		name  string
		group string
	}
	envelopeMap := make(map[string]envInfo)
	envQuery := `
		SELECT e.id, COALESCE(e.name, ''), COALESCE(eg.name, '')
		FROM envelope e
		LEFT JOIN envelope_group eg ON e.envelope_group_id = eg.id
		WHERE e.user_uuid = $1
	`
	envRows, err := r.db.QueryContext(context.Background(), envQuery, userUUID)
	if err != nil {
		return nil, err
	}
	defer envRows.Close()
	for envRows.Next() {
		var envID uuid.UUID
		var envName, groupName string
		if err := envRows.Scan(&envID, &envName, &groupName); err == nil {
			envelopeMap[envID.String()] = envInfo{name: envName, group: groupName}
		}
	}

	// 2. Fetch subscriptions for user
	subTxnIDs := make(map[uuid.UUID]bool)
	var subNames []string
	subQuery := `
		SELECT last_transaction_id, name, COALESCE(merchant_pattern, '')
		FROM subscriptions
		WHERE user_uuid = $1
	`
	subRows, err := r.db.QueryContext(context.Background(), subQuery, userUUID)
	if err == nil {
		defer subRows.Close()
		for subRows.Next() {
			var lastTxnID *uuid.UUID
			var subName, pattern string
			if err := subRows.Scan(&lastTxnID, &subName, &pattern); err == nil {
				if lastTxnID != nil && *lastTxnID != uuid.Nil {
					subTxnIDs[*lastTxnID] = true
				}
				if trimmed := strings.TrimSpace(strings.ToLower(subName)); trimmed != "" {
					subNames = append(subNames, trimmed)
				}
				if trimmed := strings.TrimSpace(strings.ToLower(pattern)); trimmed != "" {
					subNames = append(subNames, trimmed)
				}
			}
		}
	}

	isSubTxn := func(t *core.Transaction) bool {
		if t.ID != uuid.Nil && subTxnIDs[t.ID] {
			return true
		}
		desc := strings.ToLower(t.Description)
		if strings.HasPrefix(desc, "subscription:") {
			return true
		}
		for _, name := range subNames {
			if name != "" && strings.Contains(desc, name) {
				return true
			}
		}
		return false
	}

	// 3. Fetch transactions for current month
	txnQuery := `
		SELECT id, user_id, envelope_id, amount_e5, country_iso2, payment_method, txn_type, created_at, COALESCE(description, ''), wishlist_item_id
		FROM transactionrows
		WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
		ORDER BY created_at ASC, id ASC
	`
	txnRows, err := r.db.QueryContext(context.Background(), txnQuery, userUUID, startOfMonth, endOfMonth)
	if err != nil {
		return nil, err
	}
	defer txnRows.Close()

	var transactions []*core.Transaction
	for txnRows.Next() {
		txn := &core.Transaction{}
		if err := txnRows.Scan(
			&txn.ID,
			&txn.UserID,
			&txn.EnvelopeID,
			&txn.AmountE5,
			&txn.CountryISO,
			&txn.PaymentMethod,
			&txn.Type,
			&txn.CreatedAt,
			&txn.Description,
			&txn.WishlistItemID,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, txn)
	}

	// 4. Fetch previous month debit sum
	prevStart := startOfMonth.AddDate(0, -1, 0)
	prevEnd := startOfMonth
	var prevExpenseE5 int64
	prevQuery := `
		SELECT COALESCE(SUM(amount_e5), 0)
		FROM transactionrows
		WHERE user_id = $1 AND txn_type = 'debit' AND created_at >= $2 AND created_at < $3
	`
	_ = r.db.QueryRowContext(context.Background(), prevQuery, userUUID, prevStart, prevEnd).Scan(&prevExpenseE5)

	// 5. Aggregate metrics
	var totalIncomeE5, totalExpenseE5, subscriptionExpenseE5, discretionaryExpenseE5 int64
	var subscriptionCount int

	type dailyBucket struct {
		dateKey string
		dayName string
		totalE5 int64
		txns    []*core.Transaction
	}
	dailyDiscretionaryMap := make(map[string]*dailyBucket)

	type categoryBucket struct {
		spentE5 int64
		count   int
	}
	categorySpendMap := make(map[string]*categoryBucket)

	type paymentBucket struct {
		spentE5 int64
		count   int
	}
	paymentMethodMap := make(map[string]*paymentBucket)

	var largestTxn *core.Transaction

	for _, t := range transactions {
		if t.Type == "credit" {
			totalIncomeE5 += t.AmountE5
		} else if t.Type == "debit" {
			totalExpenseE5 += t.AmountE5

			if isSubTxn(t) {
				subscriptionExpenseE5 += t.AmountE5
				subscriptionCount++
			} else {
				discretionaryExpenseE5 += t.AmountE5
				if largestTxn == nil || t.AmountE5 > largestTxn.AmountE5 {
					largestTxn = t
				}

				dateKey := t.CreatedAt.UTC().Format("2006-01-02")
				bucket, ok := dailyDiscretionaryMap[dateKey]
				if !ok {
					bucket = &dailyBucket{
						dateKey: dateKey,
						dayName: t.CreatedAt.UTC().Weekday().String(),
					}
					dailyDiscretionaryMap[dateKey] = bucket
				}
				bucket.totalE5 += t.AmountE5
				bucket.txns = append(bucket.txns, t)
			}

			// Category aggregation
			envIDStr := "unassigned"
			if t.EnvelopeID != nil && *t.EnvelopeID != uuid.Nil {
				envIDStr = t.EnvelopeID.String()
			}
			cBucket, ok := categorySpendMap[envIDStr]
			if !ok {
				cBucket = &categoryBucket{}
				categorySpendMap[envIDStr] = cBucket
			}
			cBucket.spentE5 += t.AmountE5
			cBucket.count++

			// Payment method aggregation
			pm := strings.ToLower(t.PaymentMethod)
			if pm == "" {
				pm = "bank_card"
			}
			pBucket, ok := paymentMethodMap[pm]
			if !ok {
				pBucket = &paymentBucket{}
				paymentMethodMap[pm] = pBucket
			}
			pBucket.spentE5 += t.AmountE5
			pBucket.count++
		}
	}

	resolveTxnDetails := func(t *core.Transaction) (desc, category, envName string) {
		if t.EnvelopeID != nil {
			if info, ok := envelopeMap[t.EnvelopeID.String()]; ok {
				envName = info.name
				category = info.group
			}
		}
		if category == "" {
			category = "General"
		}
		desc = strings.TrimSpace(t.Description)
		if desc == "" || strings.EqualFold(desc, "discretionary purchase") || strings.EqualFold(desc, "single expense") {
			if envName != "" {
				desc = envName
			} else if category != "" && category != "General" {
				desc = category
			} else {
				desc = "Discretionary Purchase"
			}
		}
		return desc, category, envName
	}

	// Peak spending day
	var peakDay *core.PeakSpendDayInfo
	var maxDailySpendE5 int64
	for _, bucket := range dailyDiscretionaryMap {
		if bucket.totalE5 > maxDailySpendE5 {
			maxDailySpendE5 = bucket.totalE5
			sortedTxns := make([]*core.Transaction, len(bucket.txns))
			copy(sortedTxns, bucket.txns)
			sort.Slice(sortedTxns, func(i, j int) bool {
				return sortedTxns[i].AmountE5 > sortedTxns[j].AmountE5
			})
			var topList []core.PeakSpendDayTransaction
			for i := 0; i < len(sortedTxns) && i < 3; i++ {
				t := sortedTxns[i]
				desc, cat, env := resolveTxnDetails(t)
				topList = append(topList, core.PeakSpendDayTransaction{
					ID:            t.ID.String(),
					Description:   desc,
					AmountE5:      t.AmountE5,
					PaymentMethod: t.PaymentMethod,
					Date:          t.CreatedAt.UTC().Format(time.RFC3339),
					Category:      cat,
					EnvelopeName:  env,
				})
			}
			peakDay = &core.PeakSpendDayInfo{
				Date:             bucket.dateKey,
				DayName:          bucket.dayName,
				TotalSpentE5:     bucket.totalE5,
				TransactionCount: len(bucket.txns),
				TopTransactions:  topList,
			}
		}
	}

	// No spend days count
	noSpendDaysCount := 0
	for day := 1; day <= daysElapsed; day++ {
		dateKey := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		if bucket, ok := dailyDiscretionaryMap[dateKey]; !ok || bucket.totalE5 == 0 {
			noSpendDaysCount++
		}
	}

	// Net savings and savings rate
	netSavingsE5 := totalIncomeE5 - totalExpenseE5
	var savingsRatePct float64
	if totalIncomeE5 > 0 {
		savingsRatePct = math.Max(0, math.Round(float64(netSavingsE5)/float64(totalIncomeE5)*1000.0)/10.0)
	}

	// Daily average
	var dailyAverageE5 int64
	if daysElapsed > 0 {
		dailyAverageE5 = discretionaryExpenseE5 / int64(daysElapsed)
	}

	// Category splits
	var categorySplits []core.CategorySpendSplit
	for envIDStr, bucket := range categorySpendMap {
		envName := "Unassigned Surplus"
		groupName := "General"
		if envIDStr != "unassigned" {
			if info, ok := envelopeMap[envIDStr]; ok {
				if info.name != "" {
					envName = info.name
				} else {
					envName = "Custom Category"
				}
				if info.group != "" {
					groupName = info.group
				}
			}
		}
		var pct float64
		if totalExpenseE5 > 0 {
			pct = math.Round(float64(bucket.spentE5)/float64(totalExpenseE5)*1000.0) / 10.0
		}
		categorySplits = append(categorySplits, core.CategorySpendSplit{
			EnvelopeID:       envIDStr,
			EnvelopeName:     envName,
			GroupName:        groupName,
			SpentE5:          bucket.spentE5,
			Percentage:       pct,
			TransactionCount: bucket.count,
		})
	}
	sort.Slice(categorySplits, func(i, j int) bool {
		return categorySplits[i].SpentE5 > categorySplits[j].SpentE5
	})

	// Payment method splits
	paymentMethodLabels := map[string]string{
		"bank_card":    "Debit / Credit Card",
		"upi":          "UPI AutoPay / QR",
		"bank_account": "Bank Auto-Debit / Netbanking",
		"back_account": "Bank Auto-Debit",
	}
	var pmSplits []core.PaymentMethodSplit
	for method, bucket := range paymentMethodMap {
		label, ok := paymentMethodLabels[method]
		if !ok {
			label = strings.ToUpper(method)
		}
		pmSplits = append(pmSplits, core.PaymentMethodSplit{
			Method:  method,
			Label:   label,
			SpentE5: bucket.spentE5,
			Count:   bucket.count,
		})
	}
	sort.Slice(pmSplits, func(i, j int) bool {
		return pmSplits[i].SpentE5 > pmSplits[j].SpentE5
	})

	// Previous month delta
	var prevMonthDelta *core.PreviousMonthInsightsDelta
	if prevExpenseE5 > 0 {
		deltaPct := math.Round(float64(totalExpenseE5-prevExpenseE5)/float64(prevExpenseE5)*1000.0) / 10.0
		prevMonthDelta = &core.PreviousMonthInsightsDelta{
			TotalExpenseE5: prevExpenseE5,
			DeltaPct:       deltaPct,
			IsLower:        totalExpenseE5 <= prevExpenseE5,
		}
	}

	// Heatmap items
	firstDayOffset := int(startOfMonth.Weekday())
	dailyHeatmap := make([]core.DailySpendingHeatmapItem, 0, daysInMonth)
	for day := 1; day <= daysInMonth; day++ {
		d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		dateKey := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		dayOfWeek := int(d.Weekday())
		dayName := d.Weekday().String()
		isFuture := isCurrentMonth && day > now.Day()

		var daySpentE5 int64
		var txnCount int
		var dayTxns []core.PeakSpendDayTransaction

		if bucket, ok := dailyDiscretionaryMap[dateKey]; ok {
			daySpentE5 = bucket.totalE5
			txnCount = len(bucket.txns)
			for _, t := range bucket.txns {
				desc, cat, env := resolveTxnDetails(t)
				dayTxns = append(dayTxns, core.PeakSpendDayTransaction{
					ID:            t.ID.String(),
					Description:   desc,
					AmountE5:      t.AmountE5,
					PaymentMethod: t.PaymentMethod,
					Date:          t.CreatedAt.UTC().Format(time.RFC3339),
					Category:      cat,
					EnvelopeName:  env,
				})
			}
		}

		intensity := 0
		if !isFuture && daySpentE5 > 0 {
			if maxDailySpendE5 > 0 {
				ratio := float64(daySpentE5) / float64(maxDailySpendE5)
				if ratio <= 0.2 {
					intensity = 1
				} else if ratio <= 0.45 {
					intensity = 2
				} else if ratio <= 0.75 {
					intensity = 3
				} else {
					intensity = 4
				}
			} else {
				intensity = 1
			}
		}

		dailyHeatmap = append(dailyHeatmap, core.DailySpendingHeatmapItem{
			Date:             dateKey,
			Day:              day,
			DayOfWeek:        dayOfWeek,
			DayName:          dayName,
			TotalSpentE5:     daySpentE5,
			TransactionCount: txnCount,
			IntensityLevel:   intensity,
			IsFuture:         isFuture,
			Transactions:     dayTxns,
		})
	}

	var largestTxnInfo *core.PeakSpendDayTransaction
	if largestTxn != nil {
		desc, cat, env := resolveTxnDetails(largestTxn)
		largestTxnInfo = &core.PeakSpendDayTransaction{
			ID:            largestTxn.ID.String(),
			Description:   desc,
			AmountE5:      largestTxn.AmountE5,
			PaymentMethod: largestTxn.PaymentMethod,
			Date:          largestTxn.CreatedAt.UTC().Format(time.RFC3339),
			Category:      cat,
			EnvelopeName:  env,
		}
	}

	return &core.MonthlyInsightsReport{
		Year:                   year,
		Month:                  month,
		MonthLabel:             fmt.Sprintf("%s %d", startOfMonth.Month().String(), year),
		DaysInMonth:            daysInMonth,
		DaysElapsed:            daysElapsed,
		TotalIncomeE5:          totalIncomeE5,
		TotalExpenseE5:         totalExpenseE5,
		NetSavingsE5:           netSavingsE5,
		SavingsRatePct:         savingsRatePct,
		SubscriptionExpenseE5:  subscriptionExpenseE5,
		DiscretionaryExpenseE5: discretionaryExpenseE5,
		SubscriptionCount:      subscriptionCount,
		PeakDay:                peakDay,
		DailyHeatmap:           dailyHeatmap,
		FirstDayOffset:         firstDayOffset,
		MaxDailySpendE5:        maxDailySpendE5,
		CategorySplits:         categorySplits,
		NoSpendDaysCount:       noSpendDaysCount,
		DailyAverageE5:         dailyAverageE5,
		LargestTransaction:     largestTxnInfo,
		PaymentMethodSplits:    pmSplits,
		PreviousMonth:          prevMonthDelta,
	}, nil
}

