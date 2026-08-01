package db

import (
	"database/sql"

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
	_, err := r.db.Exec(query, user.UUID, user.Name)
	return err
}

func (r *pgUserRepo) GetUserByUUID(uuid string) (*core.User, error) {
	query := `
		SELECT uuid, name, created_at, updated_at
		FROM users
		WHERE uuid = $1
	`
	user := &core.User{}
	err := r.db.QueryRow(query, uuid).Scan(
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
