// Package testutil provides testing utilities for Waterflow tests
// This file contains factory functions for creating test data
package testutil

import (
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ========================================
// Workflow Test Data Factory
// ========================================

// WorkflowInputs represents inputs for workflow tests
type WorkflowInputs struct {
	Name      string
	ID        string
	TaskQueue string
	YAML      string
}

// WorkflowInputOption is a functional option for WorkflowInputs
type WorkflowInputOption func(*WorkflowInputs)

// NewWorkflowInputs creates workflow inputs with defaults and options
func NewWorkflowInputs(opts ...WorkflowInputOption) *WorkflowInputs {
	w := &WorkflowInputs{
		Name:      fmt.Sprintf("test-workflow-%d", rand.Intn(10000)),
		ID:        fmt.Sprintf("wf-%d-%d", time.Now().Unix(), rand.Intn(1000)),
		TaskQueue: "linux-amd64",
		YAML: `name: test-workflow
jobs:
  test:
    runs-on: linux-amd64
    steps:
      - name: Echo test
        uses: run@v1
        with:
          command: echo "hello"
`,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// WithWorkflowName sets the workflow name
func WithWorkflowName(name string) WorkflowInputOption {
	return func(w *WorkflowInputs) { w.Name = name }
}

// WithWorkflowID sets the workflow ID
func WithWorkflowID(id string) WorkflowInputOption {
	return func(w *WorkflowInputs) { w.ID = id }
}

// WithTaskQueue sets the task queue
func WithTaskQueue(queue string) WorkflowInputOption {
	return func(w *WorkflowInputs) { w.TaskQueue = queue }
}

// WithYAML sets the workflow YAML content
func WithYAML(yaml string) WorkflowInputOption {
	return func(w *WorkflowInputs) { w.YAML = yaml }
}

// ========================================
// HTTP Request Test Data Factory
// ========================================

// HTTPInputs represents inputs for HTTP request tests
type HTTPInputs struct {
	URL       string
	Method    string
	Headers   map[string]string
	Body      interface{}
	Timeout   int
	VerifySSL bool
}

// HTTPInputOption is a functional option for HTTPInputs
type HTTPInputOption func(*HTTPInputs)

// NewHTTPInputs creates HTTP inputs with defaults and options
func NewHTTPInputs(opts ...HTTPInputOption) *HTTPInputs {
	h := &HTTPInputs{
		URL:       "https://api.example.com",
		Method:    "GET",
		Headers:   make(map[string]string),
		Timeout:   30,
		VerifySSL: true,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// WithURL sets the request URL
func WithURL(url string) HTTPInputOption {
	return func(h *HTTPInputs) { h.URL = url }
}

// WithMethod sets the HTTP method
func WithMethod(method string) HTTPInputOption {
	return func(h *HTTPInputs) { h.Method = method }
}

// WithHeader adds a header
func WithHeader(key, value string) HTTPInputOption {
	return func(h *HTTPInputs) { h.Headers[key] = value }
}

// WithHeaders sets all headers
func WithHeaders(headers map[string]string) HTTPInputOption {
	return func(h *HTTPInputs) { h.Headers = headers }
}

// WithBody sets the request body
func WithBody(body interface{}) HTTPInputOption {
	return func(h *HTTPInputs) { h.Body = body }
}

// WithTimeout sets the timeout in seconds
func WithTimeout(timeout int) HTTPInputOption {
	return func(h *HTTPInputs) { h.Timeout = timeout }
}

// WithVerifySSL sets SSL verification
func WithVerifySSL(verify bool) HTTPInputOption {
	return func(h *HTTPInputs) { h.VerifySSL = verify }
}

// ToMap converts HTTPInputs to map[string]interface{} for plugin execution
func (h *HTTPInputs) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"url":        h.URL,
		"method":     h.Method,
		"timeout":    h.Timeout,
		"verify_ssl": h.VerifySSL,
	}
	if len(h.Headers) > 0 {
		headers := make(map[string]interface{})
		for k, v := range h.Headers {
			headers[k] = v
		}
		m["headers"] = headers
	}
	if h.Body != nil {
		m["body"] = h.Body
	}
	return m
}

// ========================================
// Shell Command Test Data Factory
// ========================================

// ShellInputs represents inputs for shell command tests
type ShellInputs struct {
	Command string
	Args    []string
	Workdir string
	Env     map[string]string
	Timeout int
}

// ShellInputOption is a functional option for ShellInputs
type ShellInputOption func(*ShellInputs)

// NewShellInputs creates shell inputs with defaults and options
func NewShellInputs(opts ...ShellInputOption) *ShellInputs {
	s := &ShellInputs{
		Command: "echo hello",
		Args:    []string{},
		Workdir: "/tmp",
		Env:     make(map[string]string),
		Timeout: 60,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// WithCommand sets the command
func WithCommand(cmd string) ShellInputOption {
	return func(s *ShellInputs) { s.Command = cmd }
}

// WithArgs sets the arguments
func WithArgs(args ...string) ShellInputOption {
	return func(s *ShellInputs) { s.Args = args }
}

// WithWorkdir sets the working directory
func WithWorkdir(dir string) ShellInputOption {
	return func(s *ShellInputs) { s.Workdir = dir }
}

// WithEnv adds an environment variable
func WithEnv(key, value string) ShellInputOption {
	return func(s *ShellInputs) { s.Env[key] = value }
}

// WithShellTimeout sets the timeout
func WithShellTimeout(timeout int) ShellInputOption {
	return func(s *ShellInputs) { s.Timeout = timeout }
}

// ToMap converts ShellInputs to map[string]interface{} for plugin execution
func (s *ShellInputs) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"command": s.Command,
		"workdir": s.Workdir,
		"timeout": s.Timeout,
	}
	if len(s.Args) > 0 {
		args := make([]interface{}, len(s.Args))
		for i, a := range s.Args {
			args[i] = a
		}
		m["args"] = args
	}
	if len(s.Env) > 0 {
		env := make(map[string]interface{})
		for k, v := range s.Env {
			env[k] = v
		}
		m["env"] = env
	}
	return m
}

// ========================================
// Docker Compose Test Data Factory
// ========================================

// DockerComposeInputs represents inputs for docker compose tests
type DockerComposeInputs struct {
	Action      string
	File        string
	ProjectName string
	Detach      bool
	Build       bool
	Volumes     bool
	RemoveImage string
}

// DockerComposeInputOption is a functional option for DockerComposeInputs
type DockerComposeInputOption func(*DockerComposeInputs)

// NewDockerComposeInputs creates docker compose inputs with defaults
func NewDockerComposeInputs(opts ...DockerComposeInputOption) *DockerComposeInputs {
	d := &DockerComposeInputs{
		Action:      "up",
		File:        "docker-compose.yml",
		ProjectName: fmt.Sprintf("test-%d", rand.Intn(10000)),
		Detach:      true,
		Build:       false,
		Volumes:     false,
		RemoveImage: "",
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// WithAction sets the compose action (up/down)
func WithAction(action string) DockerComposeInputOption {
	return func(d *DockerComposeInputs) { d.Action = action }
}

// WithComposeFile sets the compose file
func WithComposeFile(file string) DockerComposeInputOption {
	return func(d *DockerComposeInputs) { d.File = file }
}

// WithProjectName sets the project name
func WithProjectName(name string) DockerComposeInputOption {
	return func(d *DockerComposeInputs) { d.ProjectName = name }
}

// WithDetach sets detach mode
func WithDetach(detach bool) DockerComposeInputOption {
	return func(d *DockerComposeInputs) { d.Detach = detach }
}

// WithBuild enables building images
func WithBuild(build bool) DockerComposeInputOption {
	return func(d *DockerComposeInputs) { d.Build = build }
}

// ToMap converts DockerComposeInputs to map[string]interface{}
func (d *DockerComposeInputs) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"action":       d.Action,
		"file":         d.File,
		"project_name": d.ProjectName,
		"detach":       d.Detach,
		"build":        d.Build,
		"volumes":      d.Volumes,
		"rmi":          d.RemoveImage,
	}
}

// ========================================
// Event Test Data Factory
// ========================================

// WorkflowEventInputs represents inputs for workflow event tests
type WorkflowEventInputs struct {
	EventType    string
	WorkflowID   string
	WorkflowName string
	Timestamp    time.Time
}

// WorkflowEventOption is a functional option for WorkflowEventInputs
type WorkflowEventOption func(*WorkflowEventInputs)

// NewWorkflowEventInputs creates workflow event inputs with defaults
func NewWorkflowEventInputs(opts ...WorkflowEventOption) *WorkflowEventInputs {
	e := &WorkflowEventInputs{
		EventType:    "workflow.started",
		WorkflowID:   fmt.Sprintf("wf-event-%d", rand.Intn(10000)),
		WorkflowName: "test-workflow",
		Timestamp:    time.Now(),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// WithEventType sets the event type
func WithEventType(eventType string) WorkflowEventOption {
	return func(e *WorkflowEventInputs) { e.EventType = eventType }
}

// WithEventWorkflowID sets the workflow ID
func WithEventWorkflowID(id string) WorkflowEventOption {
	return func(e *WorkflowEventInputs) { e.WorkflowID = id }
}

// WithEventWorkflowName sets the workflow name
func WithEventWorkflowName(name string) WorkflowEventOption {
	return func(e *WorkflowEventInputs) { e.WorkflowName = name }
}

// WithTimestamp sets the timestamp
func WithTimestamp(ts time.Time) WorkflowEventOption {
	return func(e *WorkflowEventInputs) { e.Timestamp = ts }
}

// ========================================
// Random Data Generators
// ========================================

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitBytes  = "0123456789"
)

// RandomString generates a random string of given length
func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// RandomEmail generates a random email address
func RandomEmail() string {
	return fmt.Sprintf("%s@example.com", RandomString(8))
}

// RandomInt generates a random integer between min and max
func RandomInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

// RandomPort generates a random port number
func RandomPort() int {
	return RandomInt(10000, 60000)
}

// RandomUUID generates a UUID-like string (not a true UUID)
func RandomUUID() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		RandomString(8),
		RandomString(4),
		RandomString(4),
		RandomString(4),
		RandomString(12),
	)
}
