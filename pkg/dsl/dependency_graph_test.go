package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDependencyGraph_CircularDependency(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a", Needs: []string{"b"}},
			"b": {Name: "b", Needs: []string{"c"}},
			"c": {Name: "c", Needs: []string{"a"}}, // circular!
		},
	}

	graph := NewDependencyGraph(workflow)
	err := graph.ValidateDependencies()

	assert.Error(t, err)
	assert.Equal(t, ErrCircularDependency, err)
}

func TestDependencyGraph_SelfDependency(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a", Needs: []string{"a"}}, // self dependency
		},
	}

	graph := NewDependencyGraph(workflow)
	err := graph.ValidateDependencies()

	assert.Error(t, err)
	assert.Equal(t, ErrCircularDependency, err)
}

func TestDependencyGraph_JobNotFound(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a", Needs: []string{"nonexistent"}},
		},
	}

	graph := NewDependencyGraph(workflow)
	err := graph.ValidateDependencies()

	assert.Error(t, err)
	assert.Equal(t, ErrJobNotFound, err)
}

func TestDependencyGraph_ValidDependencies(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a"},
			"b": {Name: "b", Needs: []string{"a"}},
			"c": {Name: "c", Needs: []string{"a", "b"}},
		},
	}

	graph := NewDependencyGraph(workflow)
	err := graph.ValidateDependencies()

	assert.NoError(t, err)
}

func TestDependencyGraph_GetReadyJobsEmpty(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{},
	}

	graph := NewDependencyGraph(workflow)
	ready := graph.GetReadyJobs()

	assert.Empty(t, ready)
}

func TestDependencyGraph_GetReadyJobsNoDependencies(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a"},
			"b": {Name: "b"},
		},
	}

	graph := NewDependencyGraph(workflow)
	ready := graph.GetReadyJobs()

	assert.Len(t, ready, 2)
	assert.Contains(t, ready, "a")
	assert.Contains(t, ready, "b")
}

func TestDependencyGraph_MarkRunning(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a"},
		},
	}

	graph := NewDependencyGraph(workflow)
	graph.MarkRunning("a")

	assert.Equal(t, "running", graph.nodes["a"].Status)

	// Should not be in ready jobs anymore
	ready := graph.GetReadyJobs()
	assert.NotContains(t, ready, "a")
}

func TestDependencyGraph_MarkFailed(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a"},
		},
	}

	graph := NewDependencyGraph(workflow)
	graph.MarkFailed("a")

	assert.Equal(t, "failed", graph.nodes["a"].Status)
}

func TestDependencyGraph_GetDependentJobs(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a"},
			"b": {Name: "b", Needs: []string{"a"}},
			"c": {Name: "c", Needs: []string{"a"}},
			"d": {Name: "d", Needs: []string{"b"}},
		},
	}

	graph := NewDependencyGraph(workflow)

	dependents := graph.GetDependentJobs("a")
	assert.Len(t, dependents, 2)
	assert.Contains(t, dependents, "b")
	assert.Contains(t, dependents, "c")

	dependents = graph.GetDependentJobs("b")
	assert.Len(t, dependents, 1)
	assert.Contains(t, dependents, "d")
}

func TestDependencyGraph_GetJobOutputs(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a"},
		},
	}

	graph := NewDependencyGraph(workflow)

	// Before marking completed
	outputs := graph.GetJobOutputs("a")
	assert.Nil(t, outputs)

	// After marking completed
	expectedOutputs := map[string]string{"version": "1.0.0"}
	graph.MarkCompleted("a", expectedOutputs)

	outputs = graph.GetJobOutputs("a")
	assert.Equal(t, expectedOutputs, outputs)
}

func TestDependencyGraph_GetJobOutputsNonexistent(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{},
	}

	graph := NewDependencyGraph(workflow)
	outputs := graph.GetJobOutputs("nonexistent")

	assert.Nil(t, outputs)
}

func TestDependencyGraph_AllJobsCompleted(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a"},
			"b": {Name: "b"},
		},
	}

	graph := NewDependencyGraph(workflow)

	assert.False(t, graph.AllJobsCompleted())

	graph.MarkCompleted("a", nil)
	assert.False(t, graph.AllJobsCompleted())

	graph.MarkCompleted("b", nil)
	assert.True(t, graph.AllJobsCompleted())
}

func TestDependencyGraph_AllJobsCompletedWithFailures(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"a": {Name: "a"},
			"b": {Name: "b"},
		},
	}

	graph := NewDependencyGraph(workflow)

	graph.MarkCompleted("a", nil)
	graph.MarkFailed("b")

	assert.True(t, graph.AllJobsCompleted())
}

func TestDependencyGraph_ComplexDependencyChain(t *testing.T) {
	workflow := &Workflow{
		Name: "Test",
		Jobs: map[string]*Job{
			"build":  {Name: "build"},
			"test":   {Name: "test", Needs: []string{"build"}},
			"deploy": {Name: "deploy", Needs: []string{"test"}},
		},
	}

	graph := NewDependencyGraph(workflow)

	// Initially only build should be ready
	ready := graph.GetReadyJobs()
	assert.Equal(t, []string{"build"}, ready)

	// Complete build
	graph.MarkCompleted("build", nil)
	ready = graph.GetReadyJobs()
	assert.Equal(t, []string{"test"}, ready)

	// Complete test
	graph.MarkCompleted("test", nil)
	ready = graph.GetReadyJobs()
	assert.Equal(t, []string{"deploy"}, ready)

	// Complete deploy
	graph.MarkCompleted("deploy", nil)
	ready = graph.GetReadyJobs()
	assert.Empty(t, ready)
	assert.True(t, graph.AllJobsCompleted())
}
