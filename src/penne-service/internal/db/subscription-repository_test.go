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

func TestPgSubscriptionRepo_CreateSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgSubscriptionRepo(db)
	userUUID := uuid.New()
	subUUID := uuid.New()
	envUUID := uuid.New()

	t.Run("Validation Errors", func(t *testing.T) {
		// Missing user UUID
		_, err := repo.CreateSubscription(&core.Subscription{Name: "Netflix", AmountE5: 1000}, nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected user UUID error, got %v", err)
		}

		// Missing name
		_, err = repo.CreateSubscription(&core.Subscription{UserUUID: userUUID, Name: "", AmountE5: 1000}, nil)
		if err == nil || err.Error() != "name is required" {
			t.Errorf("expected name error, got %v", err)
		}

		// Amount <= 0
		_, err = repo.CreateSubscription(&core.Subscription{UserUUID: userUUID, Name: "Netflix", AmountE5: 0}, nil)
		if err == nil || err.Error() != "amount must be greater than zero" {
			t.Errorf("expected amount error, got %v", err)
		}

		// Invalid billing cycle
		_, err = repo.CreateSubscription(&core.Subscription{UserUUID: userUUID, Name: "Netflix", AmountE5: 1000, BillingCycle: "invalid"}, nil)
		if err == nil || err.Error() != "invalid billing cycle" {
			t.Errorf("expected invalid billing cycle error, got %v", err)
		}

		// Invalid status
		_, err = repo.CreateSubscription(&core.Subscription{UserUUID: userUUID, Name: "Netflix", AmountE5: 1000, Status: "invalid"}, nil)
		if err == nil || err.Error() != "invalid status" {
			t.Errorf("expected invalid status error, got %v", err)
		}
	})

	t.Run("Success without Tx (Defaults applied)", func(t *testing.T) {
		sub := &core.Subscription{
			UserUUID:   userUUID,
			EnvelopeID: &envUUID,
			Name:       "Netflix",
			AmountE5:   64900000,
		}

		mock.ExpectQuery("INSERT INTO subscriptions").
			WithArgs(
				sub.UserUUID, sub.EnvelopeID, sub.Name, sub.AmountE5, core.BillingCycleMonthly,
				sqlmock.AnyArg(), "bank_card", core.SubscriptionStatusActive, false, "",
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 48, 0,
				sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(subUUID))

		id, err := repo.CreateSubscription(sub, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != subUUID {
			t.Errorf("expected id %v, got %v", subUUID, id)
		}
	})

	t.Run("Success with Tx and DB error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO subscriptions").
			WillReturnError(errors.New("db error"))

		tx, err := db.Begin()
		if err != nil {
			t.Fatalf("unexpected error beginning tx: %v", err)
		}

		sub := &core.Subscription{
			UserUUID:     userUUID,
			Name:         "Spotify",
			AmountE5:     11900000,
			BillingCycle: core.BillingCycleMonthly,
			Status:       core.SubscriptionStatusActive,
		}

		_, err = repo.CreateSubscription(sub, tx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPgSubscriptionRepo_GetSubscriptionByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgSubscriptionRepo(db)
	userUUID := uuid.New()
	subUUID := uuid.New()
	now := time.Now()

	t.Run("Missing ID", func(t *testing.T) {
		_, err := repo.GetSubscriptionByID(uuid.Nil)
		if err == nil || err.Error() != "subscription ID is required" {
			t.Errorf("expected error, got %v", err)
		}
	})

	t.Run("Success with valid notes", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_uuid", "envelope_id", "name", "amount_e5", "billing_cycle",
			"next_billing_date", "payment_method", "status", "auto_renew", "notes",
			"last_charged_at", "last_transaction_id", "merchant_pattern", "charge_window_hours", "occurrence_count",
			"created_at", "updated_at",
		}).AddRow(
			subUUID, userUUID, nil, "Netflix", int64(64900000), "monthly",
			now, "bank_card", "active", true, "Premium plan",
			nil, nil, "NETFLIX", 48, 0,
			now, now,
		)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE id = \\$1").
			WithArgs(subUUID).
			WillReturnRows(rows)

		sub, err := repo.GetSubscriptionByID(subUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sub.Name != "Netflix" || sub.Notes != "Premium plan" || sub.MerchantPattern != "NETFLIX" {
			t.Errorf("unexpected subscription fields: %+v", sub)
		}
	})

	t.Run("Not found error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE id = \\$1").
			WithArgs(subUUID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetSubscriptionByID(subUUID)
		if err != sql.ErrNoRows {
			t.Errorf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestPgSubscriptionRepo_GetSubscriptionsByUserUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgSubscriptionRepo(db)
	userUUID := uuid.New()
	subUUID := uuid.New()
	now := time.Now()

	t.Run("Missing User UUID", func(t *testing.T) {
		_, err := repo.GetSubscriptionsByUserUUID(uuid.Nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected error, got %v", err)
		}
	})

	t.Run("Success with rows", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_uuid", "envelope_id", "name", "amount_e5", "billing_cycle",
			"next_billing_date", "payment_method", "status", "auto_renew", "notes",
			"last_charged_at", "last_transaction_id", "merchant_pattern", "charge_window_hours", "occurrence_count",
			"created_at", "updated_at",
		}).
			AddRow(subUUID, userUUID, nil, "Netflix", int64(64900000), "monthly", now, "bank_card", "active", true, "Note 1", nil, nil, "NETFLIX", 48, 0, now, now).
			AddRow(uuid.New(), userUUID, nil, "Spotify", int64(11900000), "monthly", now, "bank_card", "active", true, nil, nil, nil, nil, 48, 0, now, now)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE user_uuid = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		subs, err := repo.GetSubscriptionsByUserUUID(userUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(subs) != 2 {
			t.Errorf("expected 2 subscriptions, got %d", len(subs))
		}
	})

	t.Run("Query error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE user_uuid = \\$1").
			WithArgs(userUUID).
			WillReturnError(errors.New("db error"))

		_, err := repo.GetSubscriptionsByUserUUID(userUUID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("Scan error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id"}).AddRow("not-a-uuid")
		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE user_uuid = \\$1").
			WithArgs(userUUID).
			WillReturnRows(rows)

		_, err := repo.GetSubscriptionsByUserUUID(userUUID)
		if err == nil {
			t.Fatal("expected scan error, got nil")
		}
	})
}

func TestPgSubscriptionRepo_GetDueSubscriptions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgSubscriptionRepo(db)
	userUUID := uuid.New()
	subUUID := uuid.New()
	now := time.Now()

	t.Run("Success without Tx", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "user_uuid", "envelope_id", "name", "amount_e5", "billing_cycle",
			"next_billing_date", "payment_method", "status", "auto_renew", "notes",
			"last_charged_at", "last_transaction_id", "merchant_pattern", "charge_window_hours", "occurrence_count",
			"created_at", "updated_at",
		}).AddRow(subUUID, userUUID, nil, "Netflix", int64(64900000), "monthly", now, "bank_card", "active", true, "Due Notes", nil, nil, "NETFLIX", 48, 0, now, now)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE status = 'active' AND auto_renew = true AND next_billing_date <= \\$1").
			WithArgs(now).
			WillReturnRows(rows)

		subs, err := repo.GetDueSubscriptions(now, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(subs) != 1 {
			t.Errorf("expected 1 due sub, got %d", len(subs))
		}
	})

	t.Run("Success with Tx", func(t *testing.T) {
		mock.ExpectBegin()
		rows := sqlmock.NewRows([]string{
			"id", "user_uuid", "envelope_id", "name", "amount_e5", "billing_cycle",
			"next_billing_date", "payment_method", "status", "auto_renew", "notes",
			"last_charged_at", "last_transaction_id", "merchant_pattern", "charge_window_hours", "occurrence_count",
			"created_at", "updated_at",
		}).AddRow(subUUID, userUUID, nil, "Netflix", int64(64900000), "monthly", now, "bank_card", "active", true, nil, nil, nil, nil, 48, 0, now, now)

		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE status = 'active' AND auto_renew = true AND next_billing_date <= \\$1").
			WithArgs(now).
			WillReturnRows(rows)

		tx, _ := db.Begin()
		subs, err := repo.GetDueSubscriptions(now, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(subs) != 1 {
			t.Errorf("expected 1 due sub, got %d", len(subs))
		}
	})

	t.Run("Query error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE status = 'active' AND auto_renew = true AND next_billing_date <= \\$1").
			WithArgs(now).
			WillReturnError(errors.New("db error"))

		_, err := repo.GetDueSubscriptions(now, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("Scan error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id"}).AddRow("not-a-uuid")
		mock.ExpectQuery("SELECT (.+) FROM subscriptions WHERE status = 'active' AND auto_renew = true AND next_billing_date <= \\$1").
			WithArgs(now).
			WillReturnRows(rows)

		_, err := repo.GetDueSubscriptions(now, nil)
		if err == nil {
			t.Fatal("expected scan error, got nil")
		}
	})
}

func TestPgSubscriptionRepo_UpdateSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgSubscriptionRepo(db)
	subUUID := uuid.New()

	t.Run("Validation Errors", func(t *testing.T) {
		// Missing ID
		err := repo.UpdateSubscription(&core.Subscription{Name: "Netflix", AmountE5: 100}, nil)
		if err == nil || err.Error() != "subscription ID is required" {
			t.Errorf("expected id error, got %v", err)
		}

		// Missing Name
		err = repo.UpdateSubscription(&core.Subscription{ID: subUUID, Name: "", AmountE5: 100}, nil)
		if err == nil || err.Error() != "name is required" {
			t.Errorf("expected name error, got %v", err)
		}

		// Amount <= 0
		err = repo.UpdateSubscription(&core.Subscription{ID: subUUID, Name: "Netflix", AmountE5: 0}, nil)
		if err == nil || err.Error() != "amount must be greater than zero" {
			t.Errorf("expected amount error, got %v", err)
		}

		// Invalid billing cycle
		err = repo.UpdateSubscription(&core.Subscription{ID: subUUID, Name: "Netflix", AmountE5: 1000, BillingCycle: "invalid"}, nil)
		if err == nil || err.Error() != "invalid billing cycle" {
			t.Errorf("expected invalid billing cycle error, got %v", err)
		}

		// Invalid status
		err = repo.UpdateSubscription(&core.Subscription{ID: subUUID, Name: "Netflix", AmountE5: 1000, Status: "invalid"}, nil)
		if err == nil || err.Error() != "invalid status" {
			t.Errorf("expected invalid status error, got %v", err)
		}
	})

	t.Run("Success without Tx", func(t *testing.T) {
		sub := &core.Subscription{
			ID:              subUUID,
			Name:            "Netflix Premium",
			AmountE5:        79900000,
			BillingCycle:    "monthly",
			NextBillingDate: time.Now(),
			PaymentMethod:   "bank_card",
			Status:          "active",
			AutoRenew:       true,
			Notes:           "Updated note",
		}

		mock.ExpectExec("UPDATE subscriptions SET").
			WithArgs(
				sub.EnvelopeID, sub.Name, sub.AmountE5, sub.BillingCycle,
				sub.NextBillingDate, sub.PaymentMethod, sub.Status, sub.AutoRenew,
				sub.Notes, sub.LastChargedAt, sub.LastTransactionID,
				sub.MerchantPattern, sub.ChargeWindowHours, sub.OccurrenceCount,
				sqlmock.AnyArg(), sub.ID,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateSubscription(sub, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Success with Tx", func(t *testing.T) {
		mock.ExpectBegin()
		sub := &core.Subscription{
			ID:              subUUID,
			Name:            "Netflix Premium",
			AmountE5:        79900000,
			BillingCycle:    "monthly",
			NextBillingDate: time.Now(),
			PaymentMethod:   "bank_card",
			Status:          "active",
			AutoRenew:       true,
			Notes:           "Updated note",
		}

		mock.ExpectExec("UPDATE subscriptions SET").
			WithArgs(
				sub.EnvelopeID, sub.Name, sub.AmountE5, sub.BillingCycle,
				sub.NextBillingDate, sub.PaymentMethod, sub.Status, sub.AutoRenew,
				sub.Notes, sub.LastChargedAt, sub.LastTransactionID,
				sub.MerchantPattern, sub.ChargeWindowHours, sub.OccurrenceCount,
				sqlmock.AnyArg(), sub.ID,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		tx, _ := db.Begin()
		err := repo.UpdateSubscription(sub, tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Not found", func(t *testing.T) {
		sub := &core.Subscription{
			ID:       subUUID,
			Name:     "Netflix",
			AmountE5: 500,
		}

		mock.ExpectExec("UPDATE subscriptions SET").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.UpdateSubscription(sub, nil)
		if err == nil || err.Error() != "subscription not found" {
			t.Errorf("expected not found error, got %v", err)
		}
	})

	t.Run("DB error", func(t *testing.T) {
		sub := &core.Subscription{
			ID:       subUUID,
			Name:     "Netflix",
			AmountE5: 500,
		}

		mock.ExpectExec("UPDATE subscriptions SET").
			WillReturnError(errors.New("db error"))

		err := repo.UpdateSubscription(sub, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPgSubscriptionRepo_DeleteSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPgSubscriptionRepo(db)
	subUUID := uuid.New()

	t.Run("Missing ID", func(t *testing.T) {
		err := repo.DeleteSubscription(uuid.Nil)
		if err == nil || err.Error() != "subscription ID is required" {
			t.Errorf("expected id error, got %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM subscriptions WHERE id = \\$1").
			WithArgs(subUUID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.DeleteSubscription(subUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Not found", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM subscriptions WHERE id = \\$1").
			WithArgs(subUUID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.DeleteSubscription(subUUID)
		if err == nil || err.Error() != "subscription not found" {
			t.Errorf("expected not found error, got %v", err)
		}
	})

	t.Run("DB error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM subscriptions WHERE id = \\$1").
			WithArgs(subUUID).
			WillReturnError(errors.New("db error"))

		err := repo.DeleteSubscription(subUUID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
