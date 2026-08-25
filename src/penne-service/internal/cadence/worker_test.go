package cadence

import (
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestRegisterWorkflowsAndActivities(t *testing.T) {
	RegisterWorkflows()
	RegisterActivities(core.RepoContainer{})
}

func TestStartWorker_NilClientError(t *testing.T) {
	logger := zap.NewNop()
	cfg := NewCadenceConfig()
	lc := fxtest.NewLifecycle(t)
	_, err := StartWorker(nil, cfg, logger, core.RepoContainer{}, lc)
	if err == nil {
		t.Error("expected error when serviceClient is nil, got nil")
	}
}

func TestStartWorker_WithServiceClient(t *testing.T) {
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
		t.Fatalf("failed to create serviceClient: %v", err)
	}

	lc := fxtest.NewLifecycle(t)
	w, err := StartWorker(serviceClient, cfg, logger, core.RepoContainer{}, lc)
	if err != nil {
		t.Fatalf("expected no error creating worker, got %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil worker")
	}
}
