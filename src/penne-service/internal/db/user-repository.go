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
		INSERT INTO users (name)
		VALUES ($1)
		RETURNING uuid
	`

	if strings.TrimSpace(user.Name) == "" {
		return uuid.Nil, errors.New("user name iss required")
	}

	var id uuid.UUID
	if err := Tx.QueryRow(query, user.Name).Scan(&id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *pgUserRepo) GetUserByUUID(id uuid.UUID) (*core.User, error) {
	query := `
		SELECT uuid, name, created_at, updated_at
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
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}
