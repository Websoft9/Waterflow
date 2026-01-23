package output

import (
	"testing"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
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

// TestGetStatusColor tests status color retrieval
func TestGetStatusColor(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		noColor bool
	}{
		{"completed color", "completed", false},
		{"completed no color", "completed", true},
		{"running color", "running", false},
		{"running no color", "running", true},
		{"failed color", "failed", false},
		{"failed no color", "failed", true},
		{"cancelled color", "cancelled", false},
		{"timeout color", "timeout", false},
		{"pending color", "pending", false},
		{"unknown color", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{noColor: tt.noColor}
			got := f.getStatusColor(tt.status)
			assert.NotNil(t, got)
		})
	}
}

// TestPrintLegend tests legend printing
func TestPrintLegend(t *testing.T) {
	tests := []struct {
		name    string
		noColor bool
	}{
		{"with color", false},
		{"without color", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{noColor: tt.noColor}
			// Should not panic
			f.printLegend()
		})
	}
}

// TestPrintWorkflowStatus tests workflow status printing
func TestPrintWorkflowStatus(t *testing.T) {
	status := &client.WorkflowStatus{
		ID:     "test-id-123",
		Name:   "test-workflow",
		Status: "completed",
	}

	tests := []struct {
		name    string
		format  Format
		compact bool
		wantErr bool
	}{
		{"text format", FormatText, false, false},
		{"text compact", FormatText, true, false},
		{"json format", FormatJSON, false, false},
		{"yaml format", FormatYAML, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{format: tt.format, noColor: true}
			err := f.PrintWorkflowStatus(status, tt.compact)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPrintStatusTextWithJobs tests status text with jobs
func TestPrintStatusTextWithJobs(t *testing.T) {
	status := &client.WorkflowStatus{
		ID:        "test-id",
		Name:      "test-workflow",
		Status:    "running",
		RunID:     "run-123",
		CreatedAt: "2024-01-01T00:00:00Z",
		StartedAt: "2024-01-01T00:01:00Z",
		Jobs: []client.JobStatus{
			{
				Name:        "build",
				Status:      "completed",
				StartedAt:   "2024-01-01T00:01:00Z",
				CompletedAt: "2024-01-01T00:02:00Z",
				Steps: []client.StepStatus{
					{Name: "checkout", Status: "completed"},
					{Name: "compile", Status: "completed"},
				},
			},
			{
				Name:      "deploy",
				Status:    "running",
				StartedAt: "2024-01-01T00:02:00Z",
				Steps: []client.StepStatus{
					{Name: "push", Status: "running"},
				},
			},
		},
	}

	f := &Formatter{format: "text", noColor: true}

	// Full mode
	err := f.PrintWorkflowStatus(status, false)
	assert.NoError(t, err)

	// Compact mode
	err = f.PrintWorkflowStatus(status, true)
	assert.NoError(t, err)
}

// TestPrintStatusTextWithError tests status text with error
func TestPrintStatusTextWithError(t *testing.T) {
	status := &client.WorkflowStatus{
		ID:     "test-id",
		Name:   "test-workflow",
		Status: "failed",
		Error:  "Something went wrong",
	}

	f := &Formatter{format: "text", noColor: true}
	err := f.PrintWorkflowStatus(status, false)
	assert.NoError(t, err)
}

// TestPrintJob tests job printing
func TestPrintJob(t *testing.T) {
	tests := []struct {
		name    string
		job     client.JobStatus
		compact bool
	}{
		{
			name: "completed job",
			job: client.JobStatus{
				Name:        "build",
				Status:      "completed",
				StartedAt:   "2024-01-01T00:00:00Z",
				CompletedAt: "2024-01-01T00:01:00Z",
			},
			compact: false,
		},
		{
			name: "running job",
			job: client.JobStatus{
				Name:      "deploy",
				Status:    "running",
				StartedAt: "2024-01-01T00:00:00Z",
			},
			compact: false,
		},
		{
			name: "compact mode",
			job: client.JobStatus{
				Name:   "test",
				Status: "pending",
			},
			compact: true,
		},
		{
			name: "job with steps",
			job: client.JobStatus{
				Name:   "build",
				Status: "completed",
				Steps: []client.StepStatus{
					{Name: "step1", Status: "completed"},
					{Name: "step2", Status: "completed"},
				},
			},
			compact: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{noColor: true}
			// Should not panic
			f.printJob(tt.job, tt.compact)
		})
	}
}

// TestPrintStep tests step printing
func TestPrintStep(t *testing.T) {
	tests := []struct {
		name string
		step client.StepStatus
	}{
		{
			name: "completed step",
			step: client.StepStatus{Name: "checkout", Status: "completed"},
		},
		{
			name: "running step",
			step: client.StepStatus{Name: "build", Status: "running"},
		},
		{
			name: "failed step",
			step: client.StepStatus{Name: "test", Status: "failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{noColor: true}
			// Should not panic
			f.printStep(tt.step)
		})
	}
}

// TestToJSON tests ToJSON function
func TestToJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		contains string
	}{
		{
			name:     "simple map",
			data:     map[string]string{"key": "value"},
			contains: "key",
		},
		{
			name:     "nested data",
			data:     map[string]interface{}{"outer": map[string]string{"inner": "val"}},
			contains: "inner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToJSON(tt.data)
			assert.Contains(t, result, tt.contains)
		})
	}
}
