package activities

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockUserRepo struct {
	core.UserRepository
	createFn func(user *core.User, Tx *sql.Tx) (uuid.UUID, error)
}

func (m *mockUserRepo) CreateUser(user *core.User, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(user, Tx)
	}
	return uuid.Nil, nil
}

type mockEnvelopeGroupRepo struct {
	core.EnvelopeGroupRepository
	createFn func(envelopeGroup *core.EnvelopeGroup, Tx *sql.Tx) (uuid.UUID, error)
}

func (m *mockEnvelopeGroupRepo) CreateEnvelopeGroup(envelopeGroup *core.EnvelopeGroup, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(envelopeGroup, Tx)
	}
	return uuid.Nil, nil
}

type mockEnvelopeRepo struct {
	core.EnvelopeRepository
	createFn func(envelope *core.Envelope, Tx *sql.Tx) (uuid.UUID, error)
}

func (m *mockEnvelopeRepo) CreateEnvelope(envelope *core.Envelope, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(envelope, Tx)
	}
	return uuid.Nil, nil
}

type mockAllocationRepo struct {
	core.AllocationRepository
	createFn func(allocation *core.Allocation, Tx *sql.Tx) (uuid.UUID, error)
}

func (m *mockAllocationRepo) CreateAllocation(allocation *core.Allocation, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(allocation, Tx)
	}
	return uuid.Nil, nil
}

type mockTokenRepo struct {
	core.TokenRepository
	createFn func(token *core.Token, Tx *sql.Tx) (uuid.UUID, error)
}

func (m *mockTokenRepo) CreateToken(token *core.Token, Tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(token, Tx)
	}
	return uuid.Nil, nil
}

func TestUserActivities(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("CreateUserActivity - Success", func(t *testing.T) {
		expectedID := uuid.New()
		userRepo := &mockUserRepo{
			createFn: func(user *core.User, Tx *sql.Tx) (uuid.UUID, error) {
				return expectedID, nil
			},
		}
		act := NewUserActivities(core.RepoContainer{User: userRepo}, logger)
		id, err := act.CreateUserActivity(ctx, core.User{Name: "Alice"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if *id != expectedID {
			t.Fatalf("expected ID %v, got %v", expectedID, *id)
		}
	})

	t.Run("CreateUserActivity - Error", func(t *testing.T) {
		userRepo := &mockUserRepo{
			createFn: func(user *core.User, Tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("db error")
			},
		}
		act := NewUserActivities(core.RepoContainer{User: userRepo}, logger)
		_, err := act.CreateUserActivity(ctx, core.User{Name: "Alice"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CreateSystemEnvelopeGroupActivity - Success", func(t *testing.T) {
		expectedID := uuid.New()
		envGroupRepo := &mockEnvelopeGroupRepo{
			createFn: func(g *core.EnvelopeGroup, Tx *sql.Tx) (uuid.UUID, error) {
				if g.Name != "Unallocated Budget" || !g.IsSystem {
					t.Errorf("unexpected envelope group attributes: %+v", g)
				}
				return expectedID, nil
			},
		}
		act := NewUserActivities(core.RepoContainer{EnvelopeGroup: envGroupRepo}, logger)
		id, err := act.CreateSystemEnvelopeGroupActivity(ctx, uuid.New())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if *id != expectedID {
			t.Fatalf("expected ID %v, got %v", expectedID, *id)
		}
	})

	t.Run("CreateSystemEnvelopeGroupActivity - Error", func(t *testing.T) {
		envGroupRepo := &mockEnvelopeGroupRepo{
			createFn: func(g *core.EnvelopeGroup, Tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("group creation error")
			},
		}
		act := NewUserActivities(core.RepoContainer{EnvelopeGroup: envGroupRepo}, logger)
		_, err := act.CreateSystemEnvelopeGroupActivity(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CreateSystemEnvelopeActivity - Success", func(t *testing.T) {
		expectedID := uuid.New()
		userUUID := uuid.New()
		groupUUID := uuid.New()
		envRepo := &mockEnvelopeRepo{
			createFn: func(e *core.Envelope, Tx *sql.Tx) (uuid.UUID, error) {
				if e.UserUUID != userUUID || e.EnvelopeGroupID != groupUUID || !e.IsSystem {
					t.Errorf("unexpected envelope attributes: %+v", e)
				}
				return expectedID, nil
			},
		}
		act := NewUserActivities(core.RepoContainer{Envelope: envRepo}, logger)
		id, err := act.CreateSystemEnvelopeActivity(ctx, core.CreateSystemEnvelopeActivityInput{
			UserUUID:        userUUID,
			EnvelopeGroupID: groupUUID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if *id != expectedID {
			t.Fatalf("expected ID %v, got %v", expectedID, *id)
		}
	})

	t.Run("CreateSystemEnvelopeActivity - Error", func(t *testing.T) {
		envRepo := &mockEnvelopeRepo{
			createFn: func(e *core.Envelope, Tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("env create error")
			},
		}
		act := NewUserActivities(core.RepoContainer{Envelope: envRepo}, logger)
		_, err := act.CreateSystemEnvelopeActivity(ctx, core.CreateSystemEnvelopeActivityInput{
			UserUUID:        uuid.New(),
			EnvelopeGroupID: uuid.New(),
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CreateDefaultAllocationActivity - Success", func(t *testing.T) {
		expectedID := uuid.New()
		envUUID := uuid.New()
		allocRepo := &mockAllocationRepo{
			createFn: func(a *core.Allocation, Tx *sql.Tx) (uuid.UUID, error) {
				if a.EnvelopeID != envUUID || a.AllocatedAmountE5 != 0 {
					t.Errorf("unexpected allocation attributes: %+v", a)
				}
				return expectedID, nil
			},
		}
		act := NewUserActivities(core.RepoContainer{Allocation: allocRepo}, logger)
		id, err := act.CreateDefaultAllocationActivity(ctx, envUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if *id != expectedID {
			t.Fatalf("expected ID %v, got %v", expectedID, *id)
		}
	})

	t.Run("CreateDefaultAllocationActivity - Error", func(t *testing.T) {
		allocRepo := &mockAllocationRepo{
			createFn: func(a *core.Allocation, Tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("alloc error")
			},
		}
		act := NewUserActivities(core.RepoContainer{Allocation: allocRepo}, logger)
		_, err := act.CreateDefaultAllocationActivity(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CreateUserTokenActivity - Success", func(t *testing.T) {
		expectedID := uuid.New()
		userUUID := uuid.New()
		tokenRepo := &mockTokenRepo{
			createFn: func(token *core.Token, Tx *sql.Tx) (uuid.UUID, error) {
				if token.UserUUID != userUUID || token.Prefix != core.AuthToken {
					t.Errorf("unexpected token attributes: %+v", token)
				}
				return expectedID, nil
			},
		}
		act := NewUserActivities(core.RepoContainer{Token: tokenRepo}, logger)
		id, err := act.CreateUserTokenActivity(ctx, userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if *id != expectedID {
			t.Fatalf("expected ID %v, got %v", expectedID, *id)
		}
	})

	t.Run("CreateUserTokenActivity - Error", func(t *testing.T) {
		tokenRepo := &mockTokenRepo{
			createFn: func(token *core.Token, Tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("token error")
			},
		}
		act := NewUserActivities(core.RepoContainer{Token: tokenRepo}, logger)
		_, err := act.CreateUserTokenActivity(ctx, uuid.New())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
