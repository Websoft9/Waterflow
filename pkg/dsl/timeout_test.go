package dsl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeoutResolver_ResolveStepTimeout(t *testing.T) {
	resolver := NewTimeoutResolver()

	tests := []struct {
		name             string
		step             *Step
		job              *Job
		expectedDuration time.Duration
	}{
		{
			name: "Step explicit timeout",
			step: &Step{
				TimeoutMinutes: 10,
			},
			job: &Job{
				TimeoutMinutes: 30,
			},
			expectedDuration: 10 * time.Minute,
		},
		{
			name: "Inherit from Job timeout",
			step: &Step{
				TimeoutMinutes: 0, // 未配置
			},
			job: &Job{
				TimeoutMinutes: 60,
			},
			expectedDuration: 60 * time.Minute,
		},
		{
			name: "Use default timeout",
			step: &Step{
				TimeoutMinutes: 0,
			},
			job: &Job{
				TimeoutMinutes: 0,
			},
			expectedDuration: 360 * time.Minute, // 默认 6 小时
		},
		{
			name: "Step timeout overrides Job",
			step: &Step{
				TimeoutMinutes: 5,
			},
			job: &Job{
				TimeoutMinutes: 120,
			},
			expectedDuration: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.ResolveStepTimeout(tt.step, tt.job)
			assert.Equal(t, tt.expectedDuration, result)
		})
	}
}

func TestTimeoutResolver_ResolveJobTimeout(t *testing.T) {
	resolver := NewTimeoutResolver()

	tests := []struct {
		name             string
		job              *Job
		expectedDuration time.Duration
	}{
		{
			name: "Job with explicit timeout",
			job: &Job{
				TimeoutMinutes: 120,
			},
			expectedDuration: 120 * time.Minute,
		},
		{
			name: "Job with default timeout",
			job: &Job{
				TimeoutMinutes: 0,
			},
			expectedDuration: 360 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.ResolveJobTimeout(tt.job)
			assert.Equal(t, tt.expectedDuration, result)
		})
	}
}

func TestTimeoutResolver_ValidateTimeout(t *testing.T) {
	resolver := NewTimeoutResolver()

	tests := []struct {
		name           string
		timeoutMinutes int
		fieldName      string
		expectError    bool
	}{
		{
			name:           "Valid timeout",
			timeoutMinutes: 60,
			fieldName:      "step.timeout",
			expectError:    false,
		},
		{
			name:           "Negative timeout",
			timeoutMinutes: -1,
			fieldName:      "step.timeout",
			expectError:    true,
		},
		{
			name:           "Timeout exceeds 24 hours",
			timeoutMinutes: 1500,
			fieldName:      "job.timeout",
			expectError:    true,
		},
		{
			name:           "Zero timeout (valid)",
			timeoutMinutes: 0,
			fieldName:      "step.timeout",
			expectError:    false,
		},
		{
			name:           "Maximum valid timeout",
			timeoutMinutes: 1440,
			fieldName:      "job.timeout",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := resolver.ValidateTimeout(tt.timeoutMinutes, tt.fieldName)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
