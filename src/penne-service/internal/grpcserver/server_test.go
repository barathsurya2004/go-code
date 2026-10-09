package grpcserver

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	pennev1 "github.com/barathsurya2004/go-code/pkg/proto/pennev1"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type mockTxRepo struct {
	createFn    func(txn *core.Transaction, tx *sql.Tx) (uuid.UUID, error)
	getByUUIDFn func(id uuid.UUID) (*core.Transaction, error)
	getByUserFn func(userUUID uuid.UUID) ([]*core.Transaction, error)
	updateFn    func(txn *core.Transaction, tx *sql.Tx) error
	deleteFn    func(id uuid.UUID) error
}

func (m *mockTxRepo) CreateTransaction(txn *core.Transaction, tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(txn, tx)
	}
	return uuid.New(), nil
}
func (m *mockTxRepo) GetTransactionByUUID(id uuid.UUID) (*core.Transaction, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id)
	}
	return &core.Transaction{ID: id, UserID: uuid.New(), AmountE5: 100000}, nil
}
func (m *mockTxRepo) GetTransactionsByUserUUID(userUUID uuid.UUID) ([]*core.Transaction, error) {
	if m.getByUserFn != nil {
		return m.getByUserFn(userUUID)
	}
	return []*core.Transaction{{ID: uuid.New(), UserID: userUUID}}, nil
}
func (m *mockTxRepo) GetTransactionByTime(t1, t2 time.Time, tx *sql.Tx) (*core.Transaction, error) {
	return nil, nil
}
func (m *mockTxRepo) GetTransactionByAmountAndTime(u uuid.UUID, a int64, t1, t2 time.Time, tx *sql.Tx) (*core.Transaction, error) {
	return nil, nil
}
func (m *mockTxRepo) UpdateTransaction(txn *core.Transaction, tx *sql.Tx) error {
	if m.updateFn != nil {
		return m.updateFn(txn, tx)
	}
	return nil
}
func (m *mockTxRepo) DeleteTransaction(id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}
func (m *mockTxRepo) GetDashboardSummary(id uuid.UUID) (*core.DashboardSummary, error) {
	return nil, nil
}
func (m *mockTxRepo) GetTransactionByUserUUIDPaginated(u uuid.UUID, t time.Time, id uuid.UUID, l int) ([]*core.Transaction, error) {
	return nil, nil
}
func (m *mockTxRepo) GetMonthlyInsights(u uuid.UUID, y, mth int) (*core.MonthlyInsightsReport, error) {
	return nil, nil
}

type mockUserRepo struct {
	createFn func(u *core.User, tx *sql.Tx) (uuid.UUID, error)
	getByUID func(u uuid.UUID) (*core.User, error)
}

func (m *mockUserRepo) CreateUser(u *core.User, tx *sql.Tx) (uuid.UUID, error) {
	if m.createFn != nil {
		return m.createFn(u, tx)
	}
	return uuid.New(), nil
}
func (m *mockUserRepo) GetUserByUUID(id uuid.UUID) (*core.User, error) {
	if m.getByUID != nil {
		return m.getByUID(id)
	}
	return &core.User{UUID: id, Name: "Test", Email: "test@example.com", MonthlyBudgetE5: 500000}, nil
}
func (m *mockUserRepo) GetUserByEmail(e string) (*core.User, error) { return nil, nil }
func (m *mockUserRepo) UpdateBudgetSettings(u uuid.UUID, b int64, s int, tx *sql.Tx) error {
	return nil
}

type mockEnvRepo struct {
	getByIDFn   func(id uuid.UUID) (*core.Envelope, error)
	getByUserFn func(userUUID uuid.UUID) ([]*core.Envelope, error)
}

func (m *mockEnvRepo) CreateEnvelope(e *core.Envelope, tx *sql.Tx) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockEnvRepo) GetEnvelopeByID(id uuid.UUID) (*core.Envelope, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return &core.Envelope{ID: id, Name: "Groceries", TargetAmountE5: 200000}, nil
}
func (m *mockEnvRepo) GetEnvelopesByUserUUID(id uuid.UUID) ([]*core.Envelope, error) {
	if m.getByUserFn != nil {
		return m.getByUserFn(id)
	}
	return []*core.Envelope{{ID: uuid.New(), Name: "Groceries", TargetAmountE5: 200000}}, nil
}
func (m *mockEnvRepo) UpdateEnvelope(e *core.Envelope) error                                 { return nil }
func (m *mockEnvRepo) DeleteEnvelope(id uuid.UUID) error                                      { return nil }
func (m *mockEnvRepo) GetEnvelopeIdByName(n string, u uuid.UUID, tx *sql.Tx) (uuid.UUID, error) {
	return uuid.New(), nil
}

func TestHealthCheck(t *testing.T) {
	srv := NewPenneGRPCServer(&mockTxRepo{}, &mockUserRepo{}, &mockEnvRepo{}, zap.NewNop())
	res, err := srv.CheckHealth(context.Background(), &pennev1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "SERVING" {
		t.Errorf("expected SERVING, got %s", res.Status)
	}
}

func TestGetUser(t *testing.T) {
	userUUID := uuid.New()
	userRepo := &mockUserRepo{
		getByUID: func(u uuid.UUID) (*core.User, error) {
			if u == userUUID {
				return &core.User{UUID: u, Name: "Alice", Email: "alice@test.com", MonthlyBudgetE5: 500000}, nil
			}
			return nil, fmt.Errorf("user not found")
		},
	}
	srv := NewPenneGRPCServer(&mockTxRepo{}, userRepo, &mockEnvRepo{}, zap.NewNop())

	// Via metadata
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-uuid", userUUID.String()))
	res, err := srv.GetUser(ctx, &pennev1.GetUserRequest{})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if res.Name != "Alice" {
		t.Errorf("expected Alice, got %s", res.Name)
	}

	// Via request fallback
	res2, err := srv.GetUser(context.Background(), &pennev1.GetUserRequest{UserUuid: userUUID.String()})
	if err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}
	if res2.Email != "alice@test.com" {
		t.Errorf("expected alice@test.com, got %s", res2.Email)
	}

	// Invalid UUID error
	_, err = srv.GetUser(context.Background(), &pennev1.GetUserRequest{UserUuid: "invalid"})
	if err == nil {
		t.Fatal("expected invalid uuid error")
	}

	// Not found error
	_, err = srv.GetUser(context.Background(), &pennev1.GetUserRequest{UserUuid: uuid.New().String()})
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestTransactionOps(t *testing.T) {
	userUUID := uuid.New()
	txUUID := uuid.New()
	envUUID := uuid.New()

	txRepo := &mockTxRepo{
		createFn: func(txn *core.Transaction, tx *sql.Tx) (uuid.UUID, error) {
			if txn.AmountE5 < 0 {
				return uuid.Nil, fmt.Errorf("negative amount")
			}
			return txUUID, nil
		},
		getByUUIDFn: func(id uuid.UUID) (*core.Transaction, error) {
			if id == txUUID {
				return &core.Transaction{ID: id, UserID: userUUID, AmountE5: 250000, Type: core.TxnTypeDebit, EnvelopeID: &envUUID}, nil
			}
			return nil, fmt.Errorf("not found")
		},
		getByUserFn: func(u uuid.UUID) ([]*core.Transaction, error) {
			if u == userUUID {
				return []*core.Transaction{{ID: txUUID, UserID: u, AmountE5: 250000}}, nil
			}
			return nil, fmt.Errorf("db error")
		},
		updateFn: func(txn *core.Transaction, tx *sql.Tx) error {
			if txn.AmountE5 < 0 {
				return fmt.Errorf("invalid update")
			}
			return nil
		},
		deleteFn: func(id uuid.UUID) error {
			if id == txUUID {
				return nil
			}
			return fmt.Errorf("delete failed")
		},
	}

	srv := NewPenneGRPCServer(txRepo, &mockUserRepo{}, &mockEnvRepo{}, zap.NewNop())

	// Create transaction success
	createRes, err := srv.CreateTransaction(context.Background(), &pennev1.CreateTransactionRequest{
		UserUuid:   userUUID.String(),
		Amount:     2.5,
		EnvelopeId: envUUID.String(),
		Remarks:    "lunch",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if createRes.Remarks != "lunch" {
		t.Errorf("expected lunch, got %s", createRes.Remarks)
	}

	// Create transaction user error
	_, err = srv.CreateTransaction(context.Background(), &pennev1.CreateTransactionRequest{UserUuid: ""})
	if err == nil {
		t.Fatal("expected user error")
	}

	// Create transaction repo error
	_, err = srv.CreateTransaction(context.Background(), &pennev1.CreateTransactionRequest{
		UserUuid: userUUID.String(),
		Amount:   -1,
	})
	if err == nil {
		t.Fatal("expected repo error")
	}

	// Get transaction success
	getRes, err := srv.GetTransaction(context.Background(), &pennev1.GetTransactionRequest{TransactionUuid: txUUID.String()})
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if getRes.Amount != 2.5 {
		t.Errorf("expected 2.5, got %f", getRes.Amount)
	}

	// Get transaction invalid UUID
	_, err = srv.GetTransaction(context.Background(), &pennev1.GetTransactionRequest{TransactionUuid: "bad"})
	if err == nil {
		t.Fatal("expected invalid uuid")
	}

	// Get transaction not found
	_, err = srv.GetTransaction(context.Background(), &pennev1.GetTransactionRequest{TransactionUuid: uuid.New().String()})
	if err == nil {
		t.Fatal("expected not found")
	}

	// Get transactions by user success
	listRes, err := srv.GetTransactionsByUser(context.Background(), &pennev1.GetTransactionsByUserRequest{UserUuid: userUUID.String()})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if listRes.TotalCount != 1 {
		t.Errorf("expected count 1, got %d", listRes.TotalCount)
	}

	// Get transactions by user invalid uuid
	_, err = srv.GetTransactionsByUser(context.Background(), &pennev1.GetTransactionsByUserRequest{UserUuid: "bad"})
	if err == nil {
		t.Fatal("expected bad uuid error")
	}

	// Get transactions by user error
	_, err = srv.GetTransactionsByUser(context.Background(), &pennev1.GetTransactionsByUserRequest{UserUuid: uuid.New().String()})
	if err == nil {
		t.Fatal("expected error")
	}

	// Update transaction success
	upRes, err := srv.UpdateTransaction(context.Background(), &pennev1.UpdateTransactionRequest{
		Id:         txUUID.String(),
		UserUuid:   userUUID.String(),
		Amount:     3.0,
		EnvelopeId: envUUID.String(),
		Remarks:    "dinner",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if upRes.Remarks != "dinner" {
		t.Errorf("expected dinner, got %s", upRes.Remarks)
	}

	// Update transaction bad id
	_, err = srv.UpdateTransaction(context.Background(), &pennev1.UpdateTransactionRequest{Id: "bad"})
	if err == nil {
		t.Fatal("expected bad id error")
	}

	// Update transaction bad user
	_, err = srv.UpdateTransaction(context.Background(), &pennev1.UpdateTransactionRequest{Id: txUUID.String(), UserUuid: "bad"})
	if err == nil {
		t.Fatal("expected bad user error")
	}

	// Update transaction repo error
	_, err = srv.UpdateTransaction(context.Background(), &pennev1.UpdateTransactionRequest{
		Id:       txUUID.String(),
		UserUuid: userUUID.String(),
		Amount:   -10,
	})
	if err == nil {
		t.Fatal("expected repo error")
	}

	// Delete transaction success
	delRes, err := srv.DeleteTransaction(context.Background(), &pennev1.DeleteTransactionRequest{TransactionUuid: txUUID.String()})
	if err != nil || !delRes.Success {
		t.Fatalf("delete failed: %v", err)
	}

	// Delete transaction bad id
	_, err = srv.DeleteTransaction(context.Background(), &pennev1.DeleteTransactionRequest{TransactionUuid: "bad"})
	if err == nil {
		t.Fatal("expected bad id error")
	}

	// Delete transaction repo error
	_, err = srv.DeleteTransaction(context.Background(), &pennev1.DeleteTransactionRequest{TransactionUuid: uuid.New().String()})
	if err == nil {
		t.Fatal("expected delete error")
	}
}

func TestEnvelopeOps(t *testing.T) {
	userUUID := uuid.New()
	envUUID := uuid.New()

	envRepo := &mockEnvRepo{
		getByIDFn: func(id uuid.UUID) (*core.Envelope, error) {
			if id == envUUID {
				return &core.Envelope{ID: id, Name: "Rent", TargetAmountE5: 1000000, UserUUID: userUUID}, nil
			}
			return nil, fmt.Errorf("not found")
		},
		getByUserFn: func(u uuid.UUID) ([]*core.Envelope, error) {
			if u == userUUID {
				return []*core.Envelope{{ID: envUUID, Name: "Rent", TargetAmountE5: 1000000, UserUUID: u}}, nil
			}
			return nil, fmt.Errorf("db failure")
		},
	}

	srv := NewPenneGRPCServer(&mockTxRepo{}, &mockUserRepo{}, envRepo, zap.NewNop())

	// Get envelope success
	eRes, err := srv.GetEnvelope(context.Background(), &pennev1.GetEnvelopeRequest{EnvelopeId: envUUID.String()})
	if err != nil {
		t.Fatalf("get env failed: %v", err)
	}
	if eRes.Name != "Rent" {
		t.Errorf("expected Rent, got %s", eRes.Name)
	}

	// Bad id
	_, err = srv.GetEnvelope(context.Background(), &pennev1.GetEnvelopeRequest{EnvelopeId: "bad"})
	if err == nil {
		t.Fatal("expected bad id")
	}

	// Not found
	_, err = srv.GetEnvelope(context.Background(), &pennev1.GetEnvelopeRequest{EnvelopeId: uuid.New().String()})
	if err == nil {
		t.Fatal("expected not found")
	}

	// Get envelopes by user
	elRes, err := srv.GetEnvelopesByUser(context.Background(), &pennev1.GetEnvelopesByUserRequest{UserUuid: userUUID.String()})
	if err != nil {
		t.Fatalf("get user envs failed: %v", err)
	}
	if len(elRes.Envelopes) != 1 {
		t.Errorf("expected 1 envelope, got %d", len(elRes.Envelopes))
	}

	// Bad user
	_, err = srv.GetEnvelopesByUser(context.Background(), &pennev1.GetEnvelopesByUserRequest{UserUuid: "bad"})
	if err == nil {
		t.Fatal("expected bad user")
	}

	// Repo error
	_, err = srv.GetEnvelopesByUser(context.Background(), &pennev1.GetEnvelopesByUserRequest{UserUuid: uuid.New().String()})
	if err == nil {
		t.Fatal("expected repo error")
	}
}

func TestStartGRPCServer(t *testing.T) {
	srv := NewPenneGRPCServer(&mockTxRepo{}, &mockUserRepo{}, &mockEnvRepo{}, zap.NewNop())

	// Test TCP
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	grpcSrv, err := StartGRPCServer(srv, addr, nil)
	if err != nil {
		t.Fatalf("failed to start on tcp: %v", err)
	}
	grpcSrv.Stop()

	// Test Unix Domain Socket with lifecycle
	tmpDir, err := os.MkdirTemp("", "grpc-uds-*")
	if err != nil {
		t.Fatalf("temp dir error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sockPath := filepath.Join(tmpDir, "test.sock")
	app := fx.New(
		fx.NopLogger,
		fx.Provide(func() *PenneGRPCServer { return srv }),
		fx.Invoke(func(s *PenneGRPCServer, lc fx.Lifecycle) {
			_, err := StartGRPCServer(s, "unix://"+sockPath, lc)
			if err != nil {
				t.Fatalf("start error: %v", err)
			}
		}),
	)

	startCtx, cancelStart := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelStart()
	if err := app.Start(startCtx); err != nil {
		t.Fatalf("app start error: %v", err)
	}

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelStop()
	if err := app.Stop(stopCtx); err != nil {
		t.Fatalf("app stop error: %v", err)
	}

	// Invalid listen target
	_, err = StartGRPCServer(srv, "invalid:port:format:999999", nil)
	if err == nil {
		t.Fatal("expected listen error for invalid address")
	}
}
