package core

import (
	"time"

	"github.com/google/uuid"
)

const (
	AuthToken   = "auth_token"
	DefaultName = "default"
)

type Transaction struct {
	UUID       string    `json:"uuid"`
	AmountE5   float64   `json:"amount_e5"`
	UserUUID   string    `json:"user_uuid"`
	CountryISO string    `json:"country_iso2"`
	Category   string    `json:"category"`
	BankName   string    `json:"bank_name"`
	Type       string    `json:"txn_type"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type TransactionRepository interface {
	CreateTransaction(txn *Transaction) error
	GetTransactionByUUID(uuid string) (*Transaction, error)
	GetTransactionsByUserUUID(userUUID string) ([]*Transaction, error)
	UpdateTransaction(txn *Transaction) error
	DeleteTransaction(uuid string) error
}

type User struct {
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserRepository interface {
	CreateUser(user *User) error
	GetUserByUUID(uuid string) (*User, error)
}

type Token struct {
	UserUUID   string     `json:"user_uuid"`
	Token      string     `json:"token"`
	Prefix     string     `json:"prefix"`
	Name       string     `json:"name"`
	Scope      []string   `json:"scope"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type TokenRepository interface {
	CreateToken(token *Token) (uuid.UUID, error)
	DeleteToken(userUUID string) error
	GetToken(token string) (*Token, error)
	GetTokenWithUserUUID(userUUID string) (*Token, error)
	UpdateToken(token *Token) error
}

type EnvelopeGroup struct {
	ID        uuid.UUID `json:"id"`
	UserUUID  string    `json:"user_uuid"`
	Name      string    `json:"name"`
	IsSystem  bool      `json:"is_system"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EnvelopeGroupRepository interface {
	CreateEnvelopeGroup(envelopeGroup *EnvelopeGroup) error
	GetEnvelopeGroupByID(id uuid.UUID) (*EnvelopeGroup, error)
	GetEnvelopeGroupsByUserUUID(userUUID string) ([]*EnvelopeGroup, error)
	UpdateEnvelopeGroup(envelopeGroup *EnvelopeGroup) error
	DeleteEnvelopeGroup(id uuid.UUID) error
}

type Envelope struct {
	ID              uuid.UUID `json:"id"`
	UserUUID        string    `json:"user_uuid"`
	EnvelopeGroupID uuid.UUID `json:"envelope_group_id"`
	TargetAmountE5  float64   `json:"target_amount_e5"`
	Cadence         string    `json:"cadence"`
	CountryISO      string    `json:"country_iso2"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	IsSystem        bool      `json:"is_system"`
}

type EnvelopeRepository interface {
	CreateEnvelope(envelope *Envelope) error
	GetEnvelopeByID(id uuid.UUID) (*Envelope, error)
	GetEnvelopesByUserUUID(userUUID string) ([]*Envelope, error)
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
	CreateAllocation(allocation *Allocation) error
	GetAllocationByID(id uuid.UUID) (*Allocation, error)
	GetAllocationsByEnvelopeID(envelopeID uuid.UUID) ([]*Allocation, error)
	GetActiveAllocationsByUserUUID(userUUID string, targetDate time.Time) ([]*Allocation, error)
	UpdateAllocation(allocation *Allocation) error
	DeleteAllocation(id uuid.UUID) error
}
