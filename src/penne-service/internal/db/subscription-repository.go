package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
)

type pgSubscriptionRepo struct {
	db *sql.DB
}

func NewPgSubscriptionRepo(db *sql.DB) core.SubscriptionRepository {
	return &pgSubscriptionRepo{
		db: db,
	}
}

func (r *pgSubscriptionRepo) CreateSubscription(sub *core.Subscription, tx *sql.Tx) (uuid.UUID, error) {
	if sub.UserUUID == uuid.Nil {
		return uuid.Nil, errors.New("user UUID is required")
	}
	if sub.Name == "" {
		return uuid.Nil, errors.New("name is required")
	}
	if sub.AmountE5 <= 0 {
		return uuid.Nil, errors.New("amount must be greater than zero")
	}
	if sub.BillingCycle == "" {
		sub.BillingCycle = core.BillingCycleMonthly
	}
	switch sub.BillingCycle {
	case core.BillingCycleWeekly, core.BillingCycleMonthly, core.BillingCycleQuarterly, core.BillingCycleYearly:
	default:
		return uuid.Nil, errors.New("invalid billing cycle")
	}

	if sub.Status == "" {
		sub.Status = core.SubscriptionStatusActive
	}
	switch sub.Status {
	case core.SubscriptionStatusActive, core.SubscriptionStatusPaused, core.SubscriptionStatusCancelled:
	default:
		return uuid.Nil, errors.New("invalid status")
	}
	if sub.PaymentMethod == "" {
		sub.PaymentMethod = "bank_card"
	}
	if sub.NextBillingDate.IsZero() {
		sub.NextBillingDate = utils.NowUTC().AddDate(0, 1, 0)
	}

	if sub.ChargeWindowHours <= 0 {
		sub.ChargeWindowHours = 48
	}

	now := utils.NowUTC()
	sub.CreatedAt = now
	sub.UpdatedAt = now

	query := `
		INSERT INTO subscriptions (
			user_uuid, envelope_id, name, amount_e5, billing_cycle,
			next_billing_date, payment_method, status, auto_renew, notes,
			last_charged_at, last_transaction_id, merchant_pattern,
			charge_window_hours, occurrence_count,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`

	var id uuid.UUID
	var err error
	if tx != nil {
		err = tx.QueryRowContext(context.Background(), query,
			sub.UserUUID, sub.EnvelopeID, sub.Name, sub.AmountE5, sub.BillingCycle,
			sub.NextBillingDate, sub.PaymentMethod, sub.Status, sub.AutoRenew, sub.Notes,
			sub.LastChargedAt, sub.LastTransactionID, sub.MerchantPattern,
			sub.ChargeWindowHours, sub.OccurrenceCount,
			sub.CreatedAt, sub.UpdatedAt,
		).Scan(&id)
	} else {
		err = r.db.QueryRowContext(context.Background(), query,
			sub.UserUUID, sub.EnvelopeID, sub.Name, sub.AmountE5, sub.BillingCycle,
			sub.NextBillingDate, sub.PaymentMethod, sub.Status, sub.AutoRenew, sub.Notes,
			sub.LastChargedAt, sub.LastTransactionID, sub.MerchantPattern,
			sub.ChargeWindowHours, sub.OccurrenceCount,
			sub.CreatedAt, sub.UpdatedAt,
		).Scan(&id)
	}

	if err != nil {
		return uuid.Nil, err
	}
	sub.ID = id
	return id, nil
}

func (r *pgSubscriptionRepo) GetSubscriptionByID(id uuid.UUID) (*core.Subscription, error) {
	if id == uuid.Nil {
		return nil, errors.New("subscription ID is required")
	}

	query := `
		SELECT id, user_uuid, envelope_id, name, amount_e5, billing_cycle,
		       next_billing_date, payment_method, status, auto_renew, notes,
		       last_charged_at, last_transaction_id, merchant_pattern,
		       charge_window_hours, occurrence_count,
		       created_at, updated_at
		FROM subscriptions
		WHERE id = $1
	`

	sub := &core.Subscription{}
	var notes sql.NullString
	var merchantPattern sql.NullString
	err := r.db.QueryRowContext(context.Background(), query, id).Scan(
		&sub.ID, &sub.UserUUID, &sub.EnvelopeID, &sub.Name, &sub.AmountE5, &sub.BillingCycle,
		&sub.NextBillingDate, &sub.PaymentMethod, &sub.Status, &sub.AutoRenew, &notes,
		&sub.LastChargedAt, &sub.LastTransactionID, &merchantPattern,
		&sub.ChargeWindowHours, &sub.OccurrenceCount,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if notes.Valid {
		sub.Notes = notes.String
	}
	if merchantPattern.Valid {
		sub.MerchantPattern = merchantPattern.String
	}
	return sub, nil
}

func (r *pgSubscriptionRepo) GetSubscriptionsByUserUUID(userUUID uuid.UUID) ([]*core.Subscription, error) {
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}

	query := `
		SELECT id, user_uuid, envelope_id, name, amount_e5, billing_cycle,
		       next_billing_date, payment_method, status, auto_renew, notes,
		       last_charged_at, last_transaction_id, merchant_pattern,
		       charge_window_hours, occurrence_count,
		       created_at, updated_at
		FROM subscriptions
		WHERE user_uuid = $1
		ORDER BY next_billing_date ASC, created_at DESC
	`

	rows, err := r.db.QueryContext(context.Background(), query, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*core.Subscription
	for rows.Next() {
		sub := &core.Subscription{}
		var notes sql.NullString
		var merchantPattern sql.NullString
		if err := rows.Scan(
			&sub.ID, &sub.UserUUID, &sub.EnvelopeID, &sub.Name, &sub.AmountE5, &sub.BillingCycle,
			&sub.NextBillingDate, &sub.PaymentMethod, &sub.Status, &sub.AutoRenew, &notes,
			&sub.LastChargedAt, &sub.LastTransactionID, &merchantPattern,
			&sub.ChargeWindowHours, &sub.OccurrenceCount,
			&sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if notes.Valid {
			sub.Notes = notes.String
		}
		if merchantPattern.Valid {
			sub.MerchantPattern = merchantPattern.String
		}
		subscriptions = append(subscriptions, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (r *pgSubscriptionRepo) GetDueSubscriptions(asOf time.Time, tx *sql.Tx) ([]*core.Subscription, error) {
	query := `
		SELECT id, user_uuid, envelope_id, name, amount_e5, billing_cycle,
		       next_billing_date, payment_method, status, auto_renew, notes,
		       last_charged_at, last_transaction_id, merchant_pattern,
		       charge_window_hours, occurrence_count,
		       created_at, updated_at
		FROM subscriptions
		WHERE status = 'active' AND auto_renew = true AND next_billing_date <= $1
		ORDER BY next_billing_date ASC
	`

	var rows *sql.Rows
	var err error
	if tx != nil {
		rows, err = tx.QueryContext(context.Background(), query, asOf)
	} else {
		rows, err = r.db.QueryContext(context.Background(), query, asOf)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []*core.Subscription
	for rows.Next() {
		sub := &core.Subscription{}
		var notes sql.NullString
		var merchantPattern sql.NullString
		if err := rows.Scan(
			&sub.ID, &sub.UserUUID, &sub.EnvelopeID, &sub.Name, &sub.AmountE5, &sub.BillingCycle,
			&sub.NextBillingDate, &sub.PaymentMethod, &sub.Status, &sub.AutoRenew, &notes,
			&sub.LastChargedAt, &sub.LastTransactionID, &merchantPattern,
			&sub.ChargeWindowHours, &sub.OccurrenceCount,
			&sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if notes.Valid {
			sub.Notes = notes.String
		}
		if merchantPattern.Valid {
			sub.MerchantPattern = merchantPattern.String
		}
		subscriptions = append(subscriptions, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (r *pgSubscriptionRepo) UpdateSubscription(sub *core.Subscription, tx *sql.Tx) error {
	if sub.ID == uuid.Nil {
		return errors.New("subscription ID is required")
	}
	if sub.Name == "" {
		return errors.New("name is required")
	}
	if sub.AmountE5 <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if sub.BillingCycle == "" {
		sub.BillingCycle = core.BillingCycleMonthly
	}
	switch sub.BillingCycle {
	case core.BillingCycleWeekly, core.BillingCycleMonthly, core.BillingCycleQuarterly, core.BillingCycleYearly:
	default:
		return errors.New("invalid billing cycle")
	}
	if sub.Status == "" {
		sub.Status = core.SubscriptionStatusActive
	}
	switch sub.Status {
	case core.SubscriptionStatusActive, core.SubscriptionStatusPaused, core.SubscriptionStatusCancelled:
	default:
		return errors.New("invalid status")
	}

	sub.UpdatedAt = utils.NowUTC()

	query := `
		UPDATE subscriptions
		SET envelope_id = $1, name = $2, amount_e5 = $3, billing_cycle = $4,
		    next_billing_date = $5, payment_method = $6, status = $7, auto_renew = $8,
		    notes = $9, last_charged_at = $10, last_transaction_id = $11,
		    merchant_pattern = $12, charge_window_hours = $13, occurrence_count = $14,
		    updated_at = $15
		WHERE id = $16
	`

	var res sql.Result
	var err error
	if tx != nil {
		res, err = tx.ExecContext(context.Background(), query,
			sub.EnvelopeID, sub.Name, sub.AmountE5, sub.BillingCycle,
			sub.NextBillingDate, sub.PaymentMethod, sub.Status, sub.AutoRenew,
			sub.Notes, sub.LastChargedAt, sub.LastTransactionID,
			sub.MerchantPattern, sub.ChargeWindowHours, sub.OccurrenceCount,
			sub.UpdatedAt, sub.ID,
		)
	} else {
		res, err = r.db.ExecContext(context.Background(), query,
			sub.EnvelopeID, sub.Name, sub.AmountE5, sub.BillingCycle,
			sub.NextBillingDate, sub.PaymentMethod, sub.Status, sub.AutoRenew,
			sub.Notes, sub.LastChargedAt, sub.LastTransactionID,
			sub.MerchantPattern, sub.ChargeWindowHours, sub.OccurrenceCount,
			sub.UpdatedAt, sub.ID,
		)
	}

	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("subscription not found")
	}
	return nil
}

func (r *pgSubscriptionRepo) DeleteSubscription(id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("subscription ID is required")
	}

	query := `DELETE FROM subscriptions WHERE id = $1`
	res, err := r.db.ExecContext(context.Background(), query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("subscription not found")
	}
	return nil
}
