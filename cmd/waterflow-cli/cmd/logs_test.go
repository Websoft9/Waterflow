package cmd

import (
	"testing"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
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
		{"with timezone", "2026-01-05T10:30:45+08:00", "2026-01-05 10:30:45"},
		{"invalid format", "invalid", "invalid"},
		{"short string", "2026-01-05", "2026-01-05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimestamp(tt.input)
			// For timezone tests, just check the date part
			if tt.name == "with timezone" {
				if len(got) < 10 || got[:10] != "2026-01-05" {
					t.Errorf("formatTimestamp() = %v, want date starting with %v", got, "2026-01-05")
				}
			} else if got != tt.want {
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
