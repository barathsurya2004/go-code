package activities

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SubscriptionActivities struct {
	repos  core.RepoContainer
	logger *zap.Logger
}

func NewSubscriptionActivities(repos core.RepoContainer, logger *zap.Logger) *SubscriptionActivities {
	return &SubscriptionActivities{
		repos:  repos,
		logger: logger,
	}
}

func (a *SubscriptionActivities) CreateSubscriptionActivity(ctx context.Context, sub *core.Subscription) (uuid.UUID, error) {
	if sub == nil {
		a.logger.Error("Subscription cannot be nil")
		return uuid.Nil, errors.New("subscription cannot be nil")
	}

	id, err := a.repos.Subscription.CreateSubscription(sub, nil)
	if err != nil {
		a.logger.Error("Failed to create subscription in activity", zap.Error(err))
		return uuid.Nil, err
	}

	a.logger.Info("Created subscription in Cadence activity", zap.String("id", id.String()))
	return id, nil
}

func (a *SubscriptionActivities) GetSubscriptionByIDActivity(ctx context.Context, id uuid.UUID) (*core.Subscription, error) {
	if id == uuid.Nil {
		a.logger.Error("Subscription ID cannot be nil")
		return nil, errors.New("subscription ID cannot be nil")
	}

	sub, err := a.repos.Subscription.GetSubscriptionByID(id)
	if err != nil {
		a.logger.Error("Failed to get subscription by ID in activity", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	return sub, nil
}

func (a *SubscriptionActivities) RenewSubscriptionActivity(ctx context.Context, id uuid.UUID) (*core.Transaction, error) {
	if id == uuid.Nil {
		a.logger.Error("Subscription ID cannot be nil")
		return nil, errors.New("subscription ID cannot be nil")
	}

	sub, err := a.repos.Subscription.GetSubscriptionByID(id)
	if err != nil {
		a.logger.Error("Failed to get subscription for renewal", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	if sub.Status != core.SubscriptionStatusActive {
		a.logger.Warn("Cannot renew inactive subscription", zap.String("id", id.String()), zap.String("status", sub.Status))
		return nil, fmt.Errorf("subscription is not active (status: %s)", sub.Status)
	}

	now := utils.NowUTC()
	txn := &core.Transaction{
		UserID:        sub.UserUUID,
		EnvelopeID:    sub.EnvelopeID,
		AmountE5:      sub.AmountE5,
		Type:          core.TxnTypeDebit,
		PaymentMethod: sub.PaymentMethod,
		CountryISO:    "IN",
		CreatedAt:     now,
		Description:   "Subscription: " + sub.Name,
	}

	txnID, err := a.repos.Transaction.CreateTransaction(txn, nil)
	if err != nil {
		a.logger.Error("Failed to post subscription debit transaction", zap.Error(err), zap.String("subscription_id", id.String()))
		return nil, err
	}
	txn.ID = txnID

	// Update envelope allocation spent amount if linked
	if sub.EnvelopeID != nil && *sub.EnvelopeID != uuid.Nil {
		if err := a.repos.Allocation.UpdateSpentAmount(*sub.EnvelopeID, now, sub.AmountE5, nil); err != nil {
			a.logger.Warn("Failed to update allocation spent for subscription envelope (continuing)", zap.Error(err))
		}
	}

	// Advance next billing date and persist
	sub.RecordCharge(txnID, now, txn.Description, sub.AmountE5)
	if err := a.repos.Subscription.UpdateSubscription(sub, nil); err != nil {
		a.logger.Error("Failed to update subscription next billing date", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	a.logger.Info("Successfully renewed subscription",
		zap.String("subscription_id", id.String()),
		zap.String("transaction_id", txnID.String()),
		zap.Time("next_billing_date", sub.NextBillingDate),
	)

	return txn, nil
}

func (a *SubscriptionActivities) GetDueSubscriptionsActivity(ctx context.Context, asOf time.Time) ([]*core.Subscription, error) {
	if asOf.IsZero() {
		asOf = utils.NowUTC()
	}

	subs, err := a.repos.Subscription.GetDueSubscriptions(asOf, nil)
	if err != nil {
		a.logger.Error("Failed to fetch due subscriptions in activity", zap.Error(err))
		return nil, err
	}

	return subs, nil
}

func (a *SubscriptionActivities) MatchSubscriptionIntentActivity(ctx context.Context, txn core.Transaction) (*core.Subscription, error) {
	if txn.UserID == uuid.Nil {
		return nil, errors.New("user ID cannot be nil")
	}

	subs, err := a.repos.Subscription.GetSubscriptionsByUserUUID(txn.UserID)
	if err != nil {
		a.logger.Error("Failed to fetch subscriptions for intent matching", zap.Error(err), zap.String("user_id", txn.UserID.String()))
		return nil, err
	}

	for _, sub := range subs {
		if sub.MatchesTransaction(&txn) {
			a.logger.Info("Matched recurring subscription intent to transaction",
				zap.String("subscription_id", sub.ID.String()),
				zap.String("subscription_name", sub.Name),
				zap.Int64("amount_e5", sub.AmountE5),
			)
			return sub, nil
		}
	}

	return nil, nil
}

func (a *SubscriptionActivities) RecordSubscriptionChargeActivity(ctx context.Context, subID uuid.UUID, txnID uuid.UUID, chargedAt time.Time, rawDesc string, amountE5 int64) error {
	if subID == uuid.Nil {
		return errors.New("subscription ID cannot be nil")
	}

	sub, err := a.repos.Subscription.GetSubscriptionByID(subID)
	if err != nil {
		a.logger.Error("Failed to fetch subscription for recording charge", zap.Error(err), zap.String("id", subID.String()))
		return err
	}

	if chargedAt.IsZero() {
		chargedAt = utils.NowUTC()
	}

	sub.RecordCharge(txnID, chargedAt, rawDesc, amountE5)

	if err := a.repos.Subscription.UpdateSubscription(sub, nil); err != nil {
		a.logger.Error("Failed to update subscription after recording charge", zap.Error(err), zap.String("id", subID.String()))
		return err
	}

	a.logger.Info("Recorded subscription charge and updated learning profile",
		zap.String("subscription_id", sub.ID.String()),
		zap.String("transaction_id", txnID.String()),
		zap.Int("occurrence_count", sub.OccurrenceCount),
		zap.Int("charge_window_hours", sub.ChargeWindowHours),
		zap.Time("next_billing_date", sub.NextBillingDate),
	)

	return nil
}

