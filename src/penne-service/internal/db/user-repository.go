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

func NewPgUserRepo(db *sql.DB) *pgUserRepo {
	return &pgUserRepo{
		db: db,
	}
}

func (r *pgUserRepo) CreateUser(user *core.User) error {
	query := `
		INSERT INTO users (uuid, name)
		VALUES ($1, $2)
	`

	// validation check
	if user.UUID == "" {
		return errors.New("user UUID is required")
	}

	if strings.TrimSpace(user.Name) == "" {
		return errors.New("user name is required")
	}

	_, err := r.db.Exec(query, user.UUID, user.Name)
	return err
}

func (r *pgUserRepo) GetUserByUUID(id string) (*core.User, error) {
	query := `
		SELECT uuid, name, created_at, updated_at
		FROM users
		WHERE uuid = $1
	`

	// validation checks
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("user UUID is required")
	}
	if err := uuid.Validate(id); err != nil {
		return nil, errors.New("user UUID is invalid")
	}

	user := &core.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.UUID,
		&user.Name,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}
