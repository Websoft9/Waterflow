package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateWorkflowID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
		wantMsg bool // 是否期望警告消息
	}{
		{
			name:    "valid UUID",
			id:      "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
			wantMsg: false,
		},
		{
			name:    "another valid UUID",
			id:      "123e4567-e89b-12d3-a456-426614174000",
			wantErr: false,
			wantMsg: false,
		},
		{
			name:    "invalid UUID - too short",
			id:      "550e8400-e29b-41d4",
			wantErr: false, // 不返回错误，只是警告
			wantMsg: true,
		},
		{
			name:    "invalid UUID - no hyphens",
			id:      "550e8400e29b41d4a716446655440000",
			wantErr: false,
			wantMsg: true,
		},
		{
			name:    "invalid UUID - wrong format",
			id:      "invalid-id",
			wantErr: false,
			wantMsg: true,
		},
		{
			name:    "empty string",
			id:      "",
			wantErr: true,
			wantMsg: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkflowID(tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "required")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsTerminalStatusState(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "completed is terminal",
			status: "completed",
			want:   true,
		},
		{
			name:   "failed is terminal",
			status: "failed",
			want:   true,
		},
		{
			name:   "cancelled is terminal",
			status: "cancelled",
			want:   true,
		},
		{
			name:   "timeout is terminal",
			status: "timeout",
			want:   true,
		},
		{
			name:   "running is not terminal",
			status: "running",
			want:   false,
		},
		{
			name:   "pending is not terminal",
			status: "pending",
			want:   false,
		},
		{
			name:   "unknown status is not terminal",
			status: "unknown",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTerminalStatusState(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}
