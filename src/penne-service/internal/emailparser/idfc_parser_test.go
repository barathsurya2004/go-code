package emailparser

import (
	"strings"
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIDFCEmail_BankAccount_Debit(t *testing.T) {
	emailBody := `Dear Mr. Barath Surya M,
Greetings from IDFC FIRST Bank.
Your A/C XXXXXXX2559 has been debited by INR 10.00 on 22/09/2026 16:08. New balance is INR 33,918.42CR.
Always You First,
Team IDFC FIRST Bank
If this transaction was not initiated by you, SMS BLOCK (last 4 digits of your card) to 5676732 or call 180010888 from your registered mobile number immediately.
This is an auto generated e-mail. Please do not reply.
Disclaimer
IDFC FIRST Bank will never ask for your confidential information.`

	result, err := ParseIDFCEmail("Transaction Alert", emailBody)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, VariantBankAccount, result.Variant)
	assert.Equal(t, "XXXXXXX2559", result.AccountOrCardNumber)
	assert.Equal(t, 10.00, result.Amount)
	assert.Equal(t, int64(1000000), result.AmountE5)
	assert.Equal(t, "INR", result.Currency)
	assert.Equal(t, core.TxnTypeDebit, result.Type)
	assert.Equal(t, "bank_account", result.PaymentMethod)
	require.NotNil(t, result.Balance)
	assert.Equal(t, 33918.42, *result.Balance)
}

func TestParseIDFCEmail_BankAccount_Credit(t *testing.T) {
	emailBody := `Dear Mr. Barath Surya M,
Greetings from IDFC FIRST Bank.
Your A/C XXXXXXX2559 has been credited by INR 5,250.50 on 22/09/2026 18:30. New balance is INR 39,168.92CR.
Always You First,
Team IDFC FIRST Bank`

	result, err := ParseIDFCEmail("Account Credited", emailBody)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, VariantBankAccount, result.Variant)
	assert.Equal(t, "XXXXXXX2559", result.AccountOrCardNumber)
	assert.Equal(t, 5250.50, result.Amount)
	assert.Equal(t, int64(525050000), result.AmountE5)
	assert.Equal(t, "INR", result.Currency)
	assert.Equal(t, core.TxnTypeCredit, result.Type)
	assert.Equal(t, "bank_account", result.PaymentMethod)
	require.NotNil(t, result.Balance)
	assert.Equal(t, 39168.92, *result.Balance)
}

func TestParseIDFCEmail_CreditCard_Spent(t *testing.T) {
	emailBody := `Dear Cardmember,

All Stocked Up! INR 1171.50 spent on your IDFC FIRST BANK Credit Card ending XX1110 at AVENUE SUPERMARTS LI on 22 SEP 2026.

Available Limit: INR 38413.57 .`

	result, err := ParseIDFCEmail("Card Spend Alert", emailBody)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, VariantCreditCard, result.Variant)
	assert.Equal(t, "XX1110", result.AccountOrCardNumber)
	assert.Equal(t, 1171.50, result.Amount)
	assert.Equal(t, int64(117150000), result.AmountE5)
	assert.Equal(t, "INR", result.Currency)
	assert.Equal(t, core.TxnTypeDebit, result.Type)
	assert.Equal(t, "bank_card", result.PaymentMethod)
	assert.Equal(t, "AVENUE SUPERMARTS LI", result.Merchant)
	require.NotNil(t, result.AvailableLimit)
	assert.Equal(t, 38413.57, *result.AvailableLimit)
}

func TestParseIDFCEmail_CreditCard_Refund(t *testing.T) {
	emailBody := `Dear Cardmember,

INR 500.00 refunded on your IDFC FIRST BANK Credit Card ending XX1110 at FLIPKART on 22 SEP 2026.

Available Limit: INR 38913.57 .`

	result, err := ParseIDFCEmail("Card Refund Alert", emailBody)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, VariantCreditCard, result.Variant)
	assert.Equal(t, "XX1110", result.AccountOrCardNumber)
	assert.Equal(t, 500.00, result.Amount)
	assert.Equal(t, int64(50000000), result.AmountE5)
	assert.Equal(t, "INR", result.Currency)
	assert.Equal(t, core.TxnTypeCredit, result.Type)
	assert.Equal(t, "bank_card", result.PaymentMethod)
	assert.Equal(t, "FLIPKART", result.Merchant)
	require.NotNil(t, result.AvailableLimit)
	assert.Equal(t, 38913.57, *result.AvailableLimit)
}

func TestParseIDFCEmail_UnsupportedEmail(t *testing.T) {
	_, err := ParseIDFCEmail("Promo", "Get 50% discount on your next flight booking!")
	assert.ErrorIs(t, err, ErrUnsupportedEmail)
}

func TestParseIDFCEmail_BankAccount_WithoutBalance(t *testing.T) {
	emailBody := "Your A/C XXXXXXX2559 has been debited by INR 100.00."
	res, err := ParseIDFCEmail("Alert", emailBody)
	require.NoError(t, err)
	assert.Equal(t, int64(10000000), res.AmountE5)
	assert.Nil(t, res.Balance)
}

func TestParseIDFCEmail_CreditCard_WithoutLimit(t *testing.T) {
	emailBody := "INR 250.00 spent on your IDFC FIRST BANK Credit Card ending XX1110 at STORE on 22 SEP 2026."
	res, err := ParseIDFCEmail("Alert", emailBody)
	require.NoError(t, err)
	assert.Equal(t, int64(25000000), res.AmountE5)
	assert.Nil(t, res.AvailableLimit)
}

func TestParseAmountToE5_Errors(t *testing.T) {
	_, _, err := parseAmountToE5("invalid")
	assert.Error(t, err)

	_, _, err = parseAmountToE5("-50.00")
	assert.Error(t, err)
}

func TestParseIDFCEmail_Variations(t *testing.T) {
	emailBody1 := "Your A/C XXXXXXX2559 has been credited with INR 500.00 on 22/09/2026."
	res1, err := ParseIDFCEmail("Alert", emailBody1)
	require.NoError(t, err)
	assert.Equal(t, core.TxnTypeCredit, res1.Type)

	emailBody2 := "INR 100.00 credited to your IDFC FIRST BANK Credit Card ending XX1110 at STORE on 22 SEP 2026."
	res2, err := ParseIDFCEmail("Alert", emailBody2)
	require.NoError(t, err)
	assert.Equal(t, core.TxnTypeCredit, res2.Type)

	// Overflow amount triggers ErrInvalidAmount branch
	hugeNum := "1" + strings.Repeat("9", 400)
	emailBody3 := "Your A/C XXXXXXX2559 has been debited by INR " + hugeNum + "."
	_, err = ParseIDFCEmail("Alert", emailBody3)
	assert.ErrorIs(t, err, ErrInvalidAmount)

	emailBody4 := "INR " + hugeNum + " spent on your IDFC FIRST BANK Credit Card ending XX1110 at STORE on 22 SEP 2026."
	_, err = ParseIDFCEmail("Alert", emailBody4)
	assert.ErrorIs(t, err, ErrInvalidAmount)
}



