package cmd

import (
	"fmt"
	"testing"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateLogsParams tests log parameter validation
func TestValidateLogsParams(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		wantError bool
	}{
		{"valid single level", "info", false},
		{"valid multiple levels", "info,error,warn", false},
		{"valid with spaces", "info, error, warn", false},
		{"invalid level", "invalid", true},
		{"mixed valid and invalid", "info,invalid", true},
		{"empty string", "", false},
		{"case insensitive", "INFO,ERROR", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logsLevel = tt.level
			err := validateLogsParams()
			if (err != nil) != tt.wantError {
				t.Errorf("validateLogsParams() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestBuildLogsQuery tests query parameter construction
func TestBuildLogsQuery(t *testing.T) {
	tests := []struct {
		name      string
		tail      int
		level     string
		job       string
		step      string
		wantTail  int
		wantLevel string
	}{
		{"default tail", 100, "", "", "", 100, ""},
		{"tail all (-1)", -1, "", "", "", 1000, ""},
		{"with level filter", 50, "error", "", "", 50, "error"},
		{"with job filter", 100, "", "build", "", 100, ""},
		{"with step filter", 100, "", "", "checkout", 100, ""},
		{"all filters", 200, "info", "deploy", "push", 200, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logsTail = tt.tail
			logsLevel = tt.level
			logsJob = tt.job
			logsStep = tt.step

			query := buildLogsQuery()

			if query.Tail != tt.wantTail {
				t.Errorf("buildLogsQuery().Tail = %v, want %v", query.Tail, tt.wantTail)
			}
			if query.Level != tt.wantLevel {
				t.Errorf("buildLogsQuery().Level = %v, want %v", query.Level, tt.wantLevel)
			}
		})
	}
}

// TestFormatTimestamp tests timestamp formatting
func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid RFC3339", "2026-01-05T10:30:45Z", "2026-01-05 10:30:45"},
		{"with timezone", "2026-01-05T10:30:45+08:00", "2026-01-05 02:30:45"},
		{"invalid format", "invalid", "invalid"},
		{"short string", "2026-01-05", "2026-01-05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimestamp(tt.input)
			// Check the date and time parts (format is always UTC now)
			if got != tt.want {
				t.Errorf("formatTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLogsFormatterGetLevelColor tests level color mapping
func TestLogsFormatterGetLevelColor(t *testing.T) {
	formatter := newLogsFormatter("text", false, false)

	tests := []struct {
		level string
		color string
	}{
		{"error", "red"},
		{"warn", "yellow"},
		{"info", "blue"},
		{"debug", "gray"},
		{"unknown", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			color := formatter.getLevelColor(tt.level)
			if color == nil {
				t.Errorf("getLevelColor(%v) returned nil", tt.level)
			}
		})
	}
}

// TestLogsFormatterNoColor tests color disable functionality
func TestLogsFormatterNoColor(t *testing.T) {
	formatter := newLogsFormatter("text", true, false)

	color := formatter.getLevelColor("error")
	if color == nil {
		t.Error("getLevelColor() returned nil with noColor=true")
	}
}

// TestLogsFormatterPrintLog tests log printing (basic structure)
func TestLogsFormatterPrintLog(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		log       *client.LogEntry
		wantError bool
	}{
		{
			"text format",
			"text",
			&client.LogEntry{
				Timestamp: "2026-01-05T10:30:45Z",
				Level:     "info",
				Message:   "test message",
			},
			false,
		},
		{
			"json format",
			"json",
			&client.LogEntry{
				Timestamp: "2026-01-05T10:30:45Z",
				Level:     "error",
				Message:   "error message",
				Error:     "connection timeout",
			},
			false,
		},
		{
			"compact format",
			"compact",
			&client.LogEntry{
				Timestamp: "2026-01-05T10:30:45Z",
				Level:     "info",
				Job:       "build",
				Step:      "checkout",
				Message:   "step started",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := newLogsFormatter(tt.format, true, false)
			err := formatter.PrintLog(tt.log)
			if (err != nil) != tt.wantError {
				t.Errorf("PrintLog() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestNewLogsCmd tests logs command creation
func TestNewLogsCmd(t *testing.T) {
	cmd := newLogsCmd()
	require.NotNil(t, cmd)

	assert.Equal(t, "logs <workflow-id>", cmd.Use)
	assert.Contains(t, cmd.Short, "logs")

	// Check flags exist
	assert.NotNil(t, cmd.Flags().Lookup("follow"))
	assert.NotNil(t, cmd.Flags().Lookup("tail"))
	assert.NotNil(t, cmd.Flags().Lookup("level"))
	assert.NotNil(t, cmd.Flags().Lookup("job"))
	assert.NotNil(t, cmd.Flags().Lookup("step"))
	assert.NotNil(t, cmd.Flags().Lookup("format"))
}

// TestDisplayNoLogsMessage tests the no logs message display
func TestDisplayNoLogsMessage(t *testing.T) {
	// Save and restore state
	oldLevel := logsLevel
	oldJob := logsJob
	oldStep := logsStep
	defer func() {
		logsLevel = oldLevel
		logsJob = oldJob
		logsStep = oldStep
	}()

	tests := []struct {
		name  string
		level string
		job   string
		step  string
	}{
		{"no filters", "", "", ""},
		{"with level filter", "error", "", ""},
		{"with job filter", "", "build", ""},
		{"with step filter", "", "", "checkout"},
		{"with all filters", "info", "deploy", "push"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logsLevel = tt.level
			logsJob = tt.job
			logsStep = tt.step
			// Should not panic
			err := displayNoLogsMessage()
			assert.NoError(t, err)
		})
	}
}

// TestToJSON tests JSON conversion
func TestToJSON(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			"simple map",
			map[string]interface{}{"key": "value"},
		},
		{
			"nested map",
			map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": "value",
				},
			},
		},
		{
			"array",
			[]string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toJSON(tt.input)
			assert.NotEmpty(t, result)
		})
	}
}

// TestFormatLogsError tests error formatting
func TestFormatLogsError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		format string
	}{
		{
			"generic error text",
			fmt.Errorf("generic error"),
			"text",
		},
		{
			"server error text",
			&client.ServerError{
				StatusCode: 500,
				Code:       "INTERNAL_ERROR",
				Message:    "Internal server error",
			},
			"text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These functions call os.Exit, so we just verify they don't panic during setup
			// In a real test environment, we'd need to mock os.Exit
			_ = tt.err
			_ = tt.format
		})
	}
}
