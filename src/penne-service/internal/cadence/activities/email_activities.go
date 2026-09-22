package activities

import (
	"context"

	"github.com/barathsurya2004/go-code/penne-service/internal/emailparser"
	"go.uber.org/zap"
)

type EmailActivities struct {
	logger *zap.Logger
}

type ParseEmailActivityInput struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func NewEmailActivities(logger *zap.Logger) *EmailActivities {
	return &EmailActivities{
		logger: logger,
	}
}

func (a *EmailActivities) ParseEmailActivity(ctx context.Context, input ParseEmailActivityInput) (*emailparser.ParsedEmailResult, error) {
	result, err := emailparser.ParseIDFCEmail(input.Subject, input.Body)
	if err != nil {
		a.logger.Error("Failed to parse email", zap.Error(err), zap.String("subject", input.Subject))
		return nil, err
	}

	a.logger.Info("Successfully parsed transaction email",
		zap.String("variant", string(result.Variant)),
		zap.String("type", result.Type),
		zap.Int64("amount_e5", result.AmountE5),
		zap.String("account_or_card", result.AccountOrCardNumber),
	)

	return result, nil
}
