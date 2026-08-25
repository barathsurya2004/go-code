package activities

import (
	"context"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
)

type TransactionActivities struct {
	Repos core.RepoContainer
}

func NewTransactionActivities(repos core.RepoContainer) *TransactionActivities {
	return &TransactionActivities{
		Repos: repos,
	}
}

func (a *TransactionActivities) CreateTransaction(ctx context.Context, txn core.Transaction) (*uuid.UUID, error) {
	temp := uuid.New()
	return &temp, nil
}
