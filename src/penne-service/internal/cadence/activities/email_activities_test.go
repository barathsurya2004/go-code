package activities

import (
	"context"
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/emailparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEmailActivities_ParseEmailActivity_Success(t *testing.T) {
	acts := NewEmailActivities(zap.NewNop())
	input := ParseEmailActivityInput{
		Subject: "Transaction Alert",
		Body: `Dear Mr. Barath Surya M,
Greetings from IDFC FIRST Bank.
Your A/C XXXXXXX2559 has been debited by INR 10.00 on 22/09/2026 16:08. New balance is INR 33,918.42CR.
Always You First,
Team IDFC FIRST Bank`,
	}

	res, err := acts.ParseEmailActivity(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, emailparser.VariantBankAccount, res.Variant)
	assert.Equal(t, "XXXXXXX2559", res.AccountOrCardNumber)
	assert.Equal(t, int64(1000000), res.AmountE5)
	assert.Equal(t, core.TxnTypeDebit, res.Type)
	assert.Equal(t, "bank_account", res.PaymentMethod)
}

func TestEmailActivities_ParseEmailActivity_Error(t *testing.T) {
	acts := NewEmailActivities(zap.NewNop())
	input := ParseEmailActivityInput{
		Subject: "Random Newsletter",
		Body:    "Hello world, no financial transaction here.",
	}

	res, err := acts.ParseEmailActivity(context.Background(), input)
	assert.Error(t, err)
	assert.Nil(t, res)
}
