// Package perfutil provides performance testing utilities for Waterflow.
package perfutil

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"go.temporal.io/sdk/client"
)

// CrashSimulator helps simulate and verify crash recovery scenarios.
type CrashSimulator struct {
	t          *testing.T
	cmd        *exec.Cmd
	workflowID string
}

// NewCrashSimulator creates a new crash simulator for testing server recovery.
func NewCrashSimulator(t *testing.T, cmd *exec.Cmd) *CrashSimulator {
	t.Helper()
	return &CrashSimulator{
		t:   t,
		cmd: cmd,
	}
}

// WaitForStartup waits for the process to start and become ready.
func (cs *CrashSimulator) WaitForStartup(duration time.Duration) {
	cs.t.Helper()
	cs.t.Logf("Waiting %v for server startup...", duration)
	time.Sleep(duration)
}

// SimulateCrash simulates a process crash by sending SIGKILL.
// Returns after the process has been killed and reaped.
func (cs *CrashSimulator) SimulateCrash() error {
	cs.t.Helper()

	if cs.cmd == nil || cs.cmd.Process == nil {
		cs.t.Fatal("No process to crash")
	}

	pid := cs.cmd.Process.Pid
	cs.t.Logf("Simulating crash: killing process %d", pid)

	if err := cs.cmd.Process.Kill(); err != nil {
		return err
	}

	// Reap zombie process
	cs.cmd.Wait()

	cs.t.Logf("Process %d killed successfully", pid)
	return nil
}

// VerifyEventHistory verifies that Event History is preserved after crash.
// Returns the number of events found.
func VerifyEventHistory(t *testing.T, temporalClient client.Client, workflowID string) int {
	t.Helper()

	ctx := context.Background()
	iter := temporalClient.GetWorkflowHistory(ctx, workflowID, "", false, 0)

	eventCount := 0
	for iter.HasNext() {
		if _, err := iter.Next(); err != nil {
			t.Fatalf("Failed to get event: %v", err)
		}
		eventCount++
	}

	t.Logf("Event History: %d events found for workflow %s", eventCount, workflowID)
	return eventCount
}

// AssertStateRecovery verifies that workflow state is fully recovered after crash.
// PRD#L235: 崩溃测试中 100% 状态恢复
func AssertStateRecovery(t *testing.T, temporalClient client.Client, workflowID string, expectedEvents int) {
	t.Helper()

	actualEvents := VerifyEventHistory(t, temporalClient, workflowID)

	if actualEvents < expectedEvents {
		t.Errorf("PRD#L235 violation: Event loss detected\n"+
			"  Expected: %d events\n"+
			"  Actual:   %d events\n"+
			"  Loss:     %d events",
			expectedEvents, actualEvents, expectedEvents-actualEvents)
	} else {
		t.Logf("State recovery: 100%% (%d/%d events) ✓", actualEvents, expectedEvents)
	}
}

// AssertWorkflowRecoverable verifies workflow can continue after recovery.
func AssertWorkflowRecoverable(t *testing.T, temporalClient client.Client, workflowID string) {
	t.Helper()

	ctx := context.Background()
	desc, err := temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		t.Fatalf("Failed to describe workflow: %v", err)
	}

	status := desc.WorkflowExecutionInfo.Status
	t.Logf("Workflow status after recovery: %s", status)

	// Workflow should be in a valid state (not unknown or failed due to crash)
	validStates := []string{"Running", "Completed", "Failed", "Terminated"}
	valid := false
	for _, s := range validStates {
		if string(status) == s {
			valid = true
			break
		}
	}

	if !valid {
		t.Errorf("Workflow in invalid state after recovery: %s", status)
	}
}
