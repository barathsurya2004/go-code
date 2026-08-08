package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
)

type pgTransactionRowsRepo struct {
	db *sql.DB
}

func NewPgTransactionRowsRepo(db *sql.DB) core.TransactionRepository {
	return &pgTransactionRowsRepo{
		db: db,
	}
}

func (r *pgTransactionRowsRepo) CreateTransaction(txn *core.Transaction, Tx *sql.Tx) (uuid.UUID, error) {
	query := `
		INSERT INTO transactionrows (amount_e5, user_uuid, country_iso2, category, bank_name, txn_type)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING uuid
	`

	//validation checks

	if txn.AmountE5 < 0 {
		return uuid.Nil, errors.New("transaction amount cannot be negative")
	}
	if txn.CountryISO == "" {
		return uuid.Nil, errors.New("transaction country ISO cannot be empty")
	}
	if txn.Category == "" {
		return uuid.Nil, errors.New("transaction category cannot be empty")
	}
	if txn.Type == "" {
		return uuid.Nil, errors.New("transaction type cannot be empty")
	}
	var txnUUID uuid.UUID
	if err := Tx.QueryRow(query,
		txn.AmountE5,
		txn.UserUUID,
		txn.CountryISO,
		txn.Category,
		txn.BankName,
		txn.Type,
	).Scan(&txnUUID); err != nil {
		return uuid.Nil, err
	}
	return txnUUID, nil
}

func (r *pgTransactionRowsRepo) GetTransactionByUUID(id uuid.UUID) (*core.Transaction, error) {
	query := `
		SELECT uuid, amount_e5, user_uuid, country_iso2, category, bank_name, txn_type, created_at, updated_at
		FROM transactionrows
		WHERE uuid = $1
	`
	//validation checks
	if id == uuid.Nil {
		return nil, errors.New("transaction UUID is required")
	}

	txn := &core.Transaction{}
	err := r.db.QueryRowContext(context.Background(), query, id).Scan(
		&txn.UUID,
		&txn.AmountE5,
		&txn.UserUUID,
		&txn.CountryISO,
		&txn.Category,
		&txn.BankName,
		&txn.Type,
		&txn.CreatedAt,
		&txn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return txn, nil
}

func (r *pgTransactionRowsRepo) GetTransactionsByUserUUID(userUUID uuid.UUID) ([]*core.Transaction, error) {
	query := `
		SELECT uuid, amount_e5, user_uuid, country_iso2, category, bank_name, txn_type, created_at, updated_at
		FROM transactionrows
		WHERE user_uuid = $1
	`
	//validation checks
	if userUUID == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}

	rows, err := r.db.QueryContext(context.Background(), query, userUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*core.Transaction
	for rows.Next() {
		txn := &core.Transaction{}
		if err := rows.Scan(
			&txn.UUID,
			&txn.AmountE5,
			&txn.UserUUID,
			&txn.CountryISO,
			&txn.Category,
			&txn.BankName,
			&txn.Type,
			&txn.CreatedAt,
			&txn.UpdatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, txn)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *pgTransactionRowsRepo) UpdateTransaction(txn *core.Transaction) error {
	query := `
		UPDATE transactionrows
		SET amount_e5 = $1, country_iso2 = $2, category = $3, bank_name = $4, txn_type = $5, updated_at = NOW()
		WHERE uuid = $6
	`

	//validation checks
	if txn.UUID == uuid.Nil {
		return errors.New("transaction UUID is required")
	}
	if txn.AmountE5 < 0 {
		return errors.New("transaction amount cannot be negative")
	}
	if txn.CountryISO == "" {
		return errors.New("transaction country ISO cannot be empty")
	}
	if txn.Category == "" {
		return errors.New("transaction category cannot be empty")
	}
	if txn.Type == "" {
		return errors.New("transaction type cannot be empty")
	}

	_, err := r.db.ExecContext(context.Background(), query,
		txn.AmountE5,
		txn.CountryISO,
		txn.Category,
		txn.BankName,
		txn.Type,
		txn.UUID,
	)
	return err
}

func (r *pgTransactionRowsRepo) DeleteTransaction(id uuid.UUID) error {
	query := `
		DELETE FROM transactionrows
		WHERE uuid = $1
	`
	//validation checks
	if id == uuid.Nil {
		return errors.New("transaction UUID is required")
	}

	_, err := r.db.ExecContext(context.Background(), query, id)
	return err
}
