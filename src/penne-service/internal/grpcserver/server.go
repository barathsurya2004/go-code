package grpcserver

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	pennev1 "github.com/barathsurya2004/go-code/pkg/proto/pennev1"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type PenneGRPCServer struct {
	pennev1.UnimplementedPenneServiceServer
	txRepo   core.TransactionRepository
	userRepo core.UserRepository
	envRepo  core.EnvelopeRepository
	log      *zap.Logger
}

func NewPenneGRPCServer(
	txRepo core.TransactionRepository,
	userRepo core.UserRepository,
	envRepo core.EnvelopeRepository,
	log *zap.Logger,
) *PenneGRPCServer {
	return &PenneGRPCServer{
		txRepo:   txRepo,
		userRepo: userRepo,
		envRepo:  envRepo,
		log:      log,
	}
}

func extractUserUUID(ctx context.Context, fallback string) (uuid.UUID, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		vals := md.Get("x-user-uuid")
		if len(vals) > 0 && vals[0] != "" {
			return uuid.Parse(vals[0])
		}
	}
	if fallback != "" {
		return uuid.Parse(fallback)
	}
	return uuid.Nil, fmt.Errorf("user uuid missing from metadata and request")
}

func (s *PenneGRPCServer) CheckHealth(ctx context.Context, req *pennev1.HealthCheckRequest) (*pennev1.HealthCheckResponse, error) {
	return &pennev1.HealthCheckResponse{
		Status:      "SERVING",
		ServiceName: "penne-service",
		Timestamp:   time.Now().Unix(),
	}, nil
}

func (s *PenneGRPCServer) GetUser(ctx context.Context, req *pennev1.GetUserRequest) (*pennev1.UserResponse, error) {
	userUUID, err := extractUserUUID(ctx, req.GetUserUuid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user UUID: %v", err)
	}

	user, err := s.userRepo.GetUserByUUID(userUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	return &pennev1.UserResponse{
		Uuid:          user.UUID.String(),
		Name:          user.Name,
		Email:         user.Email,
		MonthlyIncome: float64(user.MonthlyBudgetE5) / 100000.0,
	}, nil
}

func (s *PenneGRPCServer) CreateTransaction(ctx context.Context, req *pennev1.CreateTransactionRequest) (*pennev1.TransactionItem, error) {
	userUUID, err := extractUserUUID(ctx, req.GetUserUuid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user UUID: %v", err)
	}

	var envID *uuid.UUID
	if req.GetEnvelopeId() != "" {
		parsedEnv, err := uuid.Parse(req.GetEnvelopeId())
		if err == nil {
			envID = &parsedEnv
		}
	}

	amountE5 := int64(req.GetAmount() * 100000.0)
	txnType := req.GetCategory()
	if txnType == "" {
		txnType = core.TxnTypeDebit
	}

	txn := &core.Transaction{
		ID:          uuid.New(),
		UserID:      userUUID,
		EnvelopeID:  envID,
		AmountE5:    amountE5,
		Type:        txnType,
		CountryISO:  "IN",
		CreatedAt:   time.Now(),
		Description: req.GetRemarks(),
	}

	txID, err := s.txRepo.CreateTransaction(txn, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create transaction: %v", err)
	}
	txn.ID = txID

	return toProtoTransaction(txn), nil
}

func (s *PenneGRPCServer) GetTransaction(ctx context.Context, req *pennev1.GetTransactionRequest) (*pennev1.TransactionItem, error) {
	txUUID, err := uuid.Parse(req.GetTransactionUuid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction UUID: %v", err)
	}

	txn, err := s.txRepo.GetTransactionByUUID(txUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "transaction not found: %v", err)
	}

	return toProtoTransaction(txn), nil
}

func (s *PenneGRPCServer) GetTransactionsByUser(ctx context.Context, req *pennev1.GetTransactionsByUserRequest) (*pennev1.TransactionListResponse, error) {
	userUUID, err := extractUserUUID(ctx, req.GetUserUuid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user UUID: %v", err)
	}

	txns, err := s.txRepo.GetTransactionsByUserUUID(userUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch transactions: %v", err)
	}

	items := make([]*pennev1.TransactionItem, 0, len(txns))
	for _, t := range txns {
		items = append(items, toProtoTransaction(t))
	}

	return &pennev1.TransactionListResponse{
		Transactions: items,
		TotalCount:   int32(len(items)),
	}, nil
}

func (s *PenneGRPCServer) UpdateTransaction(ctx context.Context, req *pennev1.UpdateTransactionRequest) (*pennev1.TransactionItem, error) {
	txUUID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction ID: %v", err)
	}

	userUUID, err := extractUserUUID(ctx, req.GetUserUuid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user UUID: %v", err)
	}

	var envID *uuid.UUID
	if req.GetEnvelopeId() != "" {
		parsedEnv, err := uuid.Parse(req.GetEnvelopeId())
		if err == nil {
			envID = &parsedEnv
		}
	}

	amountE5 := int64(req.GetAmount() * 100000.0)
	txnType := req.GetCategory()
	if txnType == "" {
		txnType = core.TxnTypeDebit
	}

	txn := &core.Transaction{
		ID:          txUUID,
		UserID:      userUUID,
		EnvelopeID:  envID,
		AmountE5:    amountE5,
		Type:        txnType,
		CountryISO:  "IN",
		Description: req.GetRemarks(),
	}

	if err := s.txRepo.UpdateTransaction(txn, nil); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update transaction: %v", err)
	}

	return toProtoTransaction(txn), nil
}

func (s *PenneGRPCServer) DeleteTransaction(ctx context.Context, req *pennev1.DeleteTransactionRequest) (*pennev1.DeleteResponse, error) {
	txUUID, err := uuid.Parse(req.GetTransactionUuid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction ID: %v", err)
	}

	if err := s.txRepo.DeleteTransaction(txUUID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete transaction: %v", err)
	}

	return &pennev1.DeleteResponse{
		Success: true,
		Message: "Transaction deleted successfully",
	}, nil
}

func (s *PenneGRPCServer) GetEnvelope(ctx context.Context, req *pennev1.GetEnvelopeRequest) (*pennev1.EnvelopeItem, error) {
	envUUID, err := uuid.Parse(req.GetEnvelopeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid envelope ID: %v", err)
	}

	env, err := s.envRepo.GetEnvelopeByID(envUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "envelope not found: %v", err)
	}

	return &pennev1.EnvelopeItem{
		Id:          env.ID.String(),
		Name:        env.Name,
		Balance:     env.TargetAmountE5 / 100000.0,
		BudgetLimit: env.TargetAmountE5 / 100000.0,
		UserUuid:    env.UserUUID.String(),
	}, nil
}

func (s *PenneGRPCServer) GetEnvelopesByUser(ctx context.Context, req *pennev1.GetEnvelopesByUserRequest) (*pennev1.EnvelopeListResponse, error) {
	userUUID, err := extractUserUUID(ctx, req.GetUserUuid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user UUID: %v", err)
	}

	envs, err := s.envRepo.GetEnvelopesByUserUUID(userUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch envelopes: %v", err)
	}

	items := make([]*pennev1.EnvelopeItem, 0, len(envs))
	for _, e := range envs {
		items = append(items, &pennev1.EnvelopeItem{
			Id:          e.ID.String(),
			Name:        e.Name,
			Balance:     e.TargetAmountE5 / 100000.0,
			BudgetLimit: e.TargetAmountE5 / 100000.0,
			UserUuid:    e.UserUUID.String(),
		})
	}

	return &pennev1.EnvelopeListResponse{
		Envelopes: items,
	}, nil
}

func toProtoTransaction(txn *core.Transaction) *pennev1.TransactionItem {
	var envIDStr string
	if txn.EnvelopeID != nil {
		envIDStr = txn.EnvelopeID.String()
	}
	return &pennev1.TransactionItem{
		Id:         txn.ID.String(),
		UserUuid:   txn.UserID.String(),
		Amount:     float64(txn.AmountE5) / 100000.0,
		Category:   txn.Type,
		Date:       txn.CreatedAt.Format(time.RFC3339),
		Remarks:    txn.Description,
		EnvelopeId: envIDStr,
	}
}

// StartGRPCServer starts a gRPC listener on either TCP or Unix Domain Socket.
func StartGRPCServer(server *PenneGRPCServer, targetAddr string, lc fx.Lifecycle) (*grpc.Server, error) {
	var lis net.Listener
	var err error

	if strings.HasPrefix(targetAddr, "unix://") || strings.HasPrefix(targetAddr, "unix:") {
		sockPath := strings.TrimPrefix(targetAddr, "unix://")
		sockPath = strings.TrimPrefix(sockPath, "unix:")
		_ = os.Remove(sockPath)
		lis, err = net.Listen("unix", sockPath)
	} else {
		lis, err = net.Listen("tcp", targetAddr)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", targetAddr, err)
	}

	grpcSrv := grpc.NewServer(
		grpc.InitialWindowSize(1024*1024),
		grpc.InitialConnWindowSize(4*1024*1024),
	)
	pennev1.RegisterPenneServiceServer(grpcSrv, server)

	if lc != nil {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					if err := grpcSrv.Serve(lis); err != nil && err != grpc.ErrServerStopped {
						server.log.Error("gRPC server serve failure", zap.Error(err))
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				grpcSrv.GracefulStop()
				return nil
			},
		})
	}

	return grpcSrv, nil
}
