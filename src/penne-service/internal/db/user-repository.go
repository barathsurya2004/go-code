package db

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
)

type pgUserRepo struct {
	db *sql.DB
}

func NewPgUserRepo(db *sql.DB) core.UserRepository {
	return &pgUserRepo{
		db: db,
	}
}

func (r *pgUserRepo) CreateUser(user *core.User, Tx *sql.Tx) (uuid.UUID, error) {
	query := `
		INSERT INTO users (name, email, password_hash, monthly_budget_e5, salary_day)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING uuid
	`

	if strings.TrimSpace(user.Name) == "" {
		return uuid.Nil, errors.New("user name is required")
	}
	if strings.TrimSpace(user.Email) == "" {
		return uuid.Nil, errors.New("user email is required")
	}
	if strings.TrimSpace(user.PasswordHash) == "" {
		return uuid.Nil, errors.New("user password hash is required")
	}
	if user.SalaryDay <= 0 {
		user.SalaryDay = 1
	}

	var id uuid.UUID
	var row *sql.Row
	if Tx != nil {
		row = Tx.QueryRow(query, user.Name, user.Email, user.PasswordHash, user.MonthlyBudgetE5, user.SalaryDay)
	} else {
		row = r.db.QueryRow(query, user.Name, user.Email, user.PasswordHash, user.MonthlyBudgetE5, user.SalaryDay)
	}
	if err := row.Scan(&id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *pgUserRepo) GetUserByUUID(id uuid.UUID) (*core.User, error) {
	query := `
		SELECT uuid, name, created_at, updated_at, monthly_budget_e5, salary_day
		FROM users
		WHERE uuid = $1
	`

	// validation checks
	if id == uuid.Nil {
		return nil, errors.New("user UUID is required")
	}

	user := &core.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.UUID,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.MonthlyBudgetE5,
		&user.SalaryDay,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *pgUserRepo) GetUserByEmail(email string) (*core.User, error) {
	query := `
		SELECT uuid, name, created_at, updated_at, password_hash, email, monthly_budget_e5, salary_day
		FROM users
		WHERE email = $1
	`
	user := &core.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.UUID,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.PasswordHash,
		&user.Email,
		&user.MonthlyBudgetE5,
		&user.SalaryDay,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *pgUserRepo) UpdateBudgetSettings(userUUID uuid.UUID, monthlyBudgetE5 int64, salaryDay int, Tx *sql.Tx) error {
	if userUUID == uuid.Nil {
		return errors.New("user UUID is required")
	}
	if monthlyBudgetE5 < 0 {
		return errors.New("monthly budget cannot be negative")
	}
	if salaryDay < 1 || salaryDay > 31 {
		return errors.New("salary day must be between 1 and 31")
	}

	query := `
		UPDATE users
		SET monthly_budget_e5 = $1, salary_day = $2, updated_at = NOW()
		WHERE uuid = $3
	`

	var err error
	if Tx != nil {
		_, err = Tx.Exec(query, monthlyBudgetE5, salaryDay, userUUID)
	} else {
		_, err = r.db.Exec(query, monthlyBudgetE5, salaryDay, userUUID)
	}
	return err
}
