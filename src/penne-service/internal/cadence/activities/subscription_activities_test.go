package activities

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockSubscriptionRepoForActivity struct {
	core.SubscriptionRepository
	createFn  func(*core.Subscription, *sql.Tx) (uuid.UUID, error)
	getByIDFn func(uuid.UUID) (*core.Subscription, error)
	getByUser func(uuid.UUID) ([]*core.Subscription, error)
	getDueFn  func(time.Time, *sql.Tx) ([]*core.Subscription, error)
	updateFn  func(*core.Subscription, *sql.Tx) error
	deleteFn  func(uuid.UUID) error
}

func (m *mockSubscriptionRepoForActivity) CreateSubscription(sub *core.Subscription, tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(sub, tx)
	}
	return uuid.New(), nil
}

func (m *mockSubscriptionRepoForActivity) GetSubscriptionByID(id uuid.UUID) (*core.Subscription, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}

func (m *mockSubscriptionRepoForActivity) GetSubscriptionsByUserUUID(userUUID uuid.UUID) ([]*core.Subscription, error) {
	if m.getByUser != nil {
		return m.getByUser(userUUID)
	}
	return nil, nil
}

func (m *mockSubscriptionRepoForActivity) GetDueSubscriptions(asOf time.Time, tx *sql.Tx) ([]*core.Subscription, error) {
	if m.getDueFn != nil {
		return m.getDueFn(asOf, tx)
	}
	return nil, nil
}

func (m *mockSubscriptionRepoForActivity) UpdateSubscription(sub *core.Subscription, tx *sql.Tx) error {
	if m.updateFn != nil {
		return m.updateFn(sub, tx)
	}
	return nil
}

func (m *mockSubscriptionRepoForActivity) DeleteSubscription(id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

type mockTxnRepoForActivity struct {
	core.TransactionRepository
	createFn func(*core.Transaction, *sql.Tx) (uuid.UUID, error)
}

func (m *mockTxnRepoForActivity) CreateTransaction(txn *core.Transaction, tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(txn, tx)
	}
	return uuid.New(), nil
}

type mockAllocRepoForActivity struct {
	core.AllocationRepository
	updateSpentFn func(uuid.UUID, time.Time, int64, *sql.Tx) error
}

func (m *mockAllocRepoForActivity) UpdateSpentAmount(envelopeID uuid.UUID, targetDate time.Time, delta int64, tx *sql.Tx) error {
	if m.updateSpentFn != nil {
		return m.updateSpentFn(envelopeID, targetDate, delta, tx)
	}
	return nil
}

func TestSubscriptionActivities_CreateSubscriptionActivity(t *testing.T) {
	logger := zap.NewNop()

	t.Run("Nil subscription", func(t *testing.T) {
		acts := NewSubscriptionActivities(core.RepoContainer{}, logger)
		_, err := acts.CreateSubscriptionActivity(context.Background(), nil)
		if err == nil || err.Error() != "subscription cannot be nil" {
			t.Errorf("expected error, got %v", err)
		}
	})

	t.Run("Repo error", func(t *testing.T) {
		mockRepo := &mockSubscriptionRepoForActivity{
			createFn: func(s *core.Subscription, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("insert failed")
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockRepo}, logger)
		_, err := acts.CreateSubscriptionActivity(context.Background(), &core.Subscription{Name: "Netflix"})
		if err == nil || err.Error() != "insert failed" {
			t.Errorf("expected insert error, got %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		expectedID := uuid.New()
		mockRepo := &mockSubscriptionRepoForActivity{
			createFn: func(s *core.Subscription, tx *sql.Tx) (uuid.UUID, error) {
				return expectedID, nil
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockRepo}, logger)
		id, err := acts.CreateSubscriptionActivity(context.Background(), &core.Subscription{Name: "Netflix"})
		if err != nil || id != expectedID {
			t.Fatalf("unexpected result: id=%v, err=%v", id, err)
		}
	})
}

func TestSubscriptionActivities_GetSubscriptionByIDActivity(t *testing.T) {
	logger := zap.NewNop()

	t.Run("Nil ID", func(t *testing.T) {
		acts := NewSubscriptionActivities(core.RepoContainer{}, logger)
		_, err := acts.GetSubscriptionByIDActivity(context.Background(), uuid.Nil)
		if err == nil || err.Error() != "subscription ID cannot be nil" {
			t.Errorf("expected error, got %v", err)
		}
	})

	t.Run("Repo error", func(t *testing.T) {
		subID := uuid.New()
		mockRepo := &mockSubscriptionRepoForActivity{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return nil, errors.New("not found")
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockRepo}, logger)
		_, err := acts.GetSubscriptionByIDActivity(context.Background(), subID)
		if err == nil || err.Error() != "not found" {
			t.Errorf("expected not found, got %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		subID := uuid.New()
		mockRepo := &mockSubscriptionRepoForActivity{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{ID: subID, Name: "Spotify"}, nil
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockRepo}, logger)
		sub, err := acts.GetSubscriptionByIDActivity(context.Background(), subID)
		if err != nil || sub == nil || sub.Name != "Spotify" {
			t.Fatalf("unexpected result: sub=%+v, err=%v", sub, err)
		}
	})
}

func TestSubscriptionActivities_RenewSubscriptionActivity(t *testing.T) {
	logger := zap.NewNop()

	t.Run("Nil ID", func(t *testing.T) {
		acts := NewSubscriptionActivities(core.RepoContainer{}, logger)
		_, err := acts.RenewSubscriptionActivity(context.Background(), uuid.Nil)
		if err == nil || err.Error() != "subscription ID cannot be nil" {
			t.Errorf("expected error, got %v", err)
		}
	})

	t.Run("GetByID error", func(t *testing.T) {
		subID := uuid.New()
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return nil, errors.New("db error")
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockSubRepo}, logger)
		_, err := acts.RenewSubscriptionActivity(context.Background(), subID)
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("Inactive status", func(t *testing.T) {
		subID := uuid.New()
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{ID: subID, Status: core.SubscriptionStatusPaused}, nil
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockSubRepo}, logger)
		_, err := acts.RenewSubscriptionActivity(context.Background(), subID)
		if err == nil {
			t.Fatal("expected error for paused subscription")
		}
	})

	t.Run("CreateTransaction error", func(t *testing.T) {
		subID := uuid.New()
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{
					ID:       subID,
					Status:   core.SubscriptionStatusActive,
					Name:     "Netflix",
					AmountE5: 64900000,
				}, nil
			},
		}
		mockTxnRepo := &mockTxnRepoForActivity{
			createFn: func(t *core.Transaction, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("failed to insert txn")
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{
			Subscription: mockSubRepo,
			Transaction:  mockTxnRepo,
		}, logger)

		_, err := acts.RenewSubscriptionActivity(context.Background(), subID)
		if err == nil || err.Error() != "failed to insert txn" {
			t.Errorf("expected txn insert error, got %v", err)
		}
	})

	t.Run("UpdateSubscription error", func(t *testing.T) {
		subID := uuid.New()
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{
					ID:           subID,
					Status:       core.SubscriptionStatusActive,
					Name:         "Netflix",
					AmountE5:     64900000,
					BillingCycle: core.BillingCycleMonthly,
				}, nil
			},
			updateFn: func(s *core.Subscription, tx *sql.Tx) error {
				return errors.New("failed to update subscription")
			},
		}
		mockTxnRepo := &mockTxnRepoForActivity{
			createFn: func(t *core.Transaction, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.New(), nil
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{
			Subscription: mockSubRepo,
			Transaction:  mockTxnRepo,
		}, logger)

		_, err := acts.RenewSubscriptionActivity(context.Background(), subID)
		if err == nil || err.Error() != "failed to update subscription" {
			t.Errorf("expected update error, got %v", err)
		}
	})

	t.Run("Success with envelope allocation update", func(t *testing.T) {
		subID := uuid.New()
		envID := uuid.New()
		allocUpdated := false

		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{
					ID:           subID,
					EnvelopeID:   &envID,
					Status:       core.SubscriptionStatusActive,
					Name:         "Netflix",
					AmountE5:     64900000,
					BillingCycle: core.BillingCycleMonthly,
				}, nil
			},
			updateFn: func(s *core.Subscription, tx *sql.Tx) error {
				return nil
			},
		}
		mockTxnRepo := &mockTxnRepoForActivity{
			createFn: func(t *core.Transaction, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.New(), nil
			},
		}
		mockAllocRepo := &mockAllocRepoForActivity{
			updateSpentFn: func(eID uuid.UUID, td time.Time, d int64, tx *sql.Tx) error {
				allocUpdated = true
				return nil
			},
		}

		acts := NewSubscriptionActivities(core.RepoContainer{
			Subscription: mockSubRepo,
			Transaction:  mockTxnRepo,
			Allocation:   mockAllocRepo,
		}, logger)

		txn, err := acts.RenewSubscriptionActivity(context.Background(), subID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if txn == nil || txn.Description != "Subscription: Netflix" {
			t.Errorf("unexpected txn: %+v", txn)
		}
		if !allocUpdated {
			t.Error("expected allocation spent amount update")
		}
	})
}

func TestSubscriptionActivities_GetDueSubscriptionsActivity(t *testing.T) {
	logger := zap.NewNop()

	t.Run("Repo error", func(t *testing.T) {
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getDueFn: func(asOf time.Time, tx *sql.Tx) ([]*core.Subscription, error) {
				return nil, errors.New("db error")
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockSubRepo}, logger)
		_, err := acts.GetDueSubscriptionsActivity(context.Background(), time.Time{})
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		expected := []*core.Subscription{
			{Name: "Netflix"},
		}
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getDueFn: func(asOf time.Time, tx *sql.Tx) ([]*core.Subscription, error) {
				return expected, nil
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockSubRepo}, logger)
		subs, err := acts.GetDueSubscriptionsActivity(context.Background(), time.Now())
		if err != nil || len(subs) != 1 {
			t.Fatalf("unexpected result: subs=%+v, err=%v", subs, err)
		}
	})
}

func TestSubscriptionActivities_MatchSubscriptionIntentActivity(t *testing.T) {
	logger := zap.NewNop()
	userUUID := uuid.New()
	subID := uuid.New()
	now := time.Now()

	t.Run("Nil UserID", func(t *testing.T) {
		acts := NewSubscriptionActivities(core.RepoContainer{}, logger)
		_, err := acts.MatchSubscriptionIntentActivity(context.Background(), core.Transaction{UserID: uuid.Nil})
		if err == nil || err.Error() != "user ID cannot be nil" {
			t.Errorf("expected user ID error, got %v", err)
		}
	})

	t.Run("Repo error", func(t *testing.T) {
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByUser: func(u uuid.UUID) ([]*core.Subscription, error) {
				return nil, errors.New("db error")
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockSubRepo}, logger)
		_, err := acts.MatchSubscriptionIntentActivity(context.Background(), core.Transaction{UserID: userUUID})
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("Match found", func(t *testing.T) {
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByUser: func(u uuid.UUID) ([]*core.Subscription, error) {
				return []*core.Subscription{
					{
						ID:                subID,
						Name:              "Netflix",
						AmountE5:          64900000,
						Status:            core.SubscriptionStatusActive,
						NextBillingDate:   now,
						ChargeWindowHours: 48,
					},
				}, nil
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockSubRepo}, logger)
		matched, err := acts.MatchSubscriptionIntentActivity(context.Background(), core.Transaction{
			UserID:      userUUID,
			Type:        core.TxnTypeDebit,
			Description: "NETFLIX MUMBAI",
			AmountE5:    64900000,
			CreatedAt:   now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if matched == nil || matched.ID != subID {
			t.Errorf("expected match with subID, got %+v", matched)
		}
	})

	t.Run("No match found", func(t *testing.T) {
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByUser: func(u uuid.UUID) ([]*core.Subscription, error) {
				return []*core.Subscription{
					{
						ID:              subID,
						Name:            "Netflix",
						AmountE5:        64900000,
						Status:          core.SubscriptionStatusActive,
						NextBillingDate: now,
					},
				}, nil
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockSubRepo}, logger)
		matched, err := acts.MatchSubscriptionIntentActivity(context.Background(), core.Transaction{
			UserID:      userUUID,
			Type:        core.TxnTypeDebit,
			Description: "Uber Trip",
			AmountE5:    35000000,
			CreatedAt:   now,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if matched != nil {
			t.Errorf("expected nil match, got %+v", matched)
		}
	})
}

func TestSubscriptionActivities_RecordSubscriptionChargeActivity(t *testing.T) {
	logger := zap.NewNop()
	subID := uuid.New()
	txnID := uuid.New()
	now := time.Now()

	t.Run("Nil Subscription ID", func(t *testing.T) {
		acts := NewSubscriptionActivities(core.RepoContainer{}, logger)
		err := acts.RecordSubscriptionChargeActivity(context.Background(), uuid.Nil, txnID, now, "Netflix", 64900000)
		if err == nil || err.Error() != "subscription ID cannot be nil" {
			t.Errorf("expected subscription ID error, got %v", err)
		}
	})

	t.Run("GetByID error", func(t *testing.T) {
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return nil, errors.New("not found")
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockSubRepo}, logger)
		err := acts.RecordSubscriptionChargeActivity(context.Background(), subID, txnID, now, "Netflix", 64900000)
		if err == nil || err.Error() != "not found" {
			t.Errorf("expected not found error, got %v", err)
		}
	})

	t.Run("Update error", func(t *testing.T) {
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{
					ID:              subID,
					NextBillingDate: now,
					BillingCycle:    core.BillingCycleMonthly,
				}, nil
			},
			updateFn: func(s *core.Subscription, tx *sql.Tx) error {
				return errors.New("update failed")
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockSubRepo}, logger)
		err := acts.RecordSubscriptionChargeActivity(context.Background(), subID, txnID, now, "Netflix", 64900000)
		if err == nil || err.Error() != "update failed" {
			t.Errorf("expected update failed error, got %v", err)
		}
	})

	t.Run("Success with zero chargedAt", func(t *testing.T) {
		var updatedSub *core.Subscription
		mockSubRepo := &mockSubscriptionRepoForActivity{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{
					ID:              subID,
					NextBillingDate: now,
					BillingCycle:    core.BillingCycleMonthly,
					OccurrenceCount: 0,
				}, nil
			},
			updateFn: func(s *core.Subscription, tx *sql.Tx) error {
				updatedSub = s
				return nil
			},
		}
		acts := NewSubscriptionActivities(core.RepoContainer{Subscription: mockSubRepo}, logger)
		err := acts.RecordSubscriptionChargeActivity(context.Background(), subID, txnID, time.Time{}, "NETFLIX MUMBAI", 64900000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updatedSub == nil || updatedSub.OccurrenceCount != 1 || updatedSub.ChargeWindowHours != 24 {
			t.Errorf("unexpected updated subscription: %+v", updatedSub)
		}
	})
}

