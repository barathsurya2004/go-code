package emailparser

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
)

type EmailVariant string

const (
	VariantBankAccount EmailVariant = "bank_account"
	VariantCreditCard  EmailVariant = "bank_card"
	VariantUnknown     EmailVariant = "unknown"
)

// ParsedEmailResult contains the structured transaction information extracted from an email.
type ParsedEmailResult struct {
	Variant             EmailVariant `json:"variant"`
	AccountOrCardNumber string       `json:"account_or_card_number"`
	Amount              float64      `json:"amount"`
	AmountE5            int64        `json:"amount_e5"`
	Currency            string       `json:"currency"`
	Type                string       `json:"txn_type"` // core.TxnTypeDebit or core.TxnTypeCredit
	PaymentMethod       string       `json:"payment_method"`
	Merchant            string       `json:"merchant,omitempty"`
	Balance             *float64     `json:"balance,omitempty"`
	AvailableLimit      *float64     `json:"available_limit,omitempty"`
	RawDateInBody       string       `json:"raw_date_in_body,omitempty"`
}

var (
	// Bank Account Patterns
	// E.g. "Your A/C XXXXXXX2559 has been debited by INR 10.00 on 22/09/2026 16:08."
	reBankAccount = regexp.MustCompile(`(?i)Your\s+A/C\s+([A-Za-z0-9]+)\s+has\s+been\s+(debited|credited)(?:\s+(?:by|with))?\s+INR\s*([0-9,]+(?:\.[0-9]+)?)(?:\s+on\s+([0-9/:\s-]+))?`)
	reBankBalance = regexp.MustCompile(`(?i)New\s+balance\s+is\s+INR\s*([0-9,]+(?:\.[0-9]+)?)`)

	// Credit Card Patterns
	// E.g. "INR 1171.50 spent on your IDFC FIRST BANK Credit Card ending XX1110 at AVENUE SUPERMARTS LI on 22 SEP 2026."
	reCreditCard = regexp.MustCompile(`(?i)INR\s*([0-9,]+(?:\.[0-9]+)?)\s+(spent\s+on|debited\s+from|credited\s+to|refunded\s+on|refunded\s+to|credited\s+on)\s+your\s+IDFC\s+FIRST\s+BANK\s+Credit\s+Card\s+ending\s+([A-Za-z0-9]+)(?:\s+at\s+([^\n\r.]+?))?\s+on\s+([^\n\r.]+)`)
	reCardLimit  = regexp.MustCompile(`(?i)Available\s+Limit:\s*INR\s*([0-9,]+(?:\.[0-9]+)?)`)

	ErrUnsupportedEmail = errors.New("unsupported email format or not an IDFC FIRST Bank transaction email")
	ErrInvalidAmount    = errors.New("invalid transaction amount")
)

// ParseIDFCEmail parses the subject and text body of an IDFC FIRST Bank email.
func ParseIDFCEmail(subject, body string) (*ParsedEmailResult, error) {
	// First check bank account pattern
	if match := reBankAccount.FindStringSubmatch(body); len(match) > 3 {
		accNum := strings.TrimSpace(match[1])
		action := strings.ToLower(strings.TrimSpace(match[2]))
		amountStr := strings.TrimSpace(match[3])
		rawDate := ""
		if len(match) > 4 {
			rawDate = strings.TrimSpace(match[4])
		}

		amount, amountE5, err := parseAmountToE5(amountStr)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidAmount, err)
		}

		txnType := core.TxnTypeDebit
		if strings.Contains(action, "credit") {
			txnType = core.TxnTypeCredit
		}

		res := &ParsedEmailResult{
			Variant:             VariantBankAccount,
			AccountOrCardNumber: accNum,
			Amount:              amount,
			AmountE5:            amountE5,
			Currency:            "INR",
			Type:                txnType,
			PaymentMethod:       "bank_account",
			RawDateInBody:       rawDate,
		}

		if balMatch := reBankBalance.FindStringSubmatch(body); len(balMatch) > 1 {
			if bal, _, err := parseAmountToE5(balMatch[1]); err == nil {
				res.Balance = &bal
			}
		}

		return res, nil
	}

	// Next check credit card pattern
	if match := reCreditCard.FindStringSubmatch(body); len(match) > 3 {
		amountStr := strings.TrimSpace(match[1])
		action := strings.ToLower(strings.TrimSpace(match[2]))
		cardNum := strings.TrimSpace(match[3])
		merchant := ""
		if len(match) > 4 {
			merchant = strings.TrimSpace(match[4])
		}
		rawDate := ""
		if len(match) > 5 {
			rawDate = strings.TrimSpace(match[5])
		}

		amount, amountE5, err := parseAmountToE5(amountStr)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidAmount, err)
		}

		txnType := core.TxnTypeDebit
		if strings.Contains(action, "credit") || strings.Contains(action, "refund") {
			txnType = core.TxnTypeCredit
		}

		res := &ParsedEmailResult{
			Variant:             VariantCreditCard,
			AccountOrCardNumber: cardNum,
			Amount:              amount,
			AmountE5:            amountE5,
			Currency:            "INR",
			Type:                txnType,
			PaymentMethod:       "bank_card",
			Merchant:            merchant,
			RawDateInBody:       rawDate,
		}

		if limMatch := reCardLimit.FindStringSubmatch(body); len(limMatch) > 1 {
			if limit, _, err := parseAmountToE5(limMatch[1]); err == nil {
				res.AvailableLimit = &limit
			}
		}

		return res, nil
	}

	return nil, ErrUnsupportedEmail
}

func parseAmountToE5(amountStr string) (float64, int64, error) {
	cleanStr := strings.ReplaceAll(amountStr, ",", "")
	cleanStr = strings.TrimSpace(cleanStr)
	val, err := strconv.ParseFloat(cleanStr, 64)
	if err != nil {
		return 0, 0, err
	}
	if val < 0 {
		return 0, 0, errors.New("negative amount")
	}
	// Convert to E5: amount * 10^5
	e5 := int64(math.Round(val * 100000.0))
	return val, e5, nil
}
