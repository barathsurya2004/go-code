package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func TestPgTokenRepo_CreateToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTokenRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	tokenUUID := uuid.MustParse("87654321-e89b-12d3-a456-426614174000")

	t.Run("Empty UserUUID", func(t *testing.T) {
		_, err := repo.CreateToken(&core.Token{}, nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected 'user UUID is required', got %v", err)
		}
	})

	t.Run("Exec Error", func(t *testing.T) {
		token := &core.Token{
			UserUUID: userUUID,
			Prefix:   "mcp_",
			Name:     "default",
			Scope:    []string{"all"},
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectExec("INSERT INTO user_tokens").
			WithArgs(
				token.UserUUID,
				sqlmock.AnyArg(),
				token.Prefix,
				token.Name,
				pq.Array(token.Scope),
				nil,
				nil,
				nil,
				nil,
			).
			WillReturnError(errors.New("db error"))

		_, err := repo.CreateToken(token, tx)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		token := &core.Token{
			UserUUID:  userUUID,
			Token:     tokenUUID,
			Prefix:    "mcp_",
			Name:      "default",
			Scope:     []string{"all"},
			ExpiresAt: &now,
			CreatedAt: now,
			UpdatedAt: now,
		}
		mock.ExpectBegin()
		tx, _ := db.Begin()
		mock.ExpectExec("INSERT INTO user_tokens").
			WithArgs(
				token.UserUUID,
				token.Token,
				token.Prefix,
				token.Name,
				pq.Array(token.Scope),
				*token.ExpiresAt,
				nil,
				token.CreatedAt,
				token.UpdatedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		id, err := repo.CreateToken(token, tx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if id == uuid.Nil {
			t.Error("expected non-empty token UUID")
		}
	})
}

func TestPgTokenRepo_DeleteToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTokenRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	t.Run("Empty UserUUID", func(t *testing.T) {
		err := repo.DeleteToken(uuid.Nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected 'user UUID is required', got %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM user_tokens").
			WithArgs(userUUID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.DeleteToken(userUUID)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestPgTokenRepo_GetActiveTokenWithUserUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTokenRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	tokenUUID := uuid.MustParse("87654321-e89b-12d3-a456-426614174000")

	t.Run("Empty UserUUID", func(t *testing.T) {
		_, err := repo.GetActiveTokenWithUserUUID(uuid.Nil)
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected 'user UUID is required', got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT user_id, token_uuid").
			WithArgs(userUUID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetActiveTokenWithUserUUID(userUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()

		rows := sqlmock.NewRows([]string{"user_id", "token_uuid", "prefix", "name", "scopes", "expires_at", "last_used_at", "created_at", "updated_at"}).
			AddRow(userUUID, tokenUUID, "mcp_", "default", pq.Array([]string{"all"}), now, now, now, now)

		mock.ExpectQuery("SELECT user_id, token_uuid").
			WithArgs(userUUID).
			WillReturnRows(rows)

		token, err := repo.GetActiveTokenWithUserUUID(userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token.UserUUID != userUUID || token.Token != tokenUUID || len(token.Scope) != 1 || token.Scope[0] != "all" || token.ExpiresAt == nil || token.LastUsedAt == nil {
			t.Errorf("unexpected token returned: %+v", token)
		}
	})
}

func TestPgTokenRepo_GetToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTokenRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	tokenUUID := uuid.MustParse("87654321-e89b-12d3-a456-426614174000")

	t.Run("Empty Token", func(t *testing.T) {
		_, err := repo.GetToken(uuid.Nil)
		if err == nil || err.Error() != "token is required" {
			t.Errorf("expected 'token is required', got %v", err)
		}
	})

	t.Run("Query Error", func(t *testing.T) {
		mock.ExpectQuery("SELECT user_id, token_uuid").
			WithArgs(tokenUUID).
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetToken(tokenUUID)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()

		rows := sqlmock.NewRows([]string{"user_id", "token_uuid", "prefix", "name", "scopes", "expires_at", "last_used_at", "created_at", "updated_at"}).
			AddRow(userUUID, tokenUUID, "mcp_", "default", pq.Array([]string{"all"}), now, now, now, now)

		mock.ExpectQuery("SELECT user_id, token_uuid").
			WithArgs(tokenUUID).
			WillReturnRows(rows)

		mock.ExpectExec("UPDATE user_tokens").
			WithArgs(userUUID, tokenUUID, "mcp_", "default", pq.Array([]string{"all"}), now, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		token, err := repo.GetToken(tokenUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token.UserUUID != userUUID || token.Token != tokenUUID || token.ExpiresAt == nil || token.LastUsedAt == nil {
			t.Errorf("unexpected token returned: %+v", token)
		}
	})
}

func TestPgTokenRepo_UpdateToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error creating sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTokenRepo(db)
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	tokenUUID := uuid.MustParse("87654321-e89b-12d3-a456-426614174000")

	t.Run("Empty UserUUID", func(t *testing.T) {
		err := repo.UpdateToken(&core.Token{})
		if err == nil || err.Error() != "user UUID is required" {
			t.Errorf("expected 'user UUID is required', got %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		now := time.Now()
		token := &core.Token{
			UserUUID:  userUUID,
			Token:     tokenUUID,
			Prefix:    "mcp_",
			Name:      "default",
			Scope:     []string{"all"},
			UpdatedAt: now,
		}
		mock.ExpectExec("UPDATE user_tokens").
			WithArgs(
				token.UserUUID,
				token.Token,
				token.Prefix,
				token.Name,
				pq.Array(token.Scope),
				nil,
				nil,
				token.UpdatedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.UpdateToken(token)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
