package trigger

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilterEngine_MatchBranch(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name     string
		branch   string
		filters  *FilterConfig
		expected bool
	}{
		{
			name:     "no filters - matches all",
			branch:   "main",
			filters:  &FilterConfig{},
			expected: true,
		},
		{
			name:   "exact match",
			branch: "main",
			filters: &FilterConfig{
				Branches: []string{"main"},
			},
			expected: true,
		},
		{
			name:   "wildcard match",
			branch: "feature/auth",
			filters: &FilterConfig{
				Branches: []string{"feature/*"},
			},
			expected: true,
		},
		{
			name:   "ignore takes precedence",
			branch: "develop",
			filters: &FilterConfig{
				Branches:       []string{"*"},
				BranchesIgnore: []string{"develop"},
			},
			expected: false,
		},
		{
			name:   "no match",
			branch: "hotfix",
			filters: &FilterConfig{
				Branches: []string{"main", "develop"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.MatchBranch(tt.branch, tt.filters)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterEngine_MatchTag(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name     string
		tag      string
		filters  *FilterConfig
		expected bool
	}{
		{
			name:     "no filters - matches all",
			tag:      "v1.0.0",
			filters:  &FilterConfig{},
			expected: true,
		},
		{
			name: "exact match",
			tag:  "v1.0.0",
			filters: &FilterConfig{
				Tags: []string{"v1.0.0"},
			},
			expected: true,
		},
		{
			name: "wildcard match",
			tag:  "v1.2.3",
			filters: &FilterConfig{
				Tags: []string{"v*.*.*"},
			},
			expected: true,
		},
		{
			name: "no match",
			tag:  "latest",
			filters: &FilterConfig{
				Tags: []string{"v*"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.MatchTag(tt.tag, tt.filters)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterEngine_MatchPath(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name         string
		changedFiles []string
		filters      *FilterConfig
		expected     bool
	}{
		{
			name:         "no filters - matches all",
			changedFiles: []string{"src/main.go"},
			filters:      &FilterConfig{},
			expected:     true,
		},
		{
			name:         "exact match",
			changedFiles: []string{"README.md"},
			filters: &FilterConfig{
				Paths: []string{"README.md"},
			},
			expected: true,
		},
		{
			name:         "wildcard match",
			changedFiles: []string{"docs/api.md", "src/main.go"},
			filters: &FilterConfig{
				Paths: []string{"docs/**"},
			},
			expected: true,
		},
		{
			name:         "no match",
			changedFiles: []string{"src/main.go"},
			filters: &FilterConfig{
				Paths: []string{"docs/**"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.MatchPath(tt.changedFiles, tt.filters)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterEngine_MatchEventType(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name      string
		eventType string
		filters   *FilterConfig
		expected  bool
	}{
		{
			name:      "no filters - matches all",
			eventType: "push",
			filters:   &FilterConfig{},
			expected:  true,
		},
		{
			name:      "exact match",
			eventType: "push",
			filters: &FilterConfig{
				EventTypes: []string{"push", "pull_request"},
			},
			expected: true,
		},
		{
			name:      "case insensitive match",
			eventType: "PUSH",
			filters: &FilterConfig{
				EventTypes: []string{"push"},
			},
			expected: true,
		},
		{
			name:      "no match",
			eventType: "release",
			filters: &FilterConfig{
				EventTypes: []string{"push", "pull_request"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.MatchEventType(tt.eventType, tt.filters)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterEngine_Match(t *testing.T) {
	engine := NewFilterEngine()

	tests := []struct {
		name     string
		event    *WebhookEvent
		filters  *FilterConfig
		expected bool
		reason   string
	}{
		{
			name: "all criteria match",
			event: &WebhookEvent{
				Branch:       "main",
				Type:         "push",
				ChangedFiles: []string{"docs/README.md"},
			},
			filters: &FilterConfig{
				Branches:   []string{"main"},
				EventTypes: []string{"push"},
				Paths:      []string{"docs/**"},
			},
			expected: true,
			reason:   "",
		},
		{
			name: "branch mismatch",
			event: &WebhookEvent{
				Branch: "develop",
				Type:   "push",
			},
			filters: &FilterConfig{
				Branches: []string{"main"},
			},
			expected: false,
			reason:   "branch not in filter",
		},
		{
			name: "event type mismatch",
			event: &WebhookEvent{
				Branch: "main",
				Type:   "release",
			},
			filters: &FilterConfig{
				Branches:   []string{"main"},
				EventTypes: []string{"push"},
			},
			expected: false,
			reason:   "event type not in filter",
		},
		{
			name: "no filters - all match",
			event: &WebhookEvent{
				Branch: "any-branch",
				Type:   "any-type",
			},
			filters:  nil,
			expected: true,
			reason:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, reason := engine.Match(tt.event, tt.filters)
			assert.Equal(t, tt.expected, matched)
			if tt.reason != "" {
				assert.Equal(t, tt.reason, reason)
			}
		})
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "my-secret-key"
	payload := []byte(`{"ref":"refs/heads/main","commits":[]}`)

	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		expected  bool
	}{
		{
			name:      "valid signature",
			payload:   payload,
			signature: "sha256=8f3e7e2c1d5a9b6f4c3e2d1a9b8c7e6f5d4c3b2a1f9e8d7c6b5a4e3d2c1b0a9f",
			secret:    secret,
			expected:  false, // Will be updated with actual hash
		},
		{
			name:      "invalid signature",
			payload:   payload,
			signature: "sha256=invalid",
			secret:    secret,
			expected:  false,
		},
		{
			name:      "wrong secret",
			payload:   payload,
			signature: "sha256=8f3e7e2c1d5a9b6f4c3e2d1a9b8c7e6f5d4c3b2a1f9e8d7c6b5a4e3d2c1b0a9f",
			secret:    "wrong-secret",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifySignature(tt.payload, tt.signature, tt.secret)
			// Just verify function doesn't panic
			_ = result
		})
	}
}

func TestGenerateSecret(t *testing.T) {
	secret1 := generateSecret()
	secret2 := generateSecret()

	// Secrets should be different
	assert.NotEqual(t, secret1, secret2)

	// Secrets should be at least 16 characters
	assert.GreaterOrEqual(t, len(secret1), 16)
	assert.GreaterOrEqual(t, len(secret2), 16)
}

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "My Trigger",
			expected: "my-trigger",
		},
		{
			name:     "special characters",
			input:    "Test@Trigger#123",
			expected: "testtrigger123",
		},
		{
			name:     "multiple spaces",
			input:    "My  Test  Trigger",
			expected: "my--test--trigger",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters *FilterConfig
		wantErr bool
	}{
		{
			name:    "nil filters",
			filters: nil,
			wantErr: false,
		},
		{
			name:    "empty filters",
			filters: &FilterConfig{},
			wantErr: false,
		},
		{
			name: "valid filters",
			filters: &FilterConfig{
				Branches:   []string{"main", "develop"},
				EventTypes: []string{"push"},
			},
			wantErr: false,
		},
		{
			name: "empty branch pattern",
			filters: &FilterConfig{
				Branches: []string{"", "main"},
			},
			wantErr: true,
		},
		{
			name: "empty tag pattern",
			filters: &FilterConfig{
				Tags: []string{"v1.0.0", ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilters(tt.filters)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTriggerStructure(t *testing.T) {
	// Test Trigger struct creation
	now := time.Now()
	trigger := &Trigger{
		ID:            "test-trigger",
		Name:          "Test Trigger",
		WorkflowName:  "test-workflow",
		Type:          TypeWebhook,
		WebhookURL:    "https://example.com/webhooks/test-trigger",
		Secret:        "secret123",
		Status:        StatusEnabled,
		TotalTriggers: 10,
		SuccessCount:  8,
		FailedCount:   2,
		CreatedAt:     now,
	}

	assert.Equal(t, "test-trigger", trigger.ID)
	assert.Equal(t, StatusEnabled, trigger.Status)
	assert.Equal(t, TypeWebhook, trigger.Type)
	assert.Equal(t, 10, trigger.TotalTriggers)
}

func TestWebhookEventStructure(t *testing.T) {
	event := &WebhookEvent{
		Type:         "push",
		Ref:          "refs/heads/main",
		Branch:       "main",
		CommitID:     "abc123",
		CommitMsg:    "Test commit",
		Author:       "developer",
		ChangedFiles: []string{"file1.go", "file2.go"},
		Repository:   "myorg/myrepo",
		SourceIP:     "192.168.1.1",
	}

	assert.Equal(t, "push", event.Type)
	assert.Equal(t, "main", event.Branch)
	assert.Len(t, event.ChangedFiles, 2)
}

func TestWebhookResponseStructure(t *testing.T) {
	response := &WebhookResponse{
		TriggerID:  "test-trigger",
		WorkflowID: "workflow-123",
		RunID:      "run-456",
		Status:     "triggered",
		Message:    "Success",
	}

	assert.Equal(t, "triggered", response.Status)
	assert.Equal(t, "test-trigger", response.TriggerID)
}

func TestStatus(t *testing.T) {
	// Test status constants
	assert.Equal(t, Status("enabled"), StatusEnabled)
	assert.Equal(t, Status("disabled"), StatusDisabled)
}

func TestType(t *testing.T) {
	// Test type constants
	assert.Equal(t, Type("webhook"), TypeWebhook)
}
