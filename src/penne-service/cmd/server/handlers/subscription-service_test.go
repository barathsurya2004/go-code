package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
)

type mockSubscriptionRepoInHandlers struct {
	core.SubscriptionRepository
	createFn  func(*core.Subscription, *sql.Tx) (uuid.UUID, error)
	getByIDFn func(uuid.UUID) (*core.Subscription, error)
	getByUser func(uuid.UUID) ([]*core.Subscription, error)
	getDueFn  func(time.Time, *sql.Tx) ([]*core.Subscription, error)
	updateFn  func(*core.Subscription, *sql.Tx) error
	deleteFn  func(uuid.UUID) error
}

func (m *mockSubscriptionRepoInHandlers) CreateSubscription(sub *core.Subscription, tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(sub, tx)
	}
	return uuid.New(), nil
}

func (m *mockSubscriptionRepoInHandlers) GetSubscriptionByID(id uuid.UUID) (*core.Subscription, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}

func (m *mockSubscriptionRepoInHandlers) GetSubscriptionsByUserUUID(userUUID uuid.UUID) ([]*core.Subscription, error) {
	if m.getByUser != nil {
		return m.getByUser(userUUID)
	}
	return nil, nil
}

func (m *mockSubscriptionRepoInHandlers) GetDueSubscriptions(asOf time.Time, tx *sql.Tx) ([]*core.Subscription, error) {
	if m.getDueFn != nil {
		return m.getDueFn(asOf, tx)
	}
	return nil, nil
}

func (m *mockSubscriptionRepoInHandlers) UpdateSubscription(sub *core.Subscription, tx *sql.Tx) error {
	if m.updateFn != nil {
		return m.updateFn(sub, tx)
	}
	return nil
}

func (m *mockSubscriptionRepoInHandlers) DeleteSubscription(id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}

type mockTxnRepoInSubHandlers struct {
	core.TransactionRepository
	createTxnFn func(*core.Transaction, *sql.Tx) (uuid.UUID, error)
}

func (m *mockTxnRepoInSubHandlers) CreateTransaction(txn *core.Transaction, tx *sql.Tx) (uuid.UUID, error) {
	if m.createTxnFn != nil {
		return m.createTxnFn(txn, tx)
	}
	return uuid.New(), nil
}

func TestSubscriptionServiceHandler_CreateSubscription(t *testing.T) {
	logger := zap.NewNop()
	userUUID := uuid.New()

	t.Run("Missing user UUID", func(t *testing.T) {
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, nil)
		req := httptest.NewRequest("POST", "/subscription", bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()
		h.CreateSubscription(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid payload JSON", func(t *testing.T) {
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, nil)
		req := httptest.NewRequest("POST", "/subscription?user_uuid="+userUUID.String(), bytes.NewBufferString(`invalid`))
		w := httptest.NewRecorder()
		h.CreateSubscription(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Success without Cadence", func(t *testing.T) {
		expectedID := uuid.New()
		mockRepo := &mockSubscriptionRepoInHandlers{
			createFn: func(sub *core.Subscription, tx *sql.Tx) (uuid.UUID, error) {
				return expectedID, nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("POST", "/subscription?user_uuid="+userUUID.String(), bytes.NewBufferString(`{"name":"Netflix","amount_e5":64900000}`))
		w := httptest.NewRecorder()
		h.CreateSubscription(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp core.Subscription
		json.NewDecoder(w.Body).Decode(&resp)
		assert.Equal(t, expectedID, resp.ID)
		assert.Equal(t, "Netflix", resp.Name)
	})

	t.Run("Repo error without Cadence", func(t *testing.T) {
		mockRepo := &mockSubscriptionRepoInHandlers{
			createFn: func(sub *core.Subscription, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.Nil, errors.New("repo insert error")
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("POST", "/subscription?user_uuid="+userUUID.String(), bytes.NewBufferString(`{"name":"Netflix","amount_e5":64900000}`))
		w := httptest.NewRecorder()
		h.CreateSubscription(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Success with Cadence", func(t *testing.T) {
		expectedID := uuid.New()
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						ptr := valuePtr.(*uuid.UUID)
						*ptr = expectedID
						return nil
					},
				}, nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, cc)
		req := httptest.NewRequest("POST", "/subscription?user_uuid="+userUUID.String(), bytes.NewBufferString(`{"name":"Netflix","amount_e5":64900000}`))
		w := httptest.NewRecorder()
		h.CreateSubscription(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Cadence ExecuteWorkflow error", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return nil, errors.New("cadence down")
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, cc)
		req := httptest.NewRequest("POST", "/subscription?user_uuid="+userUUID.String(), bytes.NewBufferString(`{"name":"Netflix","amount_e5":64900000}`))
		w := httptest.NewRecorder()
		h.CreateSubscription(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Cadence Workflow execution result error", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						return errors.New("activity validation failed")
					},
				}, nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, cc)
		req := httptest.NewRequest("POST", "/subscription?user_uuid="+userUUID.String(), bytes.NewBufferString(`{"name":"Netflix","amount_e5":64900000}`))
		w := httptest.NewRecorder()
		h.CreateSubscription(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSubscriptionServiceHandler_GetSubscriptions(t *testing.T) {
	logger := zap.NewNop()
	userUUID := uuid.New()
	now := time.Now()

	t.Run("Missing user UUID", func(t *testing.T) {
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, nil)
		req := httptest.NewRequest("GET", "/subscriptions", nil)
		w := httptest.NewRecorder()
		h.GetSubscriptions(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Repo error", func(t *testing.T) {
		mockRepo := &mockSubscriptionRepoInHandlers{
			getByUser: func(u uuid.UUID) ([]*core.Subscription, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("GET", "/subscriptions?user_uuid="+userUUID.String(), nil)
		w := httptest.NewRecorder()
		h.GetSubscriptions(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Success with calculations (empty list)", func(t *testing.T) {
		mockRepo := &mockSubscriptionRepoInHandlers{
			getByUser: func(u uuid.UUID) ([]*core.Subscription, error) {
				return nil, nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("GET", "/subscriptions?user_uuid="+userUUID.String(), nil)
		w := httptest.NewRecorder()
		h.GetSubscriptions(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var summary core.SubscriptionSummary
		json.NewDecoder(w.Body).Decode(&summary)
		assert.Equal(t, 0, summary.ActiveCount)
		assert.Equal(t, int64(0), summary.TotalMonthlyCommitmentE5)
	})

	t.Run("Success with active, paused, and upcoming", func(t *testing.T) {
		subs := []*core.Subscription{
			{
				ID:              uuid.New(),
				Name:            "Netflix",
				AmountE5:        64900000,
				BillingCycle:    core.BillingCycleMonthly,
				Status:          core.SubscriptionStatusActive,
				NextBillingDate: now.Add(48 * time.Hour),
			},
			{
				ID:              uuid.New(),
				Name:            "Amazon Prime",
				AmountE5:        149900000,
				BillingCycle:    core.BillingCycleYearly,
				Status:          core.SubscriptionStatusActive,
				NextBillingDate: now.Add(24 * time.Hour),
			},
			{
				ID:           uuid.New(),
				Name:         "Gym",
				AmountE5:     250000000,
				BillingCycle: core.BillingCycleMonthly,
				Status:       core.SubscriptionStatusPaused,
			},
		}

		mockRepo := &mockSubscriptionRepoInHandlers{
			getByUser: func(u uuid.UUID) ([]*core.Subscription, error) {
				return subs, nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("GET", "/subscriptions?user_uuid="+userUUID.String(), nil)
		w := httptest.NewRecorder()
		h.GetSubscriptions(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var summary core.SubscriptionSummary
		json.NewDecoder(w.Body).Decode(&summary)
		assert.Equal(t, 2, summary.ActiveCount)
		assert.Equal(t, 1, summary.PausedCount)
		assert.NotNil(t, summary.NextUpcoming)
		assert.Equal(t, "Amazon Prime", summary.NextUpcoming.Name)
	})
}

func TestSubscriptionServiceHandler_GetSubscriptionByID(t *testing.T) {
	logger := zap.NewNop()
	subID := uuid.New()

	t.Run("Missing ID", func(t *testing.T) {
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, nil)
		req := httptest.NewRequest("GET", "/subscription", nil)
		w := httptest.NewRecorder()
		h.GetSubscriptionByID(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, nil)
		req := httptest.NewRequest("GET", "/subscription?id=invalid", nil)
		w := httptest.NewRecorder()
		h.GetSubscriptionByID(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Not found", func(t *testing.T) {
		mockRepo := &mockSubscriptionRepoInHandlers{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return nil, errors.New("not found")
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("GET", "/subscription?id="+subID.String(), nil)
		w := httptest.NewRecorder()
		h.GetSubscriptionByID(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Success", func(t *testing.T) {
		mockRepo := &mockSubscriptionRepoInHandlers{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{ID: id, Name: "Netflix"}, nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("GET", "/subscription?id="+subID.String(), nil)
		w := httptest.NewRecorder()
		h.GetSubscriptionByID(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var sub core.Subscription
		json.NewDecoder(w.Body).Decode(&sub)
		assert.Equal(t, "Netflix", sub.Name)
	})
}

func TestSubscriptionServiceHandler_UpdateSubscription(t *testing.T) {
	logger := zap.NewNop()
	subID := uuid.New()

	t.Run("Invalid payload", func(t *testing.T) {
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, nil)
		req := httptest.NewRequest("PUT", "/subscription", bytes.NewBufferString(`invalid`))
		w := httptest.NewRecorder()
		h.UpdateSubscription(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Repo error", func(t *testing.T) {
		mockRepo := &mockSubscriptionRepoInHandlers{
			updateFn: func(s *core.Subscription, tx *sql.Tx) error {
				return errors.New("update failed")
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("PUT", "/subscription?id="+subID.String(), bytes.NewBufferString(`{"name":"Netflix"}`))
		w := httptest.NewRecorder()
		h.UpdateSubscription(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Success with existing metadata preservation", func(t *testing.T) {
		now := time.Now()
		txnID := uuid.New()
		mockRepo := &mockSubscriptionRepoInHandlers{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{
					ID:                id,
					UserUUID:          uuid.New(),
					OccurrenceCount:   2,
					ChargeWindowHours: 12,
					LastChargedAt:     &now,
					LastTransactionID: &txnID,
					MerchantPattern:   "NETFLIX.COM",
				}, nil
			},
			updateFn: func(s *core.Subscription, tx *sql.Tx) error {
				assert.Equal(t, 2, s.OccurrenceCount)
				assert.Equal(t, 12, s.ChargeWindowHours)
				assert.Equal(t, "NETFLIX.COM", s.MerchantPattern)
				return nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("PUT", "/subscription?id="+subID.String(), bytes.NewBufferString(`{"name":"Netflix"}`))
		w := httptest.NewRecorder()
		h.UpdateSubscription(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSubscriptionServiceHandler_DeleteSubscription(t *testing.T) {
	logger := zap.NewNop()
	subID := uuid.New()

	t.Run("Missing ID", func(t *testing.T) {
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, nil)
		req := httptest.NewRequest("DELETE", "/subscription", nil)
		w := httptest.NewRecorder()
		h.DeleteSubscription(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, nil)
		req := httptest.NewRequest("DELETE", "/subscription?id=invalid", nil)
		w := httptest.NewRecorder()
		h.DeleteSubscription(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Repo error", func(t *testing.T) {
		mockRepo := &mockSubscriptionRepoInHandlers{
			deleteFn: func(id uuid.UUID) error {
				return errors.New("not found")
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("DELETE", "/subscription?id="+subID.String(), nil)
		w := httptest.NewRecorder()
		h.DeleteSubscription(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Success", func(t *testing.T) {
		mockRepo := &mockSubscriptionRepoInHandlers{
			deleteFn: func(id uuid.UUID) error {
				return nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockRepo}, logger, nil, nil)
		req := httptest.NewRequest("DELETE", "/subscription?id="+subID.String(), nil)
		w := httptest.NewRecorder()
		h.DeleteSubscription(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSubscriptionServiceHandler_RenewSubscription(t *testing.T) {
	logger := zap.NewNop()
	subID := uuid.New()

	t.Run("Missing ID", func(t *testing.T) {
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, nil)
		req := httptest.NewRequest("POST", "/subscription/renew", bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()
		h.RenewSubscription(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid payload without query ID", func(t *testing.T) {
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, nil)
		req := httptest.NewRequest("POST", "/subscription/renew", bytes.NewBufferString(`invalid`))
		w := httptest.NewRecorder()
		h.RenewSubscription(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Success without Cadence", func(t *testing.T) {
		mockSubRepo := &mockSubscriptionRepoInHandlers{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{
					ID:           subID,
					Name:         "Netflix",
					AmountE5:     64900000,
					Status:       core.SubscriptionStatusActive,
					BillingCycle: core.BillingCycleMonthly,
				}, nil
			},
			updateFn: func(s *core.Subscription, tx *sql.Tx) error {
				return nil
			},
		}
		mockTxnRepo := &mockTxnRepoInSubHandlers{
			createTxnFn: func(txn *core.Transaction, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.New(), nil
			},
		}

		h := NewSubscriptionServiceHandler(core.RepoContainer{
			Subscription: mockSubRepo,
			Transaction:  mockTxnRepo,
		}, logger, nil, nil)

		req := httptest.NewRequest("POST", "/subscription/renew?id="+subID.String(), bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()
		h.RenewSubscription(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Activity error without Cadence", func(t *testing.T) {
		mockSubRepo := &mockSubscriptionRepoInHandlers{
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return nil, errors.New("sub not found")
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockSubRepo}, logger, nil, nil)

		req := httptest.NewRequest("POST", "/subscription/renew?id="+subID.String(), bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()
		h.RenewSubscription(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Success with Cadence", func(t *testing.T) {
		txn := &core.Transaction{ID: uuid.New(), Description: "Subscription: Netflix"}
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						ptr := valuePtr.(**core.Transaction)
						*ptr = txn
						return nil
					},
				}, nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, cc)
		req := httptest.NewRequest("POST", "/subscription/renew?id="+subID.String(), bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()
		h.RenewSubscription(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Cadence ExecuteWorkflow error", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return nil, errors.New("cadence error")
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, cc)
		req := httptest.NewRequest("POST", "/subscription/renew?id="+subID.String(), bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()
		h.RenewSubscription(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Cadence Workflow result error", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						return errors.New("renewal activity error")
					},
				}, nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, cc)
		req := httptest.NewRequest("POST", "/subscription/renew?id="+subID.String(), bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()
		h.RenewSubscription(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSubscriptionServiceHandler_ScanDueSubscriptions(t *testing.T) {
	logger := zap.NewNop()

	t.Run("Success without Cadence", func(t *testing.T) {
		sub1ID := uuid.New()
		mockSubRepo := &mockSubscriptionRepoInHandlers{
			getDueFn: func(asOf time.Time, tx *sql.Tx) ([]*core.Subscription, error) {
				return []*core.Subscription{
					{
						ID:           sub1ID,
						Name:         "Netflix",
						Status:       core.SubscriptionStatusActive,
						AmountE5:     64900000,
						BillingCycle: core.BillingCycleMonthly,
					},
				}, nil
			},
			getByIDFn: func(id uuid.UUID) (*core.Subscription, error) {
				return &core.Subscription{
					ID:           sub1ID,
					Name:         "Netflix",
					Status:       core.SubscriptionStatusActive,
					AmountE5:     64900000,
					BillingCycle: core.BillingCycleMonthly,
				}, nil
			},
			updateFn: func(s *core.Subscription, tx *sql.Tx) error {
				return nil
			},
		}
		mockTxnRepo := &mockTxnRepoInSubHandlers{
			createTxnFn: func(txn *core.Transaction, tx *sql.Tx) (uuid.UUID, error) {
				return uuid.New(), nil
			},
		}

		h := NewSubscriptionServiceHandler(core.RepoContainer{
			Subscription: mockSubRepo,
			Transaction:  mockTxnRepo,
		}, logger, nil, nil)

		req := httptest.NewRequest("POST", "/subscriptions/scan-due?as_of=2026-10-02T11:00:00Z", nil)
		w := httptest.NewRecorder()
		h.ScanDueSubscriptions(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Repo error without Cadence", func(t *testing.T) {
		mockSubRepo := &mockSubscriptionRepoInHandlers{
			getDueFn: func(asOf time.Time, tx *sql.Tx) ([]*core.Subscription, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{Subscription: mockSubRepo}, logger, nil, nil)
		req := httptest.NewRequest("POST", "/subscriptions/scan-due", nil)
		w := httptest.NewRecorder()
		h.ScanDueSubscriptions(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Success with Cadence", func(t *testing.T) {
		renewedIDs := []uuid.UUID{uuid.New()}
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						ptr := valuePtr.(*[]uuid.UUID)
						*ptr = renewedIDs
						return nil
					},
				}, nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, cc)
		req := httptest.NewRequest("POST", "/subscriptions/scan-due", nil)
		w := httptest.NewRecorder()
		h.ScanDueSubscriptions(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Cadence ExecuteWorkflow error", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return nil, errors.New("cadence error")
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, cc)
		req := httptest.NewRequest("POST", "/subscriptions/scan-due", nil)
		w := httptest.NewRecorder()
		h.ScanDueSubscriptions(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Cadence Workflow result error", func(t *testing.T) {
		cc := &mockCadenceClient{
			executeWorkflowFn: func(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
				return &mockWorkflowRun{
					getFn: func(ctx context.Context, valuePtr interface{}) error {
						return errors.New("scan workflow failed")
					},
				}, nil
			},
		}
		h := NewSubscriptionServiceHandler(core.RepoContainer{}, logger, nil, cc)
		req := httptest.NewRequest("POST", "/subscriptions/scan-due", nil)
		w := httptest.NewRecorder()
		h.ScanDueSubscriptions(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
