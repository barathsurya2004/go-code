package db

import (
	"context"
	"database/sql"

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

func (r *pgTransactionRowsRepo) CreateTransaction(txn *core.Transaction) error {
	query := `
		INSERT INTO transactionrows (uuid, amount_e5, user_uuid, country_iso2, category, bank_name, txn_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(context.Background(), query,
		txn.UUID,
		txn.AmountE5,
		txn.UserUUID,
		txn.CountryISO,
		txn.Category,
		txn.BankName,
		txn.Type,
	)
	return err
}

func (r *pgTransactionRowsRepo) GetTransactionByUUID(uuid string) (*core.Transaction, error) {
	query := `
		SELECT uuid, amount_e5, user_uuid, country_iso2, category, bank_name, txn_type, created_at, updated_at
		FROM transactionrows
		WHERE uuid = $1
	`
	txn := &core.Transaction{}
	err := r.db.QueryRowContext(context.Background(), query, uuid).Scan(
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

func (r *pgTransactionRowsRepo) GetTransactionsByUserUUID(userUUID string) ([]*core.Transaction, error) {
	query := `
		SELECT uuid, amount_e5, user_uuid, country_iso2, category, bank_name, txn_type, created_at, updated_at
		FROM transactionrows
		WHERE user_uuid = $1
	`
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

func (r *pgTransactionRowsRepo) DeleteTransaction(uuid string) error {
	query := `
		DELETE FROM transactionrows
		WHERE uuid = $1
	`
	_, err := r.db.ExecContext(context.Background(), query, uuid)
	return err
}
