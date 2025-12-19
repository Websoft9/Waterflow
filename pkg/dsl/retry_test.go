package dsl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryPolicyResolver_Resolve(t *testing.T) {
	resolver := NewRetryPolicyResolver()

	tests := []struct {
		name     string
		strategy *RetryStrategy
		expected *ResolvedRetryPolicy
		hasError bool
	}{
		{
			name:     "Nil strategy returns default",
			strategy: nil,
			expected: &ResolvedRetryPolicy{
				MaxAttempts:        3,
				InitialInterval:    1 * time.Second,
				BackoffCoefficient: 2.0,
				MaxInterval:        60 * time.Second,
			},
			hasError: false,
		},
		{
			name: "Custom strategy",
			strategy: &RetryStrategy{
				MaxAttempts:        5,
				InitialInterval:    "2s",
				BackoffCoefficient: 1.5,
				MaxInterval:        "30s",
			},
			expected: &ResolvedRetryPolicy{
				MaxAttempts:        5,
				InitialInterval:    2 * time.Second,
				BackoffCoefficient: 1.5,
				MaxInterval:        30 * time.Second,
			},
			hasError: false,
		},
		{
			name: "Partial custom strategy",
			strategy: &RetryStrategy{
				MaxAttempts: 10,
			},
			expected: &ResolvedRetryPolicy{
				MaxAttempts:        10,
				InitialInterval:    1 * time.Second,
				BackoffCoefficient: 2.0,
				MaxInterval:        60 * time.Second,
			},
			hasError: false,
		},
		{
			name: "Invalid initial interval",
			strategy: &RetryStrategy{
				InitialInterval: "invalid",
			},
			expected: nil,
			hasError: true,
		},
		{
			name: "Invalid max interval",
			strategy: &RetryStrategy{
				MaxInterval: "xyz",
			},
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(tt.strategy)

			if tt.hasError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.MaxAttempts, result.MaxAttempts)
				assert.Equal(t, tt.expected.InitialInterval, result.InitialInterval)
				assert.Equal(t, tt.expected.BackoffCoefficient, result.BackoffCoefficient)
				assert.Equal(t, tt.expected.MaxInterval, result.MaxInterval)
			}
		})
	}
}

func TestRetryPolicyResolver_DefaultRetryPolicy(t *testing.T) {
	resolver := NewRetryPolicyResolver()
	policy := resolver.DefaultRetryPolicy()

	assert.Equal(t, 3, policy.MaxAttempts)
	assert.Equal(t, 1*time.Second, policy.InitialInterval)
	assert.Equal(t, 2.0, policy.BackoffCoefficient)
	assert.Equal(t, 60*time.Second, policy.MaxInterval)
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    time.Duration
		expectError bool
	}{
		{
			name:        "Seconds",
			input:       "5s",
			expected:    5 * time.Second,
			expectError: false,
		},
		{
			name:        "Minutes",
			input:       "2m",
			expected:    2 * time.Minute,
			expectError: false,
		},
		{
			name:        "Hours",
			input:       "1h",
			expected:    1 * time.Hour,
			expectError: false,
		},
		{
			name:        "Milliseconds",
			input:       "500ms",
			expected:    500 * time.Millisecond,
			expectError: false,
		},
		{
			name:        "Invalid format",
			input:       "invalid",
			expected:    0,
			expectError: true,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDuration(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestResolvedRetryPolicy_CalculateNextRetryInterval(t *testing.T) {
	policy := &ResolvedRetryPolicy{
		MaxAttempts:        5,
		InitialInterval:    1 * time.Second,
		BackoffCoefficient: 2.0,
		MaxInterval:        60 * time.Second,
	}

	tests := []struct {
		name          string
		attemptNumber int
		expected      time.Duration
	}{
		{
			name:          "First retry (attempt 0)",
			attemptNumber: 0,
			expected:      1 * time.Second,
		},
		{
			name:          "Second retry (attempt 1)",
			attemptNumber: 1,
			expected:      2 * time.Second,
		},
		{
			name:          "Third retry (attempt 2)",
			attemptNumber: 2,
			expected:      4 * time.Second,
		},
		{
			name:          "Fourth retry (attempt 3)",
			attemptNumber: 3,
			expected:      8 * time.Second,
		},
		{
			name:          "Fifth retry (attempt 4)",
			attemptNumber: 4,
			expected:      16 * time.Second,
		},
		{
			name:          "Exceeds max interval",
			attemptNumber: 10,
			expected:      60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := policy.CalculateNextRetryInterval(tt.attemptNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolvedRetryPolicy_CalculateNextRetryInterval_CustomCoefficient(t *testing.T) {
	policy := &ResolvedRetryPolicy{
		MaxAttempts:        5,
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 1.5,
		MaxInterval:        30 * time.Second,
	}

	tests := []struct {
		name          string
		attemptNumber int
		expected      time.Duration
	}{
		{
			name:          "First retry",
			attemptNumber: 0,
			expected:      2 * time.Second,
		},
		{
			name:          "Second retry",
			attemptNumber: 1,
			expected:      3 * time.Second,
		},
		{
			name:          "Third retry (rounded)",
			attemptNumber: 2,
			expected:      4500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := policy.CalculateNextRetryInterval(tt.attemptNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}
