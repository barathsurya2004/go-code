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
}

