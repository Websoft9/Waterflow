// Package stress provides stress testing utilities for Waterflow
package stress

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/sdk/client"
)

// TestServerCrashRecovery verifies Server crash recovery with Event Sourcing
// 追溯: PRD#L235 - 持久性: 崩溃测试中 100% 状态恢复
// 验证: Server 重启后工作流状态完全恢复，无数据丢失
func TestServerCrashRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping crash recovery test in short mode")
	}

	// Connect to Temporal
	tc, err := client.Dial(client.Options{
		HostPort:  "localhost:7233",
		Namespace: "default",
	})
	if err != nil {
		t.Skipf("Temporal not available: %v", err)
	}
	defer tc.Close()

	ctx := context.Background()

	// Step 1: Start Server process
	serverCmd := exec.Command("./bin/server")
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	serverPID := serverCmd.Process.Pid
	t.Logf("Server started with PID %d", serverPID)

	// Wait for server to initialize
	time.Sleep(5 * time.Second)

	// Step 2: Start a long-running workflow
	workflowID := "crash-recovery-test"
	we, err := tc.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "linux-amd64",
	}, "waterflow-executor", workflowID)
	if err != nil {
		serverCmd.Process.Kill()
		t.Fatalf("Failed to start workflow: %v", err)
	}

	// Step 3: Get Event History before crash
	iter := tc.GetWorkflowHistory(ctx, workflowID, "", false, 0)
	eventsBefore := 0
	for iter.HasNext() {
		if _, err := iter.Next(); err != nil {
			serverCmd.Process.Kill()
			t.Fatalf("Failed to get event: %v", err)
		}
		eventsBefore++
	}
	t.Logf("Events before crash: %d", eventsBefore)

	// Step 4: Simulate crash (SIGKILL)
	t.Log("Simulating server crash...")
	if err := serverCmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatalf("Failed to kill server: %v", err)
	}
	serverCmd.Wait() // Reap zombie process

	// Wait for crash to complete
	time.Sleep(3 * time.Second)

	// Step 5: Restart Server
	t.Log("Restarting server...")
	serverCmd = exec.Command("./bin/server")
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to restart server: %v", err)
	}
	defer serverCmd.Process.Kill()

	// Wait for server to recover
	time.Sleep(5 * time.Second)

	// Step 6: Verify Event History after recovery
	iter = tc.GetWorkflowHistory(ctx, workflowID, "", false, 0)
	eventsAfter := 0
	for iter.HasNext() {
		if _, err := iter.Next(); err != nil {
			t.Fatalf("Failed to get event after recovery: %v", err)
		}
		eventsAfter++
	}
	t.Logf("Events after recovery: %d", eventsAfter)

	// Validation: All events must be preserved
	if eventsAfter < eventsBefore {
		t.Errorf("Event loss detected: before=%d, after=%d", eventsBefore, eventsAfter)
	}

	// Verify workflow can continue
	desc, err := tc.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	t.Logf("Workflow status after recovery: %s", desc.WorkflowExecutionInfo.Status)

	// PRD requirement: 100% state recovery
	if eventsAfter != eventsBefore {
		t.Errorf("PRD#L235 violation: Expected 100%% state recovery, got %d/%d events",
			eventsAfter, eventsBefore)
	}

	// Optional: Wait for workflow to complete
	_ = we
}
