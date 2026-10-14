package cadence

import (
	"context"
	"errors"
	"testing"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/cadence/activities"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/db"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
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

	t.Setenv("CADENCE_ADDR", "custom:7933")
	cfgAddr := NewCadenceConfig()
	if cfgAddr.HostPort != "custom:7933" {
		t.Errorf("expected custom:7933, got %s", cfgAddr.HostPort)
	}

	t.Setenv("CADENCE_HOST_PORT", "override:7933")
	cfgHostPort := NewCadenceConfig()
	if cfgHostPort.HostPort != "override:7933" {
		t.Errorf("expected override:7933, got %s", cfgHostPort.HostPort)
	}

	t.Setenv("CADENCE_DOMAIN", "custom-domain")
	cfgDomain := NewCadenceConfig()
	if cfgDomain.Domain != "custom-domain" {
		t.Errorf("expected custom-domain, got %s", cfgDomain.Domain)
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

	app := fxtest.New(
		t,
		fx.Provide(
			func() *CadenceConfig { return cfg },
			func() *zap.Logger { return logger },
			NewCadenceServiceClient,
		),
		fx.Populate(&serviceClient),
	)
	defer app.RequireStop()

	app.RequireStart()

	if serviceClient == nil {
		t.Fatal("expected non-nil serviceClient")
	}
}

func TestNewCadenceServiceClient_NilLifecycle(t *testing.T) {
	logger := zap.NewNop()
	cfg := NewCadenceConfig()
	cli, err := NewCadenceServiceClient(nil, cfg, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cli == nil {
		t.Fatal("expected non-nil cli")
	}
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

type mockWorker struct {
	startErr error
	started  bool
	stopped  bool
}

func (m *mockWorker) Start() error {
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *mockWorker) Run() error   { return m.startErr }
func (m *mockWorker) Stop()        { m.stopped = true }

func (m *mockWorker) RegisterWorkflow(_ interface{})                                        {}
func (m *mockWorker) RegisterWorkflowWithOptions(_ interface{}, _ workflow.RegisterOptions) {}
func (m *mockWorker) GetRegisteredWorkflows() []workflow.RegistryInfo                       { return nil }

func (m *mockWorker) RegisterActivity(_ interface{})                                         {}
func (m *mockWorker) RegisterActivityWithOptions(_ interface{}, _ activity.RegisterOptions)  {}
func (m *mockWorker) GetRegisteredActivities() []activity.RegistryInfo                       { return nil }

func TestStartCadenceWorker_ValidationErrors(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	acts := activities.NewNutritionActivities(repo, nil, logger)

	// Nil service client
	cfg := NewCadenceConfig()
	_, err := StartCadenceWorker(nil, cfg, acts, logger, nil)
	if err == nil {
		t.Error("expected error when serviceClient is nil")
	}

	// Nil config
	mockSvc := &mockServiceClient{}
	_, err = StartCadenceWorker(mockSvc, nil, acts, logger, nil)
	if err == nil {
		t.Error("expected error when cfg is nil")
	}

	// Empty domain
	cfgEmpty := &CadenceConfig{Domain: ""}
	_, err = StartCadenceWorker(mockSvc, cfgEmpty, acts, logger, nil)
	if err == nil {
		t.Error("expected error when domain is empty")
	}
}

type mockServiceClient struct {
	workflowserviceclient.Interface
}

func TestStartCadenceWorker_Lifecycle(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	acts := activities.NewNutritionActivities(repo, nil, logger)
	cfg := NewCadenceConfig()
	mockSvc := &mockServiceClient{}

	mockW := &mockWorker{}
	origFactory := workerNewFn
	defer func() { workerNewFn = origFactory }()

	workerNewFn = func(svc workflowserviceclient.Interface, domain, taskList string, opts worker.Options) (worker.Worker, error) {
		return mockW, nil
	}

	var w worker.Worker
	app := fxtest.New(
		t,
		fx.Provide(
			func() workflowserviceclient.Interface { return mockSvc },
			func() *CadenceConfig { return cfg },
			func() *activities.NutritionActivities { return acts },
			func() *zap.Logger { return logger },
			StartCadenceWorker,
		),
		fx.Populate(&w),
	)

	app.RequireStart()
	if !mockW.started {
		t.Error("expected worker to be started")
	}

	app.RequireStop()
	if !mockW.stopped {
		t.Error("expected worker to be stopped")
	}
}

func TestStartCadenceWorker_FactoryError(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	acts := activities.NewNutritionActivities(repo, nil, logger)
	cfg := NewCadenceConfig()
	mockSvc := &mockServiceClient{}

	origFactory := workerNewFn
	defer func() { workerNewFn = origFactory }()

	workerNewFn = func(svc workflowserviceclient.Interface, domain, taskList string, opts worker.Options) (worker.Worker, error) {
		return nil, errors.New("factory failed")
	}

	_, err := StartCadenceWorker(mockSvc, cfg, acts, logger, nil)
	if err == nil {
		t.Error("expected error from worker factory")
	}
}

func TestStartCadenceWorker_StartError(t *testing.T) {
	logger := zap.NewNop()
	repo := db.NewMemoryNutritionRepository(logger)
	acts := activities.NewNutritionActivities(repo, nil, logger)
	cfg := NewCadenceConfig()
	mockSvc := &mockServiceClient{}

	mockW := &mockWorker{startErr: errors.New("start failed")}
	origFactory := workerNewFn
	defer func() { workerNewFn = origFactory }()

	workerNewFn = func(svc workflowserviceclient.Interface, domain, taskList string, opts worker.Options) (worker.Worker, error) {
		return mockW, nil
	}

	var w worker.Worker
	app := fx.New(
		fx.Provide(
			func() workflowserviceclient.Interface { return mockSvc },
			func() *CadenceConfig { return cfg },
			func() *activities.NutritionActivities { return acts },
			func() *zap.Logger { return logger },
			StartCadenceWorker,
		),
		fx.Populate(&w),
	)

	ctx := context.Background()
	err := app.Start(ctx)
	if err == nil {
		t.Error("expected error starting app when worker start fails")
	}
}
