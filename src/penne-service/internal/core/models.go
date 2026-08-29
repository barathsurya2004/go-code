package core

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const (
	AuthToken   = "auth_token"
	DefaultName = "default"

	StatusPending = "PENDING"
	StatusSettled = "SETTLED"
	StatusExpired = "EXPIRED"
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
}

type TransactionRepository interface {
	CreateTransaction(txn *Transaction, Tx *sql.Tx) (uuid.UUID, error)
	GetTransactionByUUID(uuid uuid.UUID) (*Transaction, error)
	GetTransactionsByUserUUID(userUUID uuid.UUID) ([]*Transaction, error)
	GetTransactionByTime(time_lowerbound, time_upperbound time.Time, Tx *sql.Tx) (*Transaction, error)
	UpdateTransaction(txn *Transaction, Tx *sql.Tx) error
	DeleteTransaction(uuid uuid.UUID) error
	GetDashboardSummary(uuid uuid.UUID) (*DashboardSummary, error)
}

type User struct {
	UUID         uuid.UUID `json:"uuid"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserRepository interface {
	CreateUser(user *User, Tx *sql.Tx) (uuid.UUID, error)
	GetUserByUUID(uuid uuid.UUID) (*User, error)
	GetUserByEmail(email string) (*User, error)
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
}

type DashboardSummary struct {
	TotalIncomeE5    int64 `json:"total_income_e5"`
	TotalExpenseE5   int64 `json:"total_expense_e5"`
	TotalRemainingE5 int64 `json:"total_remaining_e5"`
	CardSpentE5      int64 `json:"card_spent_e5"`
	CardLimitE5      int64 `json:"card_limit_e5"`
	BankSpentE5      int64 `json:"bank_spent_e5"`
	BankLimitE5      int64 `json:"bank_limit_e5"`
}
