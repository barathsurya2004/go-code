package cadence

import (
	"context"
	"errors"
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

// ─── mock worker ─────────────────────────────────────────────────────────────

type mockWorker struct {
	startErr error
	stopped  bool
}

func (m *mockWorker) Start() error { return m.startErr }
func (m *mockWorker) Run() error   { return m.startErr }
func (m *mockWorker) Stop()        { m.stopped = true }

// worker.WorkflowRegistry stub methods
func (m *mockWorker) RegisterWorkflow(_ interface{})                                        {}
func (m *mockWorker) RegisterWorkflowWithOptions(_ interface{}, _ workflow.RegisterOptions) {}
func (m *mockWorker) GetRegisteredWorkflows() []workflow.RegistryInfo                       { return nil }

// worker.ActivityRegistry stub methods
func (m *mockWorker) RegisterActivity(_ interface{})                                         {}
func (m *mockWorker) RegisterActivityWithOptions(_ interface{}, _ activity.RegisterOptions)  {}
func (m *mockWorker) GetRegisteredActivities() []activity.RegistryInfo                       { return nil }

// ─── helper: build a real serviceClient via fx ───────────────────────────────

func newTestServiceClient(t *testing.T) workflowserviceclient.Interface {
	t.Helper()
	logger := zap.NewNop()
	cfg := NewCadenceConfig()
	var sc workflowserviceclient.Interface
	app := fx.New(
		fx.Provide(
			func() *CadenceConfig { return cfg },
			func() *zap.Logger { return logger },
			NewCadenceServiceClient,
		),
		fx.Populate(&sc),
	)
	if err := app.Err(); err != nil {
		t.Fatalf("failed to build serviceClient: %v", err)
	}
	return sc
}

// ─── RegisterWorkflows / RegisterActivities ───────────────────────────────────

func TestRegisterWorkflowsAndActivities(t *testing.T) {
	RegisterWorkflows()
	RegisterActivities(core.RepoContainer{}, nil, zap.NewNop())
}

// ─── StartWorker ─────────────────────────────────────────────────────────────

func TestStartWorker_NilClientError(t *testing.T) {
	lc := fxtest.NewLifecycle(t)
	_, err := StartWorker(nil, NewCadenceConfig(), zap.NewNop(), core.RepoContainer{}, nil, lc)
	if err == nil {
		t.Error("expected error when serviceClient is nil, got nil")
	}
}

func TestStartWorker_EmptyDomainError(t *testing.T) {
	lc := fxtest.NewLifecycle(t)
	sc := newTestServiceClient(t)
	_, err := StartWorker(sc, &CadenceConfig{Domain: ""}, zap.NewNop(), core.RepoContainer{}, nil, lc)
	if err == nil {
		t.Error("expected error when domain is empty, got nil")
	}
}

func TestStartWorker_WorkerNewFnError(t *testing.T) {
	// Inject a factory that always fails, covering the "Failed to create worker" branch.
	orig := workerNewFn
	defer func() { workerNewFn = orig }()
	wantErr := errors.New("factory error")
	workerNewFn = func(_ workflowserviceclient.Interface, _, _ string, _ worker.Options) (worker.Worker, error) {
		return nil, wantErr
	}

	lc := fxtest.NewLifecycle(t)
	sc := newTestServiceClient(t)
	_, err := StartWorker(sc, NewCadenceConfig(), zap.NewNop(), core.RepoContainer{}, nil, lc)
	if err == nil {
		t.Error("expected error from factory, got nil")
	}
}

func TestStartWorker_HookOnStartError(t *testing.T) {
	// Inject a factory that returns a mock worker whose Start() fails.
	orig := workerNewFn
	defer func() { workerNewFn = orig }()
	startErr := errors.New("start failed")
	mw := &mockWorker{startErr: startErr}
	workerNewFn = func(_ workflowserviceclient.Interface, _, _ string, _ worker.Options) (worker.Worker, error) {
		return mw, nil
	}

	lc := fxtest.NewLifecycle(t)
	sc := newTestServiceClient(t)
	w, err := StartWorker(sc, NewCadenceConfig(), zap.NewNop(), core.RepoContainer{}, nil, lc)
	if err != nil {
		t.Fatalf("StartWorker should succeed even if worker.Start() would fail later; got %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil worker returned from StartWorker")
	}

	// Now trigger the OnStart lifecycle hook, which calls mw.Start() → error.
	ctx := context.Background()
	if err := lc.Start(ctx); err == nil {
		t.Error("expected lifecycle Start to propagate worker.Start() error, got nil")
	}
}

func TestStartWorker_HookOnStartStop_Success(t *testing.T) {
	// Inject a mock worker whose Start() succeeds.
	orig := workerNewFn
	defer func() { workerNewFn = orig }()
	mw := &mockWorker{}
	workerNewFn = func(_ workflowserviceclient.Interface, _, _ string, _ worker.Options) (worker.Worker, error) {
		return mw, nil
	}

	lc := fxtest.NewLifecycle(t)
	sc := newTestServiceClient(t)
	_, err := StartWorker(sc, NewCadenceConfig(), zap.NewNop(), core.RepoContainer{}, nil, lc)
	if err != nil {
		t.Fatalf("StartWorker: %v", err)
	}

	ctx := context.Background()
	if err := lc.Start(ctx); err != nil {
		t.Fatalf("lifecycle Start: %v", err)
	}
	if err := lc.Stop(ctx); err != nil {
		t.Fatalf("lifecycle Stop: %v", err)
	}
	if !mw.stopped {
		t.Error("expected mock worker Stop() to have been called")
	}
}

// ─── StartEmailWorker ─────────────────────────────────────────────────────────

func TestStartEmailWorker_NilClientError(t *testing.T) {
	lc := fxtest.NewLifecycle(t)
	_, err := StartEmailWorker(nil, NewCadenceConfig(), zap.NewNop(), core.RepoContainer{}, nil, lc)
	if err == nil {
		t.Error("expected error when serviceClient is nil, got nil")
	}
}

func TestStartEmailWorker_EmptyDomainError(t *testing.T) {
	lc := fxtest.NewLifecycle(t)
	sc := newTestServiceClient(t)
	_, err := StartEmailWorker(sc, &CadenceConfig{Domain: ""}, zap.NewNop(), core.RepoContainer{}, nil, lc)
	if err == nil {
		t.Error("expected error when domain is empty, got nil")
	}
}

func TestStartEmailWorker_WorkerNewFnError(t *testing.T) {
	orig := workerNewFn
	defer func() { workerNewFn = orig }()
	wantErr := errors.New("email factory error")
	workerNewFn = func(_ workflowserviceclient.Interface, _, _ string, _ worker.Options) (worker.Worker, error) {
		return nil, wantErr
	}

	lc := fxtest.NewLifecycle(t)
	sc := newTestServiceClient(t)
	_, err := StartEmailWorker(sc, NewCadenceConfig(), zap.NewNop(), core.RepoContainer{}, nil, lc)
	if err == nil {
		t.Error("expected error from factory, got nil")
	}
}

func TestStartEmailWorker_HookOnStartError(t *testing.T) {
	orig := workerNewFn
	defer func() { workerNewFn = orig }()
	startErr := errors.New("email start failed")
	mw := &mockWorker{startErr: startErr}
	workerNewFn = func(_ workflowserviceclient.Interface, _, _ string, _ worker.Options) (worker.Worker, error) {
		return mw, nil
	}

	lc := fxtest.NewLifecycle(t)
	sc := newTestServiceClient(t)
	w, err := StartEmailWorker(sc, NewCadenceConfig(), zap.NewNop(), core.RepoContainer{}, nil, lc)
	if err != nil {
		t.Fatalf("StartEmailWorker unexpectedly errored: %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil worker")
	}

	ctx := context.Background()
	if err := lc.Start(ctx); err == nil {
		t.Error("expected lifecycle Start to propagate email worker.Start() error, got nil")
	}
}

func TestStartEmailWorker_HookOnStartStop_Success(t *testing.T) {
	orig := workerNewFn
	defer func() { workerNewFn = orig }()
	mw := &mockWorker{}
	workerNewFn = func(_ workflowserviceclient.Interface, _, _ string, _ worker.Options) (worker.Worker, error) {
		return mw, nil
	}

	lc := fxtest.NewLifecycle(t)
	sc := newTestServiceClient(t)
	_, err := StartEmailWorker(sc, NewCadenceConfig(), zap.NewNop(), core.RepoContainer{}, nil, lc)
	if err != nil {
		t.Fatalf("StartEmailWorker: %v", err)
	}

	ctx := context.Background()
	if err := lc.Start(ctx); err != nil {
		t.Fatalf("lifecycle Start: %v", err)
	}
	if err := lc.Stop(ctx); err != nil {
		t.Fatalf("lifecycle Stop: %v", err)
	}
	if !mw.stopped {
		t.Error("expected email mock worker Stop() to have been called")
	}
}

// ─── attachWorkerLifecycle direct tests ──────────────────────────────────────

func TestAttachWorkerLifecycle_OnStartError(t *testing.T) {
	startErr := errors.New("direct hook error")
	mw := &mockWorker{startErr: startErr}
	cfg := NewCadenceConfig()
	lc := fxtest.NewLifecycle(t)

	attachWorkerLifecycle(lc, mw, cfg, TaskListName, "", zap.NewNop())

	ctx := context.Background()
	if err := lc.Start(ctx); err == nil {
		t.Error("expected error from OnStart, got nil")
	}
}

func TestAttachWorkerLifecycle_OnStop(t *testing.T) {
	mw := &mockWorker{}
	cfg := NewCadenceConfig()
	lc := fxtest.NewLifecycle(t)

	attachWorkerLifecycle(lc, mw, cfg, TaskListName, " email", zap.NewNop())

	ctx := context.Background()
	_ = lc.Start(ctx)
	if err := lc.Stop(ctx); err != nil {
		t.Fatalf("unexpected Stop error: %v", err)
	}
	if !mw.stopped {
		t.Error("expected Stop() to have been called on mock worker")
	}
}
