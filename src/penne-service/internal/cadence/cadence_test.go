package cadence

import (
	"context"
	"testing"

	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func TestCadenceConfig(t *testing.T) {
	cfg := NewCadenceConfig()
	if cfg == nil {
		t.Fatal("expected non-nil CadenceConfig")
	}
	if cfg.Domain != Domain {
		t.Errorf("expected domain %s, got %s", Domain, cfg.Domain)
	}
	if cfg.ServiceName != ClientServiceName {
		t.Errorf("expected service name %s, got %s", ClientServiceName, cfg.ServiceName)
	}
	if cfg.CadenceService != CadenceService {
		t.Errorf("expected cadence service %s, got %s", CadenceService, cfg.CadenceService)
	}
	if cfg.HostPort != CadenceHostPort {
		t.Errorf("expected host port %s, got %s", CadenceHostPort, cfg.HostPort)
	}
}

func TestNewCadenceClient(t *testing.T) {
	cfg := NewCadenceConfig()
	cli := NewCadenceClient(nil, cfg)
	if cli == nil {
		t.Fatal("expected non-nil Cadence client")
	}
}

func TestNewCadenceServiceClient(t *testing.T) {
	logger := zap.NewNop()
	cfg := NewCadenceConfig()
	var serviceClient workflowserviceclient.Interface

	app := fx.New(
		fx.Provide(
			func() *CadenceConfig { return cfg },
			func() *zap.Logger { return logger },
			NewCadenceServiceClient,
		),
		fx.Populate(&serviceClient),
	)

	if err := app.Err(); err != nil {
		t.Fatalf("expected no error initializing Cadence service client, got %v", err)
	}

	if serviceClient == nil {
		t.Fatal("expected non-nil serviceClient")
	}

	ctx := context.Background()
	_ = app.Start(ctx)
	_ = app.Stop(ctx)
}

func TestNewCadenceServiceClient_Error(t *testing.T) {
	logger := zap.NewNop()
	cfg := &CadenceConfig{ServiceName: ""}
	var serviceClient workflowserviceclient.Interface

	app := fx.New(
		fx.Provide(
			func() *CadenceConfig { return cfg },
			func() *zap.Logger { return logger },
			NewCadenceServiceClient,
		),
		fx.Populate(&serviceClient),
	)

	if err := app.Err(); err == nil {
		t.Error("expected error with empty service name, got nil")
	}
}
