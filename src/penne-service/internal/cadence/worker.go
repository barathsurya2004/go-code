package cadence

import (
	"context"
	"errors"
	"sync"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/activities"
	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/workflows"
	"github.com/barathsurya2004/go-code/penne-service/internal/core"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var registerWorkflowOnce sync.Once
var registerActivitiesOnce sync.Once

// RegisterWorkflowsAndActivities registers all workflows and activities with Cadence.
func RegisterActivities(repos core.RepoContainer) {
	transactionAct := activities.NewTransactionActivities(repos)
	registerActivitiesOnce.Do(func() {
		activity.RegisterWithOptions(activities.HelloWorldActivity, activity.RegisterOptions{Name: "HelloWorldActivity"})
		activity.RegisterWithOptions(transactionAct.CreateTransaction, activity.RegisterOptions{Name: "CreateTransaction"})
	})
}

func RegisterWorkflows() {
	registerWorkflowOnce.Do(func() {
		workflow.RegisterWithOptions(workflows.CreateTransactionWorkflow, workflow.RegisterOptions{Name: "CreateTransactionWorkflow"})
		workflow.RegisterWithOptions(workflows.HelloWorldWorkflow, workflow.RegisterOptions{Name: "HelloWorldWorkflow"})
	})

}

// StartWorker creates, registers, and starts a standalone Cadence worker instance.
func StartWorker(serviceClient workflowserviceclient.Interface, cfg *CadenceConfig, logger *zap.Logger, repos core.RepoContainer, lc fx.Lifecycle) (worker.Worker, error) {
	if serviceClient == nil {
		return nil, errors.New("serviceClient is required")
	}

	RegisterWorkflows()
	RegisterActivities(repos)

	workerOptions := worker.Options{
		Logger: logger,
	}

	w, err := worker.NewV2(serviceClient, cfg.Domain, TaskListName, workerOptions)
	if err != nil {
		logger.Error("Failed to create Cadence worker", zap.Error(err))
		return nil, err
	}

	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				if err := w.Start(); err != nil {
					logger.Error("Failed to start Cadence worker", zap.Error(err))
					return err
				}
				return nil
			},
			OnStop: func(ctx context.Context) error {
				w.Stop()
				logger.Info("Cadence worker stopped successfully", zap.String("domain", cfg.Domain), zap.String("task_list", TaskListName))
				return nil
			},
		})

	logger.Info("Cadence worker successfully started", zap.String("domain", cfg.Domain), zap.String("task_list", TaskListName))
	return w, nil
}
