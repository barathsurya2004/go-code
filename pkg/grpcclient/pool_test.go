package grpcclient

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestNewClientPool_Validation(t *testing.T) {
	_, err := NewClientPool(PoolConfig{})
	if err == nil {
		t.Fatal("expected error for empty target, got nil")
	}
}

func TestNewClientPool_TCP(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	srv := grpc.NewServer()
	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	cfg := PoolConfig{
		Target:                lis.Addr().String(),
		Size:                  3,
		InitialWindowSize:     512 * 1024,
		InitialConnWindowSize: 1024 * 1024,
		KeepAliveTime:         10 * time.Second,
		KeepAliveTimeout:      2 * time.Second,
	}

	pool, err := NewClientPool(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating pool: %v", err)
	}
	defer pool.Close()

	if pool.Size() != 3 {
		t.Errorf("expected pool size 3, got %d", pool.Size())
	}

	c1 := pool.Get()
	c2 := pool.Get()
	c3 := pool.Get()
	c4 := pool.Get()

	if c1 == nil || c2 == nil || c3 == nil {
		t.Fatal("expected non-nil connections from pool")
	}
	if c1 != c4 {
		t.Errorf("expected round-robin wrap around to first connection")
	}
}

func TestNewClientPool_UnixDomainSocket(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uds-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sockPath := filepath.Join(tmpDir, "test.sock")
	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to listen on UDS: %v", err)
	}
	defer lis.Close()

	srv := grpc.NewServer()
	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	cfg := PoolConfig{
		Target: "unix://" + sockPath,
		Size:   2,
	}

	pool, err := NewClientPool(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating UDS pool: %v", err)
	}
	defer pool.Close()

	if pool.Size() != 2 {
		t.Errorf("expected pool size 2, got %d", pool.Size())
	}

	conn := pool.Get()
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
}

func TestNewClientPool_DefaultsAndUnixColon(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "uds-colon-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sockPath := filepath.Join(tmpDir, "test.sock")
	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to listen on UDS: %v", err)
	}
	defer lis.Close()

	srv := grpc.NewServer()
	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	// Use target "unix:/..." and defaults (size 0, winSize 0, etc.)
	cfg := PoolConfig{
		Target:      "unix:" + sockPath,
		DialOptions: []grpc.DialOption{grpc.WithNoProxy()},
	}

	pool, err := NewClientPool(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating pool with defaults: %v", err)
	}
	defer pool.Close()

	if pool.Size() != DefaultPoolSize {
		t.Errorf("expected default pool size %d, got %d", DefaultPoolSize, pool.Size())
	}
}

func TestClientPool_EmptyGet(t *testing.T) {
	pool := &ClientPool{conns: nil}
	if conn := pool.Get(); conn != nil {
		t.Errorf("expected nil for empty pool, got %v", conn)
	}
}

func TestWithUserMetadata(t *testing.T) {
	ctx := context.Background()
	ctx = WithUserMetadata(ctx, "user-123", "req-456")

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata to be set")
	}

	u := md.Get("x-user-uuid")
	if len(u) == 0 || u[0] != "user-123" {
		t.Errorf("expected user-123, got %v", u)
	}

	r := md.Get("x-request-id")
	if len(r) == 0 || r[0] != "req-456" {
		t.Errorf("expected req-456, got %v", r)
	}

	// Empty fields branch
	ctx2 := WithUserMetadata(context.Background(), "", "")
	md2, _ := metadata.FromOutgoingContext(ctx2)
	if len(md2.Get("x-user-uuid")) != 0 {
		t.Errorf("expected empty user-uuid")
	}
}

func TestNewClientPool_ClientCreationError(t *testing.T) {
	oldFn := grpcNewClient
	defer func() { grpcNewClient = oldFn }()

	callCount := 0
	grpcNewClient = func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		callCount++
		if callCount > 1 {
			return nil, fmt.Errorf("injected connection failure")
		}
		return oldFn(target, opts...)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer lis.Close()

	_, err = NewClientPool(PoolConfig{
		Target: lis.Addr().String(),
		Size:   3,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClientPool_CloseError(t *testing.T) {
	oldClose := connClose
	defer func() { connClose = oldClose }()

	connClose = func(conn *grpc.ClientConn) error {
		return fmt.Errorf("injected close error")
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer lis.Close()

	pool, err := NewClientPool(PoolConfig{
		Target: lis.Addr().String(),
		Size:   2,
	})
	if err != nil {
		t.Fatalf("unexpected pool error: %v", err)
	}

	err = pool.Close()
	if err == nil {
		t.Fatal("expected close error, got nil")
	}
}

