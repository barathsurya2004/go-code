package activities

import (
	"context"
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
)

func TestTransactionActivities(t *testing.T) {
	repos := core.RepoContainer{}
	acts := NewTransactionActivities(repos)
	if acts == nil {
		t.Fatal("expected non-nil TransactionActivities")
	}

	txnID, err := acts.CreateTransaction(context.Background(), core.Transaction{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if txnID == nil {
		t.Fatal("expected non-nil txnID")
	}
}
