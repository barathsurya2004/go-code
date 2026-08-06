package core

import "time"

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
	CreateToken(token *Token) error
	DeleteToken(userUUID string) error
	GetToken(token string) (*Token, error)
	GetTokenWithUserUUID(userUUID string) (*Token, error)
	UpdateToken(token *Token) error
}
