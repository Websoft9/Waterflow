// Package main implements a sleep/delay node for workflow control
package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/websoft9/waterflow/pkg/dsl/node"
)

// SleepNode implements a delay/sleep functionality for workflow control
type SleepNode struct{}

// Name returns the node identifier
func (n *SleepNode) Name() string {
	return "flow/sleep"
}

// Version returns the node version
func (n *SleepNode) Version() string {
	return "v1"
}

// Params returns the parameter specifications
func (n *SleepNode) Params() map[string]node.ParamSpec {
	return n.Metadata().InputSchema
}

// Metadata returns the node metadata including input/output schemas
func (n *SleepNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: "Sleep for a specified duration (supports seconds, minutes, hours)",
		Category:    "flow",
		InputSchema: map[string]node.ParamSpec{
			"duration": {
				Type:        "string",
				Required:    true,
				Description: "Duration to sleep (e.g., '30s', '5m', '1h', '1h30m', or '60' for seconds)",
				Pattern:     `^\d+[smh]?$|^\d+h\d+m$|^\d+m\d+s$|^\d+h\d+m\d+s$`,
			},
		},
		OutputSchema: map[string]interface{}{
			"duration_seconds": "int - actual sleep duration in seconds",
			"completed":        "bool - true if sleep completed, false if cancelled",
			"elapsed_ms":       "int - actual elapsed time in milliseconds",
		},
	}
}

// Execute performs the sleep operation
func (n *SleepNode) Execute(
	ctx context.Context,
	inputs map[string]interface{},
) (*node.NodeResult, error) {
	startTime := time.Now()

	// 1. Get duration parameter
	durationStr, ok := inputs["duration"].(string)
	if !ok || durationStr == "" {
		return nil, fmt.Errorf("duration is required")
	}

	// 2. Parse duration
	duration, err := parseDuration(durationStr)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}

	// 3. Validate range
	if duration < time.Second {
		return nil, fmt.Errorf("duration must be at least 1 second, got %s", duration)
	}
	if duration > 24*time.Hour {
		return nil, fmt.Errorf("duration must not exceed 24 hours, got %s", duration)
	}

	// 4. Create timer
	timer := time.NewTimer(duration)
	defer timer.Stop() // Ensure timer is stopped to prevent resource leak

	// 5. Wait for completion or cancellation using select
	// This allows the sleep to be interrupted via context cancellation
	completed := false
	select {
	case <-timer.C:
		// Timer expired: sleep completed normally
		completed = true
	case <-ctx.Done():
		// Context cancelled: sleep interrupted, return immediately
		completed = false
	}

	elapsed := time.Since(startTime)

	// 6. Return result
	logs := []string{
		fmt.Sprintf("Sleep started for %s", duration),
	}
	if completed {
		logs = append(logs, fmt.Sprintf("Sleep completed after %dms", elapsed.Milliseconds()))
	} else {
		logs = append(logs, fmt.Sprintf("Sleep cancelled after %dms", elapsed.Milliseconds()))
	}

	return &node.NodeResult{
		Outputs: map[string]interface{}{
			"duration_seconds": int(duration.Seconds()),
			"completed":        completed,
			"elapsed_ms":       elapsed.Milliseconds(),
		},
		Logs:     logs,
		Duration: elapsed,
	}, nil
}

// parseDuration parses a duration string into time.Duration
// Supported formats:
// - "30s", "5m", "1h" - Go standard duration format
// - "1h30m", "30m15s", "1h30m45s" - Combined formats
// - "30", "60" - Pure numbers default to seconds
func parseDuration(s string) (time.Duration, error) {
	// 1. Try Go standard library first (supports all formats like "1h30m45s")
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// 2. Pure numbers default to seconds
	if matched, _ := regexp.MatchString(`^\d+$`, s); matched {
		seconds, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", s)
		}
		return time.Duration(seconds) * time.Second, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s (examples: '30s', '5m', '1h', '1h30m', or '60')", s)
}

// Register is the exported function for Go plugin system
func Register() node.Node {
	return &SleepNode{}
}
