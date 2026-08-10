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
		INSERT INTO users (name,email,password_hash)
		VALUES ($1,$2,$3)
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

	var id uuid.UUID
	var row *sql.Row
	if Tx != nil {
		row = Tx.QueryRow(query, user.Name, user.Email, user.PasswordHash)
	} else {
		row = r.db.QueryRow(query, user.Name, user.Email, user.PasswordHash)
	}
	if err := row.Scan(&id); err != nil {
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

func (r *pgUserRepo) GetUserByEmail(email string) (*core.User, error) {
	query := `
		SELECT uuid, name, created_at, updated_at, password_hash, email
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
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}
