package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/barathsurya2004/go-code/penne-service/internal/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestAuthServiceHandler_Login(t *testing.T) {
	log := zap.NewNop()
	mockDB, _, _ := sqlmock.New()
	defer mockDB.Close()

	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	tokenUUID := uuid.MustParse("87654321-e89b-12d3-a456-426614174000")
	hashedPassword, _ := utils.CreatePassword("secret123")

	t.Run("Invalid Payload", func(t *testing.T) {
		h := NewAuthServiceHandler(&mockUserRepo{}, &mockTokenRepo{}, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte("invalid json")))
		rr := httptest.NewRecorder()
		h.Login(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("User Not Found", func(t *testing.T) {
		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) {
				return nil, errors.New("user not found")
			},
		}
		h := NewAuthServiceHandler(userRepo, &mockTokenRepo{}, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte(`{"email":"nobody@example.com","password":"secret"}`)))
		rr := httptest.NewRecorder()
		h.Login(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Wrong Password", func(t *testing.T) {
		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) {
				return &core.User{
					UUID:         userUUID,
					Email:        email,
					PasswordHash: hashedPassword,
				}, nil
			},
		}
		h := NewAuthServiceHandler(userRepo, &mockTokenRepo{}, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte(`{"email":"alice@example.com","password":"wrongpassword"}`)))
		rr := httptest.NewRecorder()
		h.Login(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Token Repo Error", func(t *testing.T) {
		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) {
				return &core.User{
					UUID:         userUUID,
					Email:        email,
					PasswordHash: hashedPassword,
				}, nil
			},
		}
		tokenRepo := &mockTokenRepo{
			getActiveTokenWithUserUUIDFn: func(userUUID uuid.UUID) (*core.Token, error) {
				return nil, errors.New("token err")
			},
		}
		h := NewAuthServiceHandler(userRepo, tokenRepo, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte(`{"email":"alice@example.com","password":"secret123"}`)))
		rr := httptest.NewRecorder()
		h.Login(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Success", func(t *testing.T) {
		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) {
				return &core.User{
					UUID:         userUUID,
					Email:        email,
					PasswordHash: hashedPassword,
				}, nil
			},
		}
		tokenRepo := &mockTokenRepo{
			getActiveTokenWithUserUUIDFn: func(userUUID uuid.UUID) (*core.Token, error) {
				return &core.Token{UserUUID: userUUID, Token: tokenUUID}, nil
			},
		}
		h := NewAuthServiceHandler(userRepo, tokenRepo, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte(`{"email":"alice@example.com","password":"secret123"}`)))
		rr := httptest.NewRecorder()
		h.Login(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})
}

func TestAuthServiceHandler_SignUp(t *testing.T) {
	log := zap.NewNop()
	userUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	tokenUUID := uuid.MustParse("87654321-e89b-12d3-a456-426614174000")
	hashedPassword, _ := utils.CreatePassword("secret123")

	t.Run("Invalid Payload", func(t *testing.T) {
		mockDB, _, _ := sqlmock.New()
		defer mockDB.Close()
		h := NewAuthServiceHandler(&mockUserRepo{}, &mockTokenRepo{}, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte("invalid json")))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("New User SignUp BeginTx Error", func(t *testing.T) {
		mockDB, mock, _ := sqlmock.New()
		defer mockDB.Close()
		mock.ExpectBegin().WillReturnError(errors.New("tx error"))

		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) {
				return nil, errors.New("not found")
			},
		}
		h := NewAuthServiceHandler(userRepo, &mockTokenRepo{}, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte(`{"email":"new@example.com","name":"New User","password":"secret123"}`)))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("New User SignUp CreateUser Error", func(t *testing.T) {
		mockDB, mock, _ := sqlmock.New()
		defer mockDB.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) {
				return nil, errors.New("not found")
			},
			createUserFn: func(user *core.User) (uuid.UUID, error) {
				return uuid.Nil, errors.New("create user error")
			},
		}
		h := NewAuthServiceHandler(userRepo, &mockTokenRepo{}, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte(`{"email":"new@example.com","name":"New User","password":"secret123"}`)))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("New User SignUp CreateEnvelopeGroup Error", func(t *testing.T) {
		mockDB, mock, _ := sqlmock.New()
		defer mockDB.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) { return nil, errors.New("not found") },
			createUserFn:     func(user *core.User) (uuid.UUID, error) { return userUUID, nil },
		}
		envGroupRepo := &mockEnvelopeGroupRepo{
			createFn: func(envelopeGroup *core.EnvelopeGroup) (uuid.UUID, error) {
				return uuid.Nil, errors.New("env group error")
			},
		}
		h := NewAuthServiceHandler(userRepo, &mockTokenRepo{}, envGroupRepo, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte(`{"email":"new@example.com","name":"New User","password":"secret123"}`)))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("New User SignUp CreateEnvelope Error", func(t *testing.T) {
		mockDB, mock, _ := sqlmock.New()
		defer mockDB.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) { return nil, errors.New("not found") },
			createUserFn:     func(user *core.User) (uuid.UUID, error) { return userUUID, nil },
		}
		envGroupRepo := &mockEnvelopeGroupRepo{
			createFn: func(envelopeGroup *core.EnvelopeGroup) (uuid.UUID, error) { return uuid.New(), nil },
		}
		envRepo := &mockEnvelopeRepo{
			createFn: func(envelope *core.Envelope) (uuid.UUID, error) { return uuid.Nil, errors.New("env err") },
		}
		h := NewAuthServiceHandler(userRepo, &mockTokenRepo{}, envGroupRepo, envRepo, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte(`{"email":"new@example.com","name":"New User","password":"secret123"}`)))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("New User SignUp CreateAllocation Error", func(t *testing.T) {
		mockDB, mock, _ := sqlmock.New()
		defer mockDB.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) { return nil, errors.New("not found") },
			createUserFn:     func(user *core.User) (uuid.UUID, error) { return userUUID, nil },
		}
		envGroupRepo := &mockEnvelopeGroupRepo{
			createFn: func(envelopeGroup *core.EnvelopeGroup) (uuid.UUID, error) { return uuid.New(), nil },
		}
		envRepo := &mockEnvelopeRepo{
			createFn: func(envelope *core.Envelope) (uuid.UUID, error) { return uuid.New(), nil },
		}
		allocRepo := &mockAllocationRepo{
			createFn: func(allocation *core.Allocation) (uuid.UUID, error) { return uuid.Nil, errors.New("alloc err") },
		}
		h := NewAuthServiceHandler(userRepo, &mockTokenRepo{}, envGroupRepo, envRepo, allocRepo, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte(`{"email":"new@example.com","name":"New User","password":"secret123"}`)))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("New User SignUp CreateToken Error", func(t *testing.T) {
		mockDB, mock, _ := sqlmock.New()
		defer mockDB.Close()
		mock.ExpectBegin()
		mock.ExpectRollback()

		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) { return nil, errors.New("not found") },
			createUserFn:     func(user *core.User) (uuid.UUID, error) { return userUUID, nil },
		}
		envGroupRepo := &mockEnvelopeGroupRepo{
			createFn: func(envelopeGroup *core.EnvelopeGroup) (uuid.UUID, error) { return uuid.New(), nil },
		}
		envRepo := &mockEnvelopeRepo{
			createFn: func(envelope *core.Envelope) (uuid.UUID, error) { return uuid.New(), nil },
		}
		allocRepo := &mockAllocationRepo{
			createFn: func(allocation *core.Allocation) (uuid.UUID, error) { return uuid.New(), nil },
		}
		tokenRepo := &mockTokenRepo{
			createTokenFn: func(token *core.Token) (uuid.UUID, error) { return uuid.Nil, errors.New("token err") },
		}
		h := NewAuthServiceHandler(userRepo, tokenRepo, envGroupRepo, envRepo, allocRepo, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte(`{"email":"new@example.com","name":"New User","password":"secret123"}`)))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("New User SignUp Success", func(t *testing.T) {
		mockDB, mock, _ := sqlmock.New()
		defer mockDB.Close()
		mock.ExpectBegin()
		mock.ExpectCommit()

		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) {
				return nil, errors.New("user not found")
			},
			createUserFn: func(user *core.User) (uuid.UUID, error) {
				return userUUID, nil
			},
		}
		tokenRepo := &mockTokenRepo{
			createTokenFn: func(token *core.Token) (uuid.UUID, error) {
				return tokenUUID, nil
			},
		}
		h := NewAuthServiceHandler(userRepo, tokenRepo, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte(`{"email":"new@example.com","name":"New User","password":"secret123","country_iso2":"US"}`)))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusOK {
			body, _ := io.ReadAll(rr.Body)
			t.Errorf("expected status 200, got %d body=%s", rr.Code, string(body))
		}
	})

	t.Run("Existing User SignUp Wrong Password", func(t *testing.T) {
		mockDB, _, _ := sqlmock.New()
		defer mockDB.Close()

		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) {
				return &core.User{
					UUID:         userUUID,
					Email:        email,
					PasswordHash: hashedPassword,
				}, nil
			},
		}
		h := NewAuthServiceHandler(userRepo, &mockTokenRepo{}, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte(`{"email":"existing@example.com","name":"User","password":"wrongpassword"}`)))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})

	t.Run("Existing User SignUp GetToken Error", func(t *testing.T) {
		mockDB, _, _ := sqlmock.New()
		defer mockDB.Close()

		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) {
				return &core.User{
					UUID:         userUUID,
					Email:        email,
					PasswordHash: hashedPassword,
				}, nil
			},
		}
		tokenRepo := &mockTokenRepo{
			getActiveTokenWithUserUUIDFn: func(userUUID uuid.UUID) (*core.Token, error) {
				return nil, errors.New("token err")
			},
		}
		h := NewAuthServiceHandler(userRepo, tokenRepo, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte(`{"email":"existing@example.com","name":"User","password":"secret123"}`)))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Existing User SignUp Success", func(t *testing.T) {
		mockDB, _, _ := sqlmock.New()
		defer mockDB.Close()

		userRepo := &mockUserRepo{
			getUserByEmailFn: func(email string) (*core.User, error) {
				return &core.User{
					UUID:         userUUID,
					Email:        email,
					PasswordHash: hashedPassword,
				}, nil
			},
		}
		tokenRepo := &mockTokenRepo{
			getActiveTokenWithUserUUIDFn: func(userUUID uuid.UUID) (*core.Token, error) {
				return &core.Token{UserUUID: userUUID, Token: tokenUUID}, nil
			},
		}
		h := NewAuthServiceHandler(userRepo, tokenRepo, &mockEnvelopeGroupRepo{}, &mockEnvelopeRepo{}, &mockAllocationRepo{}, log, mockDB)
		req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader([]byte(`{"email":"existing@example.com","name":"User","password":"secret123"}`)))
		rr := httptest.NewRecorder()
		h.SignUp(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})
}
