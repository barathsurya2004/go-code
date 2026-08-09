package core

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const (
	AuthToken   = "auth_token"
	DefaultName = "default"
)

type Transaction struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	EnvelopeID *uuid.UUID `json:"envelope_id" db:"envelope_id"` // Nullable if uncategorized yet
	AmountE5   int64      `json:"amount_e5" db:"amount_e5"`
	Type       string     `json:"txn_type" db:"txn_type"`
	BankName   string     `json:"bank_name" db:"bank_name"`
	CountryISO string     `json:"country_iso2" db:"country_iso2"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

type TransactionRepository interface {
	CreateTransaction(txn *Transaction, Tx *sql.Tx) (uuid.UUID, error)
	GetTransactionByUUID(uuid uuid.UUID) (*Transaction, error)
	GetTransactionsByUserUUID(userUUID uuid.UUID) ([]*Transaction, error)
	UpdateTransaction(txn *Transaction) error
	DeleteTransaction(uuid uuid.UUID) error
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
	TargetAmountE5  float64   `json:"target_amount_e5"`
	Cadence         string    `json:"cadence"`
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
	GetActiveAllocationsByUserUUID(userUUID uuid.UUID, targetDate time.Time) ([]*Allocation, error)
	UpdateAllocation(allocation *Allocation) error
	DeleteAllocation(id uuid.UUID) error
}
