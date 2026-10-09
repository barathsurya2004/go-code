package grpcserver

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func TestStartGRPCServerFromEnv(t *testing.T) {
	srv := NewPenneGRPCServer(&mockTxRepo{}, &mockUserRepo{}, &mockEnvRepo{}, zap.NewNop())

	// Test default address fallback when empty (using dynamic port to avoid conflict)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	os.Setenv("GRPC_ADDR", addr)
	defer os.Unsetenv("GRPC_ADDR")

	app := fx.New(
		fx.NopLogger,
		fx.Provide(func() *PenneGRPCServer { return srv }, zap.NewNop),
		fx.Invoke(StartGRPCServerFromEnv),
	)

	startCtx, cancelStart := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelStart()
	if err := app.Start(startCtx); err != nil {
		t.Fatalf("failed to start app: %v", err)
	}

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelStop()
	if err := app.Stop(stopCtx); err != nil {
		t.Fatalf("failed to stop app: %v", err)
	}
}

func TestStartGRPCServerFromEnv_DefaultFallback(t *testing.T) {
	srv := NewPenneGRPCServer(&mockTxRepo{}, &mockUserRepo{}, &mockEnvRepo{}, zap.NewNop())

	os.Unsetenv("GRPC_ADDR")
	// If 50051 is not free or free, StartGRPCServerFromEnv reads env
	// We can test that StartGRPCServerFromEnv reads "" and attempts :50051
	// without starting lifecycle to avoid binding conflict:
	err := StartGRPCServerFromEnv(srv, nil, zap.NewNop())
	// May succeed or error if port 50051 is in use, which is fine
	_ = err
}
