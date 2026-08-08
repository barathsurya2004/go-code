package db

import (
	"database/sql"
	"errors"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type pgTokenRepo struct {
	db *sql.DB
}

func NewTokenRepo(db *sql.DB) core.TokenRepository {
	return &pgTokenRepo{
		db: db,
	}
}

func nullTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func (r *pgTokenRepo) CreateToken(token *core.Token) (uuid.UUID, error) {
	if token.UserUUID == "" {
		return uuid.Nil, errors.New("user UUID is required")
	}
	if token.Token == "" {
		token.Token = uuid.NewString()
	}

	query := `
		INSERT INTO user_tokens (user_id, token_uuid, prefix, name, scopes, expires_at, last_used_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, NOW()), COALESCE($9, NOW()))
	`
	_, err := r.db.Exec(
		query,
		token.UserUUID,
		token.Token,
		token.Prefix,
		token.Name,
		pq.Array(token.Scope),
		nullTimePtr(token.ExpiresAt),
		nullTimePtr(token.LastUsedAt),
		nullTime(token.CreatedAt),
		nullTime(token.UpdatedAt),
	)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(token.Token)

}

func (r *pgTokenRepo) DeleteToken(userUUID string) error {
	if userUUID == "" {
		return errors.New("user UUID is required")
	}
	query := `
		DELETE FROM user_tokens WHERE user_id = $1
	`
	_, err := r.db.Exec(query, userUUID)
	return err
}

func (r *pgTokenRepo) GetTokenWithUserUUID(userUUID string) (*core.Token, error) {
	if userUUID == "" {
		return nil, errors.New("user UUID is required")
	}

	query := `
		SELECT user_id, token_uuid, prefix, name, scopes, expires_at, last_used_at, created_at, updated_at
		FROM user_tokens
		WHERE user_id = $1
	`
	token := &core.Token{}
	var expiresAt, lastUsedAt sql.NullTime

	err := r.db.QueryRow(query, userUUID).Scan(
		&token.UserUUID,
		&token.Token,
		&token.Prefix,
		&token.Name,
		pq.Array(&token.Scope),
		&expiresAt,
		&lastUsedAt,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		token.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		token.LastUsedAt = &lastUsedAt.Time
	}

	return token, nil
}

func (r *pgTokenRepo) GetToken(token string) (*core.Token, error) {
	if token == "" {
		return nil, errors.New("token is required")
	}

	query := `
		SELECT user_id, token_uuid, prefix, name, scopes, expires_at, last_used_at, created_at, updated_at
		FROM user_tokens
		WHERE token_uuid = $1
	`
	tokenObj := &core.Token{}
	var expiresAt, lastUsedAt sql.NullTime

	err := r.db.QueryRow(query, token).Scan(
		&tokenObj.UserUUID,
		&tokenObj.Token,
		&tokenObj.Prefix,
		&tokenObj.Name,
		pq.Array(&tokenObj.Scope),
		&expiresAt,
		&lastUsedAt,
		&tokenObj.CreatedAt,
		&tokenObj.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		tokenObj.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		tokenObj.LastUsedAt = &lastUsedAt.Time
	}

	timeNow := time.Now()
	tokenObj.LastUsedAt = &timeNow
	r.UpdateToken(tokenObj)

	return tokenObj, nil
}

func (r *pgTokenRepo) UpdateToken(token *core.Token) error {
	if token.UserUUID == "" {
		return errors.New("user UUID is required")
	}

	query := `
		UPDATE user_tokens
		SET 
		token_uuid = $2,
		prefix = $3,
		name = $4,
		scopes = $5,
		expires_at = $6,
		last_used_at = $7,
		updated_at = COALESCE($8, NOW())
		WHERE user_id = $1
	`
	_, err := r.db.Exec(
		query,
		token.UserUUID,
		token.Token,
		token.Prefix,
		token.Name,
		pq.Array(token.Scope),
		nullTimePtr(token.ExpiresAt),
		nullTimePtr(token.LastUsedAt),
		nullTime(token.UpdatedAt),
	)
	return err
}
