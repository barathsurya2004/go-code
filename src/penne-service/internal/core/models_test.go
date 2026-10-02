package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestModelsJSON(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	txnUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174001")
	tokenUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174002")

	t.Run("Transaction JSON", func(t *testing.T) {
		envID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174003")
		txn := Transaction{
			ID:         txnUUID,
			UserID:     userUUID,
			EnvelopeID: &envID,
			AmountE5:   500,
			CountryISO: "US",
			PaymentMethod: "Chase",
			Type:       "debit",
			CreatedAt:  now,
		}

		data, err := json.Marshal(txn)
		if err != nil {
			t.Fatalf("failed to marshal Transaction: %v", err)
		}

		var decoded Transaction
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal Transaction: %v", err)
		}

		if decoded.ID != txn.ID || decoded.AmountE5 != txn.AmountE5 || decoded.UserID != txn.UserID {
			t.Errorf("unmarshalled transaction mismatch: got %+v, want %+v", decoded, txn)
		}
	})

	t.Run("User JSON", func(t *testing.T) {
		user := User{
			UUID:      userUUID,
			Name:      "Alice",
			CreatedAt: now,
			UpdatedAt: now,
		}

		data, err := json.Marshal(user)
		if err != nil {
			t.Fatalf("failed to marshal User: %v", err)
		}

		var decoded User
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal User: %v", err)
		}

		if decoded.UUID != user.UUID || decoded.Name != user.Name {
			t.Errorf("unmarshalled user mismatch: got %+v, want %+v", decoded, user)
		}
	})

	t.Run("Token JSON", func(t *testing.T) {
		exp := now.Add(24 * time.Hour)
		token := Token{
			UserUUID:  userUUID,
			Token:     tokenUUID,
			Prefix:    "mcp_",
			Name:      "default",
			Scope:     []string{"read", "write"},
			ExpiresAt: &exp,
			CreatedAt: now,
			UpdatedAt: now,
		}

		data, err := json.Marshal(token)
		if err != nil {
			t.Fatalf("failed to marshal Token: %v", err)
		}

		var decoded Token
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal Token: %v", err)
		}

		if decoded.UserUUID != token.UserUUID || decoded.Token != token.Token || len(decoded.Scope) != 2 {
			t.Errorf("unmarshalled token mismatch: got %+v, want %+v", decoded, token)
		}
	})

	t.Run("ShortcutIntent JSON", func(t *testing.T) {
		intentID := uuid.New()
		envID := uuid.New()
		intent := ShortcutIntent{
			ID:         intentID,
			UserID:     userUUID,
			EnvelopeID: &envID,
			Latitude:   12.9716,
			Longitude:  77.5946,
			Status:     "pending",
			CreatedAt:  now,
		}

		data, err := json.Marshal(intent)
		if err != nil {
			t.Fatalf("failed to marshal ShortcutIntent: %v", err)
		}

		var decoded ShortcutIntent
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal ShortcutIntent: %v", err)
		}

		if decoded.ID != intent.ID || decoded.UserID != intent.UserID || decoded.Status != intent.Status {
			t.Errorf("unmarshalled shortcut intent mismatch: got %+v, want %+v", decoded, intent)
		}
	})

	t.Run("Subscription JSON", func(t *testing.T) {
		subID := uuid.New()
		sub := Subscription{
			ID:              subID,
			UserUUID:        userUUID,
			Name:            "Netflix",
			AmountE5:        64900000,
			BillingCycle:    BillingCycleMonthly,
			NextBillingDate: now,
			PaymentMethod:   "bank_card",
			Status:          SubscriptionStatusActive,
			AutoRenew:       true,
			Notes:           "Premium plan",
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		data, err := json.Marshal(sub)
		if err != nil {
			t.Fatalf("failed to marshal Subscription: %v", err)
		}

		var decoded Subscription
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal Subscription: %v", err)
		}

		if decoded.ID != sub.ID || decoded.Name != sub.Name || decoded.AmountE5 != sub.AmountE5 {
			t.Errorf("unmarshalled subscription mismatch: got %+v, want %+v", decoded, sub)
		}
	})
}

func TestSubscriptionHelpers(t *testing.T) {
	t.Run("CalculateMonthlyEquivalentE5", func(t *testing.T) {
		// Monthly
		if got := CalculateMonthlyEquivalentE5(1000, BillingCycleMonthly); got != 1000 {
			t.Errorf("expected 1000, got %d", got)
		}
		// Weekly (100 * 52 / 12 = 433)
		if got := CalculateMonthlyEquivalentE5(100, BillingCycleWeekly); got != 433 {
			t.Errorf("expected 433, got %d", got)
		}
		// Quarterly (3000 / 3 = 1000)
		if got := CalculateMonthlyEquivalentE5(3000, BillingCycleQuarterly); got != 1000 {
			t.Errorf("expected 1000, got %d", got)
		}
		// Yearly (12000 / 12 = 1000)
		if got := CalculateMonthlyEquivalentE5(12000, BillingCycleYearly); got != 1000 {
			t.Errorf("expected 1000, got %d", got)
		}
		// Default
		if got := CalculateMonthlyEquivalentE5(500, "unknown"); got != 500 {
			t.Errorf("expected 500, got %d", got)
		}
	})

	t.Run("AdvanceBillingDate", func(t *testing.T) {
		base := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

		// Weekly -> +7 days (2026-01-22)
		if got := AdvanceBillingDate(base, BillingCycleWeekly); got != base.AddDate(0, 0, 7) {
			t.Errorf("weekly advance failed: %v", got)
		}

		// Quarterly -> +3 months (2026-04-15)
		if got := AdvanceBillingDate(base, BillingCycleQuarterly); got != base.AddDate(0, 3, 0) {
			t.Errorf("quarterly advance failed: %v", got)
		}

		// Yearly -> +1 year (2027-01-15)
		if got := AdvanceBillingDate(base, BillingCycleYearly); got != base.AddDate(1, 0, 0) {
			t.Errorf("yearly advance failed: %v", got)
		}

		// Monthly -> +1 month (2026-02-15)
		if got := AdvanceBillingDate(base, BillingCycleMonthly); got != base.AddDate(0, 1, 0) {
			t.Errorf("monthly advance failed: %v", got)
		}

		// Default -> +1 month
		if got := AdvanceBillingDate(base, "unknown"); got != base.AddDate(0, 1, 0) {
			t.Errorf("default advance failed: %v", got)
		}
	})

	t.Run("GetExpectedChargeTime", func(t *testing.T) {
		baseDate := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
		sub := &Subscription{
			NextBillingDate: baseDate,
			OccurrenceCount: 0,
		}

		// When OccurrenceCount == 0, returns NextBillingDate
		if got := sub.GetExpectedChargeTime(); !got.Equal(baseDate) {
			t.Errorf("expected %v, got %v", baseDate, got)
		}

		// When OccurrenceCount > 0 and LastChargedAt is set
		lastCharged := time.Date(2026, 4, 10, 18, 45, 30, 0, time.UTC)
		sub.OccurrenceCount = 1
		sub.LastChargedAt = &lastCharged

		expected := time.Date(2026, 5, 10, 18, 45, 30, 0, time.UTC)
		if got := sub.GetExpectedChargeTime(); !got.Equal(expected) {
			t.Errorf("expected %v, got %v", expected, got)
		}
	})

	t.Run("MatchesTransaction", func(t *testing.T) {
		baseDate := time.Date(2026, 5, 10, 18, 0, 0, 0, time.UTC)
		sub := &Subscription{
			Name:              "Netflix",
			AmountE5:          64900000,
			Status:            SubscriptionStatusActive,
			NextBillingDate:   baseDate,
			ChargeWindowHours: 48,
		}

		// Nil checks
		if sub.MatchesTransaction(nil) {
			t.Errorf("expected false for nil transaction")
		}
		var nilSub *Subscription
		if nilSub.MatchesTransaction(&Transaction{}) {
			t.Errorf("expected false for nil subscription")
		}

		// Inactive subscription
		sub.Status = SubscriptionStatusPaused
		if sub.MatchesTransaction(&Transaction{Type: TxnTypeDebit, Description: "Netflix", AmountE5: 64900000, CreatedAt: baseDate}) {
			t.Errorf("expected false for paused subscription")
		}
		sub.Status = SubscriptionStatusActive

		// Credit transaction
		if sub.MatchesTransaction(&Transaction{Type: TxnTypeCredit, Description: "Netflix", AmountE5: 64900000, CreatedAt: baseDate}) {
			t.Errorf("expected false for credit transaction")
		}

		// Amount mismatch > 10%
		if sub.MatchesTransaction(&Transaction{Type: TxnTypeDebit, Description: "Netflix", AmountE5: 80000000, CreatedAt: baseDate}) {
			t.Errorf("expected false for amount mismatch > 10%%")
		}

		// Merchant description mismatch
		if sub.MatchesTransaction(&Transaction{Type: TxnTypeDebit, Description: "Uber Trip", AmountE5: 64900000, CreatedAt: baseDate}) {
			t.Errorf("expected false for merchant mismatch")
		}

		// Time window mismatch (> 48h)
		wayLater := baseDate.Add(50 * time.Hour)
		if sub.MatchesTransaction(&Transaction{Type: TxnTypeDebit, Description: "Netflix", AmountE5: 64900000, CreatedAt: wayLater}) {
			t.Errorf("expected false for time window mismatch")
		}

		// Exact match
		txn := &Transaction{
			Type:        TxnTypeDebit,
			Description: "NETFLIX ENTERTAINMENT MUMBAI",
			AmountE5:    64900000,
			CreatedAt:   baseDate.Add(2 * time.Hour),
		}
		if !sub.MatchesTransaction(txn) {
			t.Errorf("expected true for valid match")
		}

		// Match via learned merchant pattern
		sub.MerchantPattern = "NFLX"
		txnPattern := &Transaction{
			Type:        TxnTypeDebit,
			Description: "CHARGE NFLX DIGITAL",
			AmountE5:    64900000,
			CreatedAt:   baseDate,
		}
		if !sub.MatchesTransaction(txnPattern) {
			t.Errorf("expected true for pattern match")
		}

		// Zero CreatedAt
		if !sub.MatchesTransaction(&Transaction{Type: TxnTypeDebit, Description: "Netflix", AmountE5: 64900000}) {
			t.Errorf("expected true for zero CreatedAt")
		}
	})

	t.Run("RecordCharge", func(t *testing.T) {
		baseDate := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
		sub := &Subscription{
			Name:            "Spotify",
			AmountE5:        11900000,
			BillingCycle:    BillingCycleMonthly,
			NextBillingDate: baseDate,
			OccurrenceCount: 0,
		}

		txnID1 := uuid.New()
		chargeTime1 := time.Date(2026, 5, 10, 14, 20, 0, 0, time.UTC)

		// First charge: sets 24h window, occurrence=1, learns pattern
		sub.RecordCharge(txnID1, chargeTime1, "SPOTIFY AB STOCKHOLM", 11900000)

		if sub.OccurrenceCount != 1 {
			t.Errorf("expected occurrence count 1, got %d", sub.OccurrenceCount)
		}
		if sub.ChargeWindowHours != 24 {
			t.Errorf("expected 24h window, got %d", sub.ChargeWindowHours)
		}
		if sub.MerchantPattern != "SPOTIFY AB STOCKHOLM" {
			t.Errorf("expected merchant pattern, got %s", sub.MerchantPattern)
		}
		if sub.LastTransactionID == nil || *sub.LastTransactionID != txnID1 {
			t.Errorf("expected last transaction ID to match")
		}
		if sub.NextBillingDate != baseDate.AddDate(0, 1, 0) {
			t.Errorf("expected next billing date +1 month")
		}

		// Second charge: tightens window to 12h, occurrence=2
		txnID2 := uuid.New()
		chargeTime2 := time.Date(2026, 6, 10, 14, 22, 0, 0, time.UTC)
		sub.RecordCharge(txnID2, chargeTime2, "SPOTIFY AB", 12900000)

		if sub.OccurrenceCount != 2 {
			t.Errorf("expected occurrence count 2, got %d", sub.OccurrenceCount)
		}
		if sub.ChargeWindowHours != 12 {
			t.Errorf("expected 12h window, got %d", sub.ChargeWindowHours)
		}
		if sub.AmountE5 != 12900000 {
			t.Errorf("expected updated amount 12900000, got %d", sub.AmountE5)
		}
	})
}


