package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    string
	}{
		{
			name:    "zero seconds",
			seconds: 0,
			want:    "0s",
		},
		{
			name:    "less than 1 minute",
			seconds: 45,
			want:    "45s",
		},
		{
			name:    "exactly 1 minute",
			seconds: 60,
			want:    "1m 0s",
		},
		{
			name:    "minutes and seconds",
			seconds: 135,
			want:    "2m 15s",
		},
		{
			name:    "exactly 1 hour",
			seconds: 3600,
			want:    "1h 0m",
		},
		{
			name:    "hours and minutes",
			seconds: 3665,
			want:    "1h 1m",
		},
		{
			name:    "multiple hours",
			seconds: 7325,
			want:    "2h 2m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.seconds)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateDuration(t *testing.T) {
	tests := []struct {
		name     string
		startStr string
		endStr   string
		want     int
	}{
		{
			name:     "valid duration",
			startStr: "2026-01-05T10:00:00Z",
			endStr:   "2026-01-05T10:05:30Z",
			want:     330,
		},
		{
			name:     "zero duration",
			startStr: "2026-01-05T10:00:00Z",
			endStr:   "2026-01-05T10:00:00Z",
			want:     0,
		},
		{
			name:     "invalid start time",
			startStr: "invalid",
			endStr:   "2026-01-05T10:05:30Z",
			want:     0,
		},
		{
			name:     "invalid end time",
			startStr: "2026-01-05T10:00:00Z",
			endStr:   "invalid",
			want:     0,
		},
		{
			name:     "both invalid",
			startStr: "invalid",
			endStr:   "also-invalid",
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDuration(tt.startStr, tt.endStr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateDurationSince(t *testing.T) {
	tests := []struct {
		name     string
		startStr string
		wantZero bool // 期望返回0（错误情况）
	}{
		{
			name:     "invalid time format",
			startStr: "invalid",
			wantZero: true,
		},
		{
			name:     "empty string",
			startStr: "",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDurationSince(tt.startStr)
			if tt.wantZero {
				assert.Equal(t, 0, got)
			}
		})
	}
}

func TestTruncateName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short name - no truncation",
			input:  "build",
			maxLen: 20,
			want:   "build",
		},
		{
			name:   "exact length - no truncation",
			input:  "build-and-deploy",
			maxLen: 16,
			want:   "build-and-deploy",
		},
		{
			name:   "long name - truncated with ellipsis",
			input:  "build-and-deploy-production",
			maxLen: 20,
			want:   "build-and-deploy-...",
		},
		{
			name:   "very long name",
			input:  "very-long-job-name-that-exceeds-limit",
			maxLen: 15,
			want:   "very-long-jo...",
		},
		{
			name:   "maxLen too short for ellipsis",
			input:  "longname",
			maxLen: 3,
			want:   "lon",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateName(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, len(got), tt.maxLen)
		})
	}
}

func TestGetStatusSymbol(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		noColor bool
		want    string
	}{
		{
			name:    "completed with color",
			status:  "completed",
			noColor: false,
			want:    "✓", // 包含 ANSI 颜色码，但我们只检查符号
		},
		{
			name:    "completed without color",
			status:  "completed",
			noColor: true,
			want:    "[✓]",
		},
		{
			name:    "running with color",
			status:  "running",
			noColor: false,
			want:    "→",
		},
		{
			name:    "running without color",
			status:  "running",
			noColor: true,
			want:    "[→]",
		},
		{
			name:    "failed with color",
			status:  "failed",
			noColor: false,
			want:    "✗",
		},
		{
			name:    "failed without color",
			status:  "failed",
			noColor: true,
			want:    "[✗]",
		},
		{
			name:    "cancelled without color",
			status:  "cancelled",
			noColor: true,
			want:    "[⊗]",
		},
		{
			name:    "timeout without color",
			status:  "timeout",
			noColor: true,
			want:    "[⊗]",
		},
		{
			name:    "pending without color",
			status:  "pending",
			noColor: true,
			want:    "[○]",
		},
		{
			name:    "unknown status without color",
			status:  "unknown",
			noColor: true,
			want:    "[○]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{noColor: tt.noColor}
			got := f.getStatusSymbol(tt.status)
			assert.Contains(t, got, tt.want)
		})
	}
}
