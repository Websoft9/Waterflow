package sdk_test

import (
	"context"
	"testing"
	"time"

	"github.com/Websoft9/waterflow/pkg/sdk"
)

func TestNewClient(t *testing.T) {
	// Test with valid config
	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Test with API key
	client, err = sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
		APIKey:    "test-key",
	})
	if err != nil {
		t.Fatalf("NewClient() with API key failed: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Test with custom timeout
	client, err = sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
		Timeout:   60 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient() with timeout failed: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}
}

func TestNewClient_Errors(t *testing.T) {
	// Test with nil config
	_, err := sdk.NewClient(nil)
	if err == nil {
		t.Error("NewClient() with nil config should fail")
	}

	// Test with empty server URL
	_, err = sdk.NewClient(&sdk.ClientConfig{})
	if err == nil {
		t.Error("NewClient() with empty server URL should fail")
	}
}

func TestNewDefaultClient(t *testing.T) {
	// This test requires WATERFLOW_SERVER_URL environment variable
	// or will use default http://localhost:8080
	client, err := sdk.NewDefaultClient()
	if err != nil {
		t.Fatalf("NewDefaultClient() failed: %v", err)
	}
	if client == nil {
		t.Fatal("NewDefaultClient() returned nil client")
	}
}

func TestSubmitWorkflow_ValidationError(t *testing.T) {
	client, _ := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
	})

	ctx := context.Background()

	// Test with nil request
	_, err := client.SubmitWorkflow(ctx, nil)
	if err == nil {
		t.Error("SubmitWorkflow() with nil request should fail")
	}

	// Test with empty YAML
	_, err = client.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{})
	if err == nil {
		t.Error("SubmitWorkflow() with empty YAML should fail")
	}
}

func TestGetWorkflowStatus_ValidationError(t *testing.T) {
	client, _ := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
	})

	ctx := context.Background()

	// Test with empty workflow ID
	_, err := client.GetWorkflowStatus(ctx, "")
	if err == nil {
		t.Error("GetWorkflowStatus() with empty ID should fail")
	}
}

func TestIsNotFound(t *testing.T) {
	err := &sdk.ServerError{StatusCode: 404}
	if !sdk.IsNotFound(err) {
		t.Error("IsNotFound() should return true for 404 error")
	}

	err = &sdk.ServerError{StatusCode: 500}
	if sdk.IsNotFound(err) {
		t.Error("IsNotFound() should return false for 500 error")
	}
}

func TestIsValidationError(t *testing.T) {
	err := &sdk.ServerError{StatusCode: 400}
	if !sdk.IsValidationError(err) {
		t.Error("IsValidationError() should return true for 400 error")
	}

	err = &sdk.ServerError{StatusCode: 422}
	if !sdk.IsValidationError(err) {
		t.Error("IsValidationError() should return true for 422 error")
	}

	err = &sdk.ServerError{StatusCode: 500}
	if sdk.IsValidationError(err) {
		t.Error("IsValidationError() should return false for 500 error")
	}
}

func TestGetWorkflowLogs_ValidationError(t *testing.T) {
	client, _ := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
	})

	ctx := context.Background()

	// Test with nil request
	_, err := client.GetWorkflowLogs(ctx, nil)
	if err == nil {
		t.Error("GetWorkflowLogs() with nil request should fail")
	}

	// Test with empty workflow ID
	_, err = client.GetWorkflowLogs(ctx, &sdk.GetLogsRequest{})
	if err == nil {
		t.Error("GetWorkflowLogs() with empty workflow ID should fail")
	}
}

func TestListWorkflows(t *testing.T) {
	client, _ := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
	})

	ctx := context.Background()

	// Test with nil request (should use defaults)
	_, err := client.ListWorkflows(ctx, nil)
	// Error is expected if server is not running, but method should not panic
	_ = err
}

func TestRerunWorkflow_ValidationError(t *testing.T) {
	client, _ := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
	})

	ctx := context.Background()

	// Test with nil request
	_, err := client.RerunWorkflow(ctx, nil)
	if err == nil {
		t.Error("RerunWorkflow() with nil request should fail")
	}

	// Test with empty workflow ID
	_, err = client.RerunWorkflow(ctx, &sdk.RerunWorkflowRequest{})
	if err == nil {
		t.Error("RerunWorkflow() with empty workflow ID should fail")
	}
}
