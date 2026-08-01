package core

type Transaction struct {
	UUID       string  `json:"uuid"`
	AmountE5   float64 `json:"amount_e5"`
	UserUUID   string  `json:"user_uuid"`
	CountryISO string  `json:"country_iso2"`
	Category   string  `json:"category"`
	BankName   string  `json:"bank_name"`
	Type       string  `json:"txn_type"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type TransactionRepository interface {
	CreateTransaction(txn *Transaction) error
	GetTransactionByUUID(uuid string) (*Transaction, error)
	GetTransactionsByUserUUID(userUUID string) ([]*Transaction, error)
	UpdateTransaction(txn *Transaction) error
	DeleteTransaction(uuid string) error
}

type User struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type UserRepository interface {
	CreateUser(user *User) error
	GetUserByUUID(uuid string) (*User, error)
}
