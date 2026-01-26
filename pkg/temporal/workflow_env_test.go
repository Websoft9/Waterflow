package temporal

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

// WorkflowTestSuite is the test suite for workflow execution using Temporal SDK testsuite.
type WorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

// TestWorkflowTestSuite runs the test suite.
func TestWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}

// TestRunWorkflowExecutor_CircularDependency tests circular dependency detection.
func (s *WorkflowTestSuite) TestRunWorkflowExecutor_CircularDependency() {
	env := s.NewTestWorkflowEnvironment()

	wf := &dsl.Workflow{
		Name: "circular-dependency-workflow",
		Jobs: map[string]*dsl.Job{
			"job-a": {
				Name:   "job-a",
				RunsOn: "linux",
				Needs:  []string{"job-b"},
				Steps:  []*dsl.Step{{ID: "step", Uses: "exec"}},
			},
			"job-b": {
				Name:   "job-b",
				RunsOn: "linux",
				Needs:  []string{"job-a"},
				Steps:  []*dsl.Step{{ID: "step", Uses: "exec"}},
			},
		},
	}

	env.ExecuteWorkflow(RunWorkflowExecutor, wf)

	s.True(env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	s.Error(err)
	s.Contains(err.Error(), "circular")
}

// TestRunWorkflowExecutor_EmptyWorkflow tests empty workflow.
func (s *WorkflowTestSuite) TestRunWorkflowExecutor_EmptyWorkflow() {
	env := s.NewTestWorkflowEnvironment()

	wf := &dsl.Workflow{
		Name: "empty-workflow",
		Jobs: map[string]*dsl.Job{},
	}

	env.ExecuteWorkflow(RunWorkflowExecutor, wf)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
}

// TestRunWorkflowExecutor_JobConditionSkip tests job-level if condition that skips job.
func (s *WorkflowTestSuite) TestRunWorkflowExecutor_JobConditionSkip() {
	env := s.NewTestWorkflowEnvironment()

	wf := &dsl.Workflow{
		Name: "conditional-workflow",
		Vars: map[string]interface{}{
			"skip_job": true,
		},
		Jobs: map[string]*dsl.Job{
			"build": {
				Name:   "build",
				RunsOn: "linux",
				If:     "vars.skip_job == false", // Evaluates to false, so job is skipped
				Steps: []*dsl.Step{
					{ID: "step", Name: "Step", Uses: "exec"},
				},
			},
		},
	}

	env.ExecuteWorkflow(RunWorkflowExecutor, wf)

	s.True(env.IsWorkflowCompleted())
	s.NoError(env.GetWorkflowError())
}

// TestRunWorkflowExecutor_InvalidDependency tests invalid dependency detection.
func (s *WorkflowTestSuite) TestRunWorkflowExecutor_InvalidDependency() {
	env := s.NewTestWorkflowEnvironment()

	wf := &dsl.Workflow{
		Name: "invalid-dependency-workflow",
		Jobs: map[string]*dsl.Job{
			"build": {
				Name:   "build",
				RunsOn: "linux",
				Needs:  []string{"nonexistent-job"}, // References non-existent job
				Steps:  []*dsl.Step{{ID: "step", Uses: "exec"}},
			},
		},
	}

	env.ExecuteWorkflow(RunWorkflowExecutor, wf)

	s.True(env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	s.Error(err)
}

// TestRunWorkflowExecutor_SelfDependency tests self-dependency detection.
func (s *WorkflowTestSuite) TestRunWorkflowExecutor_SelfDependency() {
	env := s.NewTestWorkflowEnvironment()

	wf := &dsl.Workflow{
		Name: "self-dependency-workflow",
		Jobs: map[string]*dsl.Job{
			"build": {
				Name:   "build",
				RunsOn: "linux",
				Needs:  []string{"build"}, // Self dependency
				Steps:  []*dsl.Step{{ID: "step", Uses: "exec"}},
			},
		},
	}

	env.ExecuteWorkflow(RunWorkflowExecutor, wf)

	s.True(env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	s.Error(err)
}
