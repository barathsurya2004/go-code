package core

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	AuthToken   = "auth_token"
	DefaultName = "default"

	StatusPending = "PENDING"
	StatusSettled = "SETTLED"
	StatusExpired = "EXPIRED"

	TxnTypeCredit   = "credit"
	TxnTypeDebit    = "debit"
	TxnTypeTransfer = "transfer"
)

type Transaction struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	EnvelopeID       *uuid.UUID `json:"envelope_id" db:"envelope_id"` // Nullable if uncategorized yet
	AmountE5         int64      `json:"amount_e5" db:"amount_e5"`
	Type             string     `json:"txn_type" db:"txn_type"`
	PaymentMethod    string     `json:"payment_method" db:"payment_method"`
	CountryISO       string     `json:"country_iso2" db:"country_iso2"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	ShortcutIntentID *uuid.UUID `json:"shortcut_intent_id" db:"shortcut_intent_id"`
	Description      string     `json:"description" db:"description"`
	WishlistItemID   *uuid.UUID `json:"wishlist_item_id,omitempty" db:"wishlist_item_id"`
}

type TransactionRepository interface {
	CreateTransaction(txn *Transaction, Tx *sql.Tx) (uuid.UUID, error)
	GetTransactionByUUID(uuid uuid.UUID) (*Transaction, error)
	GetTransactionsByUserUUID(userUUID uuid.UUID) ([]*Transaction, error)
	GetTransactionByTime(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*Transaction, error)
	GetTransactionByAmountAndTime(userUUID uuid.UUID, amountE5 int64, time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*Transaction, error)
	UpdateTransaction(txn *Transaction, Tx *sql.Tx) error
	DeleteTransaction(uuid uuid.UUID) error
	GetDashboardSummary(uuid uuid.UUID) (*DashboardSummary, error)
	GetTransactionByUserUUIDPaginated(userUUID uuid.UUID, lastTransactionCreatedAt time.Time, lastTransactionID uuid.UUID, limit int) ([]*Transaction, error)
	GetMonthlyInsights(userUUID uuid.UUID, year int, month int) (*MonthlyInsightsReport, error)
}

type User struct {
	UUID            uuid.UUID `json:"uuid"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"password_hash"`
	MonthlyBudgetE5 int64     `json:"monthly_budget_e5"`
	SalaryDay       int       `json:"salary_day"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UserRepository interface {
	CreateUser(user *User, Tx *sql.Tx) (uuid.UUID, error)
	GetUserByUUID(uuid uuid.UUID) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateBudgetSettings(userUUID uuid.UUID, monthlyBudgetE5 int64, salaryDay int, Tx *sql.Tx) error
}

type Token struct {
	UserUUID   uuid.UUID  `json:"user_uuid"`
	Token      uuid.UUID  `json:"token"`
	Prefix     string     `json:"prefix"`
	Name       string     `json:"name"`
	Scope      []string   `json:"scope"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type TokenRepository interface {
	CreateToken(token *Token, Tx *sql.Tx) (uuid.UUID, error)
	DeleteToken(userUUID uuid.UUID) error
	GetToken(token uuid.UUID) (*Token, error)
	GetActiveTokenWithUserUUID(userUUID uuid.UUID) (*Token, error)
	UpdateToken(token *Token) error
}

type EnvelopeGroup struct {
	ID        uuid.UUID `json:"id"`
	UserUUID  uuid.UUID `json:"user_uuid"`
	Name      string    `json:"name"`
	IsSystem  bool      `json:"is_system"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EnvelopeGroupRepository interface {
	CreateEnvelopeGroup(envelopeGroup *EnvelopeGroup, Tx *sql.Tx) (uuid.UUID, error)
	GetEnvelopeGroupByID(id uuid.UUID) (*EnvelopeGroup, error)
	GetEnvelopeGroupsByUserUUID(userUUID uuid.UUID) ([]*EnvelopeGroup, error)
	UpdateEnvelopeGroup(envelopeGroup *EnvelopeGroup) error
	DeleteEnvelopeGroup(id uuid.UUID) error
}

type Envelope struct {
	ID              uuid.UUID `json:"id"`
	UserUUID        uuid.UUID `json:"user_uuid"`
	EnvelopeGroupID uuid.UUID `json:"envelope_group_id"`
	Name            string    `json:"name"`
	TargetAmountE5  float64   `json:"target_amount_e5"`
	Cadence         Cadence   `json:"cadence"`
	CountryISO      string    `json:"country_iso2"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	IsSystem        bool      `json:"is_system"`
}

type EnvelopeRepository interface {
	CreateEnvelope(envelope *Envelope, Tx *sql.Tx) (uuid.UUID, error)
	GetEnvelopeByID(id uuid.UUID) (*Envelope, error)
	GetEnvelopesByUserUUID(userUUID uuid.UUID) ([]*Envelope, error)
	UpdateEnvelope(envelope *Envelope) error
	DeleteEnvelope(id uuid.UUID) error
	GetEnvelopeIdByName(envlopeName string, userUUID uuid.UUID, tx *sql.Tx) (uuid.UUID, error)
}

type Allocation struct {
	ID                uuid.UUID  `json:"id"`
	EnvelopeID        uuid.UUID  `json:"envelope_id"`
	AllocatedAmountE5 float64    `json:"allocated_amount_e5"`
	SpentAmountE5     int64      `json:"spent_amount_e5"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	StartDate         *time.Time `json:"start_date,omitempty"`
	EndDate           *time.Time `json:"end_date,omitempty"`
}

type AllocationRepository interface {
	CreateAllocation(allocation *Allocation, Tx *sql.Tx) (uuid.UUID, error)
	GetAllocationByID(id uuid.UUID) (*Allocation, error)
	GetAllocationsByEnvelopeID(envelopeID uuid.UUID) ([]*Allocation, error)
	GetActiveAllocationsByUserUUID(userUUID uuid.UUID, targetDate time.Time, Tx *sql.Tx) ([]*Allocation, error)
	UpdateAllocation(allocation *Allocation) error
	DeleteAllocation(id uuid.UUID) error
	UpdateSpentAmount(envelopeID uuid.UUID, targetDate time.Time, amountDeltaE5 int64, Tx *sql.Tx) error
}

type ShortcutIntent struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	EnvelopeID    *uuid.UUID `json:"envelope_id"`
	Latitude      float64    `json:"latitude"`
	Longitude     float64    `json:"longitude"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	TransactionID *uuid.UUID `json:"transaction_id"`
}

type ShortcutIntentRepository interface {
	CreateShortcutIntent(shortcutIntent *ShortcutIntent, Tx *sql.Tx) (uuid.UUID, error)
	GetShortcutIntentByID(id uuid.UUID) (*ShortcutIntent, error)
	GetShortcutIntentsByUserUUID(userUUID uuid.UUID) ([]*ShortcutIntent, error)
	UpdateShortcutIntent(shortcutIntent *ShortcutIntent, Tx *sql.Tx) error
	DeleteShortcutIntent(id uuid.UUID) error
	GetPendingRecentShortcutIntent(userUUID uuid.UUID, Tx *sql.Tx, time_lowerbound, time_upperbound time.Time) (*ShortcutIntent, error)
}

type RepoContainer struct {
	Transaction    TransactionRepository
	User           UserRepository
	Token          TokenRepository
	EnvelopeGroup  EnvelopeGroupRepository
	Envelope       EnvelopeRepository
	Allocation     AllocationRepository
	ShortcutIntent ShortcutIntentRepository
	Wishlist       WishlistRepository
	Subscription   SubscriptionRepository
}

type DashboardSummary struct {
	TotalIncomeE5       int64 `json:"total_income_e5"`
	BaseIncomeE5        int64 `json:"base_income_e5"`
	BufferedIncomeE5    int64 `json:"buffered_income_e5"`
	BufferedUsedE5      int64 `json:"buffered_used_e5"`
	BufferedRemainingE5 int64 `json:"buffered_remaining_e5"`
	TotalExpenseE5      int64 `json:"total_expense_e5"`
	TotalRemainingE5    int64 `json:"total_remaining_e5"`
	CardSpentE5         int64 `json:"card_spent_e5"`
	CardLimitE5         int64 `json:"card_limit_e5"`
	BankSpentE5         int64 `json:"bank_spent_e5"`
	BankLimitE5         int64 `json:"bank_limit_e5"`
}

type CategorySpendSplit struct {
	EnvelopeID       string  `json:"envelope_id"`
	EnvelopeName     string  `json:"envelope_name"`
	GroupName        string  `json:"group_name"`
	SpentE5          int64   `json:"spent_e5"`
	Percentage       float64 `json:"percentage"`
	TransactionCount int     `json:"transaction_count"`
}

type PeakSpendDayTransaction struct {
	ID            string `json:"id"`
	Description   string `json:"description"`
	AmountE5      int64  `json:"amount_e5"`
	PaymentMethod string `json:"payment_method,omitempty"`
	Date          string `json:"date,omitempty"`
	Category      string `json:"category,omitempty"`
	EnvelopeName  string `json:"envelope_name,omitempty"`
}

type PeakSpendDayInfo struct {
	Date             string                    `json:"date"`
	DayName          string                    `json:"day_name"`
	TotalSpentE5     int64                     `json:"total_spent_e5"`
	TransactionCount int                       `json:"transaction_count"`
	TopTransactions  []PeakSpendDayTransaction `json:"top_transactions"`
}

type DailySpendingHeatmapItem struct {
	Date             string                    `json:"date"`
	Day              int                       `json:"day"`
	DayOfWeek        int                       `json:"day_of_week"`
	DayName          string                    `json:"day_name"`
	TotalSpentE5     int64                     `json:"total_spent_e5"`
	TransactionCount int                       `json:"transaction_count"`
	IntensityLevel   int                       `json:"intensity_level"`
	IsFuture         bool                      `json:"is_future"`
	Transactions     []PeakSpendDayTransaction `json:"transactions,omitempty"`
}

type PaymentMethodSplit struct {
	Method  string `json:"method"`
	Label   string `json:"label"`
	SpentE5 int64  `json:"spent_e5"`
	Count   int    `json:"count"`
}

type PreviousMonthInsightsDelta struct {
	TotalExpenseE5 int64   `json:"total_expense_e5"`
	DeltaPct       float64 `json:"delta_pct"`
	IsLower        bool    `json:"is_lower"`
}

type MonthlyInsightsReport struct {
	Year                   int                         `json:"year"`
	Month                  int                         `json:"month"`
	MonthLabel             string                      `json:"month_label"`
	DaysInMonth            int                         `json:"days_in_month"`
	DaysElapsed            int                         `json:"days_elapsed"`
	TotalIncomeE5          int64                       `json:"total_income_e5"`
	TotalExpenseE5         int64                       `json:"total_expense_e5"`
	NetSavingsE5           int64                       `json:"net_savings_e5"`
	SavingsRatePct         float64                     `json:"savings_rate_pct"`
	SubscriptionExpenseE5  int64                       `json:"subscription_expense_e5"`
	DiscretionaryExpenseE5 int64                       `json:"discretionary_expense_e5"`
	SubscriptionCount      int                         `json:"subscription_count"`
	PeakDay                *PeakSpendDayInfo           `json:"peak_day,omitempty"`
	DailyHeatmap           []DailySpendingHeatmapItem  `json:"daily_heatmap"`
	FirstDayOffset         int                         `json:"first_day_offset"`
	MaxDailySpendE5        int64                       `json:"max_daily_spend_e5"`
	CategorySplits         []CategorySpendSplit        `json:"category_splits"`
	NoSpendDaysCount       int                         `json:"no_spend_days_count"`
	DailyAverageE5         int64                       `json:"daily_average_e5"`
	LargestTransaction     *PeakSpendDayTransaction    `json:"largest_transaction,omitempty"`
	PaymentMethodSplits    []PaymentMethodSplit        `json:"payment_method_splits"`
	PreviousMonth          *PreviousMonthInsightsDelta `json:"previous_month,omitempty"`
}

type CreateUserWorkflowResult struct {
	UserUUID      uuid.UUID `json:"user_uuid"`
	UserAuthToken uuid.UUID `json:"user_auth_token"`
}

type CreateSystemEnvelopeActivityInput struct {
	UserUUID        uuid.UUID `json:"user_uuid"`
	EnvelopeGroupID uuid.UUID `json:"envelope_group_id"`
}

type UpdateTransactionCategoryRequest struct {
	TransactionID uuid.UUID  `json:"transaction_id"`
	NewEnvelopeID *uuid.UUID `json:"new_envelope_id"`
	AmountE5      int64      `json:"amount_e5,omitempty"`
	TxnType       string     `json:"txn_type,omitempty"`
	PaymentMethod string     `json:"payment_method,omitempty"`
}

const (
	BillingCycleWeekly    = "weekly"
	BillingCycleMonthly   = "monthly"
	BillingCycleQuarterly = "quarterly"
	BillingCycleYearly    = "yearly"

	SubscriptionStatusActive    = "active"
	SubscriptionStatusPaused    = "paused"
	SubscriptionStatusCancelled = "cancelled"
)

type Subscription struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	UserUUID          uuid.UUID  `json:"user_uuid" db:"user_uuid"`
	EnvelopeID        *uuid.UUID `json:"envelope_id,omitempty" db:"envelope_id"`
	Name              string     `json:"name" db:"name"`
	AmountE5          int64      `json:"amount_e5" db:"amount_e5"`
	BillingCycle      string     `json:"billing_cycle" db:"billing_cycle"`
	NextBillingDate   time.Time  `json:"next_billing_date" db:"next_billing_date"`
	PaymentMethod     string     `json:"payment_method" db:"payment_method"`
	Status            string     `json:"status" db:"status"`
	AutoRenew         bool       `json:"auto_renew" db:"auto_renew"`
	Notes             string     `json:"notes,omitempty" db:"notes"`
	LastChargedAt     *time.Time `json:"last_charged_at,omitempty" db:"last_charged_at"`
	LastTransactionID *uuid.UUID `json:"last_transaction_id,omitempty" db:"last_transaction_id"`
	MerchantPattern   string     `json:"merchant_pattern,omitempty" db:"merchant_pattern"`
	ChargeWindowHours int        `json:"charge_window_hours" db:"charge_window_hours"`
	OccurrenceCount   int        `json:"occurrence_count" db:"occurrence_count"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

type SubscriptionSummary struct {
	TotalMonthlyCommitmentE5 int64           `json:"total_monthly_commitment_e5"`
	ActiveCount              int             `json:"active_count"`
	PausedCount              int             `json:"paused_count"`
	NextUpcoming             *Subscription   `json:"next_upcoming,omitempty"`
	Subscriptions            []*Subscription `json:"subscriptions"`
}

type SubscriptionRepository interface {
	CreateSubscription(sub *Subscription, tx *sql.Tx) (uuid.UUID, error)
	GetSubscriptionByID(id uuid.UUID) (*Subscription, error)
	GetSubscriptionsByUserUUID(userUUID uuid.UUID) ([]*Subscription, error)
	GetDueSubscriptions(asOf time.Time, tx *sql.Tx) ([]*Subscription, error)
	UpdateSubscription(sub *Subscription, tx *sql.Tx) error
	DeleteSubscription(id uuid.UUID) error
}

func CalculateMonthlyEquivalentE5(amountE5 int64, cycle string) int64 {
	switch cycle {
	case BillingCycleWeekly:
		return amountE5 * 52 / 12
	case BillingCycleQuarterly:
		return amountE5 / 3
	case BillingCycleYearly:
		return amountE5 / 12
	case BillingCycleMonthly:
		fallthrough
	default:
		return amountE5
	}
}

func AdvanceBillingDate(current time.Time, cycle string) time.Time {
	switch cycle {
	case BillingCycleWeekly:
		return current.AddDate(0, 0, 7)
	case BillingCycleQuarterly:
		return current.AddDate(0, 3, 0)
	case BillingCycleYearly:
		return current.AddDate(1, 0, 0)
	case BillingCycleMonthly:
		fallthrough
	default:
		return current.AddDate(0, 1, 0)
	}
}

func (s *Subscription) GetExpectedChargeTime() time.Time {
	if s.LastChargedAt == nil || s.OccurrenceCount == 0 {
		return s.NextBillingDate
	}
	y, m, d := s.NextBillingDate.Date()
	h, min, sec := s.LastChargedAt.Clock()
	return time.Date(y, m, d, h, min, sec, 0, time.UTC)
}

func (s *Subscription) MatchesTransaction(txn *Transaction) bool {
	if s == nil || txn == nil {
		return false
	}
	if s.Status != SubscriptionStatusActive {
		return false
	}
	if txn.Type != TxnTypeDebit {
		return false
	}

	// 1. Amount match (exact or within 10% tolerance for forex/taxes)
	diff := s.AmountE5 - txn.AmountE5
	if diff < 0 {
		diff = -diff
	}
	if s.AmountE5 > 0 && diff > (s.AmountE5/10) {
		return false
	}

	// 2. Merchant / Text match
	desc := strings.ToLower(txn.Description)
	name := strings.ToLower(s.Name)
	textMatched := strings.Contains(desc, name) || strings.Contains(name, desc)
	if !textMatched && s.MerchantPattern != "" {
		pattern := strings.ToLower(s.MerchantPattern)
		textMatched = strings.Contains(desc, pattern) || strings.Contains(pattern, desc)
	}
	if !textMatched {
		return false
	}

	// 3. Time window check
	windowHours := s.ChargeWindowHours
	if windowHours <= 0 {
		windowHours = 48
	}
	window := time.Duration(windowHours) * time.Hour
	expectedTime := s.GetExpectedChargeTime()

	txnTime := txn.CreatedAt
	if txnTime.IsZero() {
		return true
	}

	timeDiff := txnTime.Sub(expectedTime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	return timeDiff <= window
}

func (s *Subscription) RecordCharge(txnID uuid.UUID, chargedAt time.Time, rawDescription string, amountE5 int64) {
	s.LastChargedAt = &chargedAt
	s.LastTransactionID = &txnID
	s.OccurrenceCount++

	if s.OccurrenceCount == 1 {
		s.ChargeWindowHours = 24
	} else if s.OccurrenceCount >= 2 {
		s.ChargeWindowHours = 12
	}

	if s.MerchantPattern == "" && rawDescription != "" {
		s.MerchantPattern = strings.TrimSpace(rawDescription)
	}

	if amountE5 > 0 {
		s.AmountE5 = amountE5
	}

	s.NextBillingDate = AdvanceBillingDate(s.NextBillingDate, s.BillingCycle)
	s.UpdatedAt = chargedAt
}



