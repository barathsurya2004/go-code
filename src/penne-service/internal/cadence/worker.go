package cadence

import (
	"errors"
	"sync"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/activities"
	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/workflows"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

var registerOnce sync.Once

// RegisterWorkflowsAndActivities registers all workflows and activities with Cadence.
func RegisterWorkflowsAndActivities() {
	registerOnce.Do(func() {
		workflow.RegisterWithOptions(workflows.HelloWorldWorkflow, workflow.RegisterOptions{Name: "HelloWorldWorkflow"})
		activity.RegisterWithOptions(activities.HelloWorldActivity, activity.RegisterOptions{Name: "HelloWorldActivity"})
	})
}

// StartWorker creates, registers, and starts a standalone Cadence worker instance.
func StartWorker(serviceClient workflowserviceclient.Interface, cfg *CadenceConfig, logger *zap.Logger) (worker.Worker, error) {
	if serviceClient == nil {
		return nil, errors.New("serviceClient is required")
	}
	RegisterWorkflowsAndActivities()

	workerOptions := worker.Options{
		Logger: logger,
	}

	w, err := worker.NewV2(serviceClient, cfg.Domain, TaskListName, workerOptions)
	if err != nil {
		logger.Error("Failed to create Cadence worker", zap.Error(err))
		return nil, err
	}

	if err := w.Start(); err != nil {
		logger.Error("Failed to start Cadence worker", zap.Error(err))
		return nil, err
	}

	logger.Info("Cadence worker successfully started", zap.String("domain", cfg.Domain), zap.String("task_list", TaskListName))
	return w, nil
}
