package dsl

import "errors"

// Dependency graph errors
var (
	ErrCircularDependency = errors.New("circular dependency detected")
	ErrJobNotFound        = errors.New("job not found in dependency graph")
)

// DependencyGraph manages job dependencies and execution order
type DependencyGraph struct {
	nodes map[string]*JobNode
	edges map[string][]string // job → dependencies
}

// JobNode represents a job in the dependency graph
type JobNode struct {
	Job     *Job
	Status  string // pending, running, completed, failed
	Outputs map[string]string
}

// NewDependencyGraph creates a dependency graph from workflow
func NewDependencyGraph(workflow *Workflow) *DependencyGraph {
	graph := &DependencyGraph{
		nodes: make(map[string]*JobNode),
		edges: make(map[string][]string),
	}

	for jobName, job := range workflow.Jobs {
		graph.nodes[jobName] = &JobNode{
			Job:    job,
			Status: "pending",
		}

		if len(job.Needs) > 0 {
			graph.edges[jobName] = job.Needs
		}
	}

	return graph
}

// detectCycle performs DFS to detect circular dependencies
func (g *DependencyGraph) detectCycle() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(string) bool
	dfs = func(jobName string) bool {
		visited[jobName] = true
		recStack[jobName] = true

		for _, dep := range g.edges[jobName] {
			if !visited[dep] {
				if dfs(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[jobName] = false
		return false
	}

	for jobName := range g.nodes {
		if !visited[jobName] {
			if dfs(jobName) {
				return ErrCircularDependency
			}
		}
	}

	return nil
}

// ValidateDependencies checks for invalid dependencies and cycles
func (g *DependencyGraph) ValidateDependencies() error {
	// Check if all dependencies exist
	for _, deps := range g.edges {
		for _, dep := range deps {
			if _, exists := g.nodes[dep]; !exists {
				return ErrJobNotFound
			}
		}
	}

	// Detect circular dependencies
	return g.detectCycle()
}

// GetReadyJobs returns jobs ready to execute (all dependencies completed)
func (g *DependencyGraph) GetReadyJobs() []string {
	ready := make([]string, 0)

	for jobName, node := range g.nodes {
		if node.Status != "pending" {
			continue
		}

		// Check if all dependencies are completed
		dependencies := g.edges[jobName]
		allDepsCompleted := true

		for _, dep := range dependencies {
			depNode := g.nodes[dep]
			if depNode == nil || depNode.Status != "completed" {
				allDepsCompleted = false
				break
			}
		}

		if allDepsCompleted {
			ready = append(ready, jobName)
		}
	}

	return ready
}

// MarkCompleted marks a job as completed
func (g *DependencyGraph) MarkCompleted(jobName string, outputs map[string]string) {
	if node, exists := g.nodes[jobName]; exists {
		node.Status = "completed"
		node.Outputs = outputs
	}
}

// MarkRunning marks a job as running
func (g *DependencyGraph) MarkRunning(jobName string) {
	if node, exists := g.nodes[jobName]; exists {
		node.Status = "running"
	}
}

// MarkFailed marks a job as failed
func (g *DependencyGraph) MarkFailed(jobName string) {
	if node, exists := g.nodes[jobName]; exists {
		node.Status = "failed"
	}
}

// GetDependentJobs returns all jobs that depend on the given job
func (g *DependencyGraph) GetDependentJobs(jobName string) []string {
	dependents := make([]string, 0)

	for jName, deps := range g.edges {
		for _, dep := range deps {
			if dep == jobName {
				dependents = append(dependents, jName)
				break
			}
		}
	}

	return dependents
}

// GetJobOutputs returns outputs for a specific job
func (g *DependencyGraph) GetJobOutputs(jobName string) map[string]string {
	if node, exists := g.nodes[jobName]; exists {
		return node.Outputs
	}
	return nil
}

// AllJobsCompleted returns true if all jobs are completed or failed
func (g *DependencyGraph) AllJobsCompleted() bool {
	for _, node := range g.nodes {
		if node.Status != "completed" && node.Status != "failed" {
			return false
		}
	}
	return true
}
