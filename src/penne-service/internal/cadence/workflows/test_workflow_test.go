package workflows

import (
	"errors"
	"testing"

	"github.com/barathsurya2004/go-code/penne-service/internal/cadence/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/testsuite"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

func (s *UnitTestSuite) Test_HelloWorldWorkflow() {
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(activities.HelloWorldActivity)

	env.ExecuteWorkflow(HelloWorldWorkflow, "Barath")

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())

	var result string
	s.NoError(env.GetWorkflowResult(&result))
	s.Equal("Hello Barath\n", result)
}

func (s *UnitTestSuite) Test_HelloWorldWorkflow_Error() {
	env := s.NewTestWorkflowEnvironment()
	env.OnActivity(activities.HelloWorldActivity, mock.Anything, "ErrorUser").Return("", errors.New("activity failed"))

	env.ExecuteWorkflow(HelloWorldWorkflow, "ErrorUser")

	s.True(env.IsWorkflowCompleted())
	s.Error(env.GetWorkflowError())
}
