package workflows

import (
	"time"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/activities"
	"go.uber.org/cadence/workflow"
)

func HelloWorldWorkflow(ctx workflow.Context, name string) (string, error) {
	opt := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, opt)

	var result string
	err := workflow.ExecuteActivity(ctx, activities.HelloWorldActivity, name).Get(ctx, &result)
	if err != nil {
		return "", err
	}
	return result, nil
}
