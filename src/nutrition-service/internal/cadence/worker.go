package cadence

import (
	"context"
	"errors"
	"sync"

	"github.com/barathsurya2004/go-code/nutrition-service/internal/cadence/activities"
	"github.com/barathsurya2004/go-code/nutrition-service/internal/cadence/workflows"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var registerWorkflowOnce sync.Once
var registerActivitiesOnce sync.Once

// RegisterActivities registers all nutrition activities with Cadence.
func RegisterActivities(acts *activities.NutritionActivities) {
	registerActivitiesOnce.Do(func() {
		activity.RegisterWithOptions(acts.UpdateJobStatus, activity.RegisterOptions{Name: "UpdateJobStatusActivity"})
		activity.RegisterWithOptions(acts.AnalyzeMealImage, activity.RegisterOptions{Name: "AnalyzeMealImageActivity"})
		activity.RegisterWithOptions(acts.PurgeRawImage, activity.RegisterOptions{Name: "PurgeRawImageActivity"})
		activity.RegisterWithOptions(acts.PersistMealLog, activity.RegisterOptions{Name: "PersistMealLogActivity"})
		activity.RegisterWithOptions(acts.CompleteJob, activity.RegisterOptions{Name: "CompleteJobActivity"})
	})
}

// RegisterWorkflows registers all nutrition workflows with Cadence.
func RegisterWorkflows() {
	registerWorkflowOnce.Do(func() {
		workflow.RegisterWithOptions(workflows.ProcessMealImageWorkflow, workflow.RegisterOptions{Name: "ProcessMealImageWorkflow"})
	})
}

var workerNewFn = func(svc workflowserviceclient.Interface, domain, taskList string, opts worker.Options) (worker.Worker, error) {
	return worker.NewV2(svc, domain, taskList, opts)
}

// StartCadenceWorker creates, registers, and starts the Cadence worker listening on nouri-tasklist.
func StartCadenceWorker(
	serviceClient workflowserviceclient.Interface,
	cfg *CadenceConfig,
	acts *activities.NutritionActivities,
	logger *zap.Logger,
	lc fx.Lifecycle,
) (worker.Worker, error) {
	if serviceClient == nil {
		return nil, errors.New("serviceClient is required")
	}
	if cfg == nil || cfg.Domain == "" {
		return nil, errors.New("domain is required")
	}

	RegisterWorkflows()
	RegisterActivities(acts)

	w, err := workerNewFn(serviceClient, cfg.Domain, cfg.TaskListName, worker.Options{Logger: logger})
	if err != nil {
		logger.Error("Failed to initialize Cadence worker", zap.Error(err), zap.String("domain", cfg.Domain))
		return nil, err
	}

	if lc != nil {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				if err := w.Start(); err != nil {
					logger.Error("Failed to start Cadence worker", zap.Error(err), zap.String("domain", cfg.Domain))
					return err
				}
				logger.Info("Cadence worker successfully started",
					zap.String("domain", cfg.Domain),
					zap.String("task_list", cfg.TaskListName),
				)
				return nil
			},
			OnStop: func(ctx context.Context) error {
				w.Stop()
				logger.Info("Cadence worker stopped successfully",
					zap.String("domain", cfg.Domain),
					zap.String("task_list", cfg.TaskListName),
				)
				return nil
			},
		})
	}

	return w, nil
}
