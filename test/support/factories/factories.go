// Package factories provides test data factories for Waterflow tests.
package factories

import (
	"fmt"
	"time"
)

// WorkflowFactory creates test workflow YAML definitions.
type WorkflowFactory struct {
	name string
	jobs []JobConfig
	vars map[string]interface{}
	on   string
}

// JobConfig represents a job configuration for factory.
type JobConfig struct {
	ID        string
	Name      string
	RunsOn    string
	Steps     []StepConfig
	DependsOn []string
}

// StepConfig represents a step configuration for factory.
type StepConfig struct {
	Name string
	Uses string
	With map[string]interface{}
}

// NewWorkflowFactory creates a new workflow factory with defaults.
func NewWorkflowFactory() *WorkflowFactory {
	return &WorkflowFactory{
		name: "test-workflow",
		jobs: make([]JobConfig, 0),
		vars: make(map[string]interface{}),
		on:   "workflow_dispatch",
	}
}

// WithName sets the workflow name.
func (f *WorkflowFactory) WithName(name string) *WorkflowFactory {
	f.name = name
	return f
}

// WithTrigger sets the workflow trigger.
func (f *WorkflowFactory) WithTrigger(trigger string) *WorkflowFactory {
	f.on = trigger
	return f
}

// AddSimpleJob adds a simple echo job.
func (f *WorkflowFactory) AddSimpleJob(id, runsOn, message string) *WorkflowFactory {
	f.jobs = append(f.jobs, JobConfig{
		ID:     id,
		Name:   fmt.Sprintf("Job %s", id),
		RunsOn: runsOn,
		Steps: []StepConfig{
			{
				Name: "Echo message",
				Uses: "shell@v1",
				With: map[string]interface{}{
					"command": fmt.Sprintf("echo '%s'", message),
				},
			},
		},
	})
	return f
}

// Build generates the workflow YAML.
func (f *WorkflowFactory) Build() string {
	yaml := fmt.Sprintf("name: %s\n", f.name)
	yaml += fmt.Sprintf("on: %s\n", f.on)
	yaml += "jobs:\n"
	for _, job := range f.jobs {
		yaml += fmt.Sprintf("  %s:\n", job.ID)
		if job.Name != "" {
			yaml += fmt.Sprintf("    name: %s\n", job.Name)
		}
		yaml += fmt.Sprintf("    runs-on: %s\n", job.RunsOn)
		yaml += "    steps:\n"
		for _, step := range job.Steps {
			yaml += fmt.Sprintf("      - name: %s\n", step.Name)
			yaml += fmt.Sprintf("        uses: %s\n", step.Uses)
			if len(step.With) > 0 {
				yaml += "        with:\n"
				for k, v := range step.With {
					yaml += fmt.Sprintf("          %s: %v\n", k, v)
				}
			}
		}
	}
	return yaml
}

// HelloWorldWorkflow returns a simple hello world workflow.
func HelloWorldWorkflow() string {
	return NewWorkflowFactory().
		WithName("hello-world").
		AddSimpleJob("greet", "default", "Hello, World!").
		Build()
}

// TestUser represents a test user for API tests.
type TestUser struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
}

// UserFactory creates test users.
type UserFactory struct {
	counter int
}

// NewUserFactory creates a new user factory.
func NewUserFactory() *UserFactory {
	return &UserFactory{}
}

// Create creates a new test user with auto-generated data.
func (f *UserFactory) Create() TestUser {
	f.counter++
	return TestUser{
		ID:        fmt.Sprintf("user-%d", f.counter),
		Email:     fmt.Sprintf("test%d@example.com", f.counter),
		Name:      fmt.Sprintf("Test User %d", f.counter),
		CreatedAt: time.Now(),
	}
}

// ConfigFactory creates test configuration objects.
type ConfigFactory struct{}

// NewConfigFactory creates a new config factory.
func NewConfigFactory() *ConfigFactory {
	return &ConfigFactory{}
}

// ServerConfig creates a test server configuration.
func (f *ConfigFactory) ServerConfig(port int) map[string]interface{} {
	return map[string]interface{}{
		"host":             "localhost",
		"port":             port,
		"read_timeout":     "10s",
		"write_timeout":    "10s",
		"shutdown_timeout": "5s",
	}
}

// AgentConfig creates a test agent configuration.
func (f *ConfigFactory) AgentConfig(taskQueues []string) map[string]interface{} {
	return map[string]interface{}{
		"task_queues":      taskQueues,
		"plugin_dir":       "/tmp/plugins",
		"shutdown_timeout": "30s",
	}
}

// TemporalConfig creates a test Temporal configuration.
func (f *ConfigFactory) TemporalConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"host":               host,
		"namespace":          "waterflow-test",
		"connection_timeout": "10s",
		"max_retries":        3,
		"retry_interval":     "1s",
	}
}

// StepFactory creates test step configurations.
type StepFactory struct {
	counter int
}

// NewStepFactory creates a new step factory.
func NewStepFactory() *StepFactory {
	return &StepFactory{}
}

// ShellStep creates a shell execution step.
func (f *StepFactory) ShellStep(command string) StepConfig {
	f.counter++
	return StepConfig{
		Name: fmt.Sprintf("Shell Step %d", f.counter),
		Uses: "shell@v1",
		With: map[string]interface{}{
			"command": command,
		},
	}
}

// HTTPStep creates an HTTP request step.
func (f *StepFactory) HTTPStep(method, url string) StepConfig {
	f.counter++
	return StepConfig{
		Name: fmt.Sprintf("HTTP Step %d", f.counter),
		Uses: "http@v1",
		With: map[string]interface{}{
			"method": method,
			"url":    url,
		},
	}
}

// SleepStep creates a sleep/delay step.
func (f *StepFactory) SleepStep(duration string) StepConfig {
	f.counter++
	return StepConfig{
		Name: fmt.Sprintf("Sleep Step %d", f.counter),
		Uses: "sleep@v1",
		With: map[string]interface{}{
			"duration": duration,
		},
	}
}

// DockerStep creates a Docker execution step.
func (f *StepFactory) DockerStep(image, command string) StepConfig {
	f.counter++
	return StepConfig{
		Name: fmt.Sprintf("Docker Step %d", f.counter),
		Uses: "docker@v1",
		With: map[string]interface{}{
			"image":   image,
			"command": command,
		},
	}
}
