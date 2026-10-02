package cadence

import (
	"context"
	"database/sql"
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

// RegisterActivities registers all activities with Cadence.
func RegisterActivities(repos core.RepoContainer, db *sql.DB, logger *zap.Logger) {
	transactionAct := activities.NewTransactionActivities(repos, logger)
	userAct := activities.NewUserActivities(repos, logger)
	wishlistAct := activities.NewWishlistActivities(repos, db, logger)
	emailAct := activities.NewEmailActivities(logger)
	subscriptionAct := activities.NewSubscriptionActivities(repos, logger)
	registerActivitiesOnce.Do(func() {
		activity.RegisterWithOptions(activities.HelloWorldActivity, activity.RegisterOptions{Name: "HelloWorldActivity"})
		activity.RegisterWithOptions(transactionAct.CreateTransaction, activity.RegisterOptions{Name: "CreateTransactionActivity"})
		activity.RegisterWithOptions(transactionAct.UpdateShortcutIntentActivity, activity.RegisterOptions{Name: "UpdateShortcutIntentActivity"})
		activity.RegisterWithOptions(transactionAct.PendingShortcutIntentActivity, activity.RegisterOptions{Name: "PendingShortcutIntentActivity"})
		activity.RegisterWithOptions(transactionAct.CreateShortcutIntent, activity.RegisterOptions{Name: "CreateShortcutIntentActivity"})
		activity.RegisterWithOptions(transactionAct.GetTransactionByTimeActivity, activity.RegisterOptions{Name: "GetTransactionByTimeActivity"})
		activity.RegisterWithOptions(transactionAct.UpdateTransactionActivity, activity.RegisterOptions{Name: "UpdateTransactionActivity"})
		activity.RegisterWithOptions(transactionAct.GetTransactionByIDActivity, activity.RegisterOptions{Name: "GetTransactionByIDActivity"})
		activity.RegisterWithOptions(transactionAct.UpdateAllocationSpentActivity, activity.RegisterOptions{Name: "UpdateAllocationSpentActivity"})
		activity.RegisterWithOptions(transactionAct.FundWishlistItemActivity, activity.RegisterOptions{Name: "FundWishlistItemActivity"})
		activity.RegisterWithOptions(transactionAct.UpdateWishlistSpentActivity, activity.RegisterOptions{Name: "UpdateWishlistSpentActivity"})

		activity.RegisterWithOptions(userAct.CreateUserActivity, activity.RegisterOptions{Name: "CreateUserActivity"})
		activity.RegisterWithOptions(userAct.CreateSystemEnvelopeGroupActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeGroupActivity"})
		activity.RegisterWithOptions(userAct.CreateSystemEnvelopeActivity, activity.RegisterOptions{Name: "CreateSystemEnvelopeActivity"})
		activity.RegisterWithOptions(userAct.CreateDefaultAllocationActivity, activity.RegisterOptions{Name: "CreateDefaultAllocationActivity"})
		activity.RegisterWithOptions(userAct.CreateUserTokenActivity, activity.RegisterOptions{Name: "CreateUserTokenActivity"})

		activity.RegisterWithOptions(wishlistAct.CalculateWishlistForecastActivity, activity.RegisterOptions{Name: "CalculateWishlistForecastActivity"})
		activity.RegisterWithOptions(wishlistAct.ApplyWishlistSurplusActivity, activity.RegisterOptions{Name: "ApplyWishlistSurplusActivity"})

		activity.RegisterWithOptions(subscriptionAct.CreateSubscriptionActivity, activity.RegisterOptions{Name: "CreateSubscriptionActivity"})
		activity.RegisterWithOptions(subscriptionAct.GetSubscriptionByIDActivity, activity.RegisterOptions{Name: "GetSubscriptionByIDActivity"})
		activity.RegisterWithOptions(subscriptionAct.RenewSubscriptionActivity, activity.RegisterOptions{Name: "RenewSubscriptionActivity"})
		activity.RegisterWithOptions(subscriptionAct.GetDueSubscriptionsActivity, activity.RegisterOptions{Name: "GetDueSubscriptionsActivity"})
		activity.RegisterWithOptions(subscriptionAct.MatchSubscriptionIntentActivity, activity.RegisterOptions{Name: "MatchSubscriptionIntentActivity"})
		activity.RegisterWithOptions(subscriptionAct.RecordSubscriptionChargeActivity, activity.RegisterOptions{Name: "RecordSubscriptionChargeActivity"})

		activity.RegisterWithOptions(emailAct.ParseEmailActivity, activity.RegisterOptions{Name: "ParseEmailActivity"})
	})
}

// RegisterWorkflows registers all workflows with Cadence.
func RegisterWorkflows() {
	registerWorkflowOnce.Do(func() {
		workflow.RegisterWithOptions(workflows.CreateTransactionWorkflow, workflow.RegisterOptions{Name: "CreateTransactionWorkflow"})
		workflow.RegisterWithOptions(workflows.CreateShortcutIntentWorkflow, workflow.RegisterOptions{Name: "CreateShortcutIntentWorkflow"})
		workflow.RegisterWithOptions(workflows.HelloWorldWorkflow, workflow.RegisterOptions{Name: "HelloWorldWorkflow"})
		workflow.RegisterWithOptions(workflows.CreateUserWorkflow, workflow.RegisterOptions{Name: "CreateUserWorkflow"})
		workflow.RegisterWithOptions(workflows.SettleWishlistWorkflow, workflow.RegisterOptions{Name: "SettleWishlistWorkflow"})
		workflow.RegisterWithOptions(workflows.ProcessEmailWorkflow, workflow.RegisterOptions{Name: "ProcessEmailWorkflow"})
		workflow.RegisterWithOptions(workflows.UpdateTransactionCategoryWorkflow, workflow.RegisterOptions{Name: "UpdateTransactionCategoryWorkflow"})
		workflow.RegisterWithOptions(workflows.CreateSubscriptionWorkflow, workflow.RegisterOptions{Name: "CreateSubscriptionWorkflow"})
		workflow.RegisterWithOptions(workflows.RenewSubscriptionWorkflow, workflow.RegisterOptions{Name: "RenewSubscriptionWorkflow"})
		workflow.RegisterWithOptions(workflows.ScanAndRenewDueSubscriptionsWorkflow, workflow.RegisterOptions{Name: "ScanAndRenewDueSubscriptionsWorkflow"})
	})
}

// workerNewFn is the factory used to create a Cadence worker. It is a package-level
// variable so tests can inject a mock factory without connecting to a live Cadence server.
var workerNewFn = func(svc workflowserviceclient.Interface, domain, taskList string, opts worker.Options) (worker.Worker, error) {
	return worker.NewV2(svc, domain, taskList, opts)
}

// attachWorkerLifecycle appends OnStart and OnStop lifecycle hooks for w to lc.
// logLabel (e.g., "" or " email") is embedded in log messages.
func attachWorkerLifecycle(lc fx.Lifecycle, w worker.Worker, cfg *CadenceConfig, taskList, logLabel string, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := w.Start(); err != nil {
				logger.Error("Failed to start Cadence"+logLabel+" worker", zap.Error(err))
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			w.Stop()
			logger.Info("Cadence"+logLabel+" worker stopped successfully",
				zap.String("domain", cfg.Domain),
				zap.String("task_list", taskList))
			return nil
		},
	})
}

// StartWorker creates, registers, and starts a standalone Cadence worker instance.
func StartWorker(serviceClient workflowserviceclient.Interface, cfg *CadenceConfig, logger *zap.Logger, repos core.RepoContainer, db *sql.DB, lc fx.Lifecycle) (worker.Worker, error) {
	if serviceClient == nil {
		return nil, errors.New("serviceClient is required")
	}
	if cfg == nil || cfg.Domain == "" {
		return nil, errors.New("domain is required")
	}

	RegisterWorkflows()
	RegisterActivities(repos, db, logger)

	w, err := workerNewFn(serviceClient, cfg.Domain, TaskListName, worker.Options{Logger: logger})
	if err != nil {
		logger.Error("Failed to create Cadence worker", zap.Error(err))
		return nil, err
	}

	attachWorkerLifecycle(lc, w, cfg, TaskListName, "", logger)
	logger.Info("Cadence worker successfully started", zap.String("domain", cfg.Domain), zap.String("task_list", TaskListName))
	return w, nil
}

// StartEmailWorker creates, registers, and starts a standalone Cadence email worker instance.
func StartEmailWorker(serviceClient workflowserviceclient.Interface, cfg *CadenceConfig, logger *zap.Logger, repos core.RepoContainer, db *sql.DB, lc fx.Lifecycle) (worker.Worker, error) {
	if serviceClient == nil {
		return nil, errors.New("serviceClient is required")
	}
	if cfg == nil || cfg.Domain == "" {
		return nil, errors.New("domain is required")
	}

	RegisterWorkflows()
	RegisterActivities(repos, db, logger)

	w, err := workerNewFn(serviceClient, cfg.Domain, EmailTaskListName, worker.Options{Logger: logger})
	if err != nil {
		logger.Error("Failed to create Cadence email worker", zap.Error(err))
		return nil, err
	}

	attachWorkerLifecycle(lc, w, cfg, EmailTaskListName, " email", logger)
	logger.Info("Cadence email worker successfully started", zap.String("domain", cfg.Domain), zap.String("task_list", EmailTaskListName))
	return w, nil
}
