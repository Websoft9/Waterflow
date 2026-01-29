package dsl_test

import (
	"testing"

	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/stretchr/testify/assert"
)

func TestValidateRunsOn(t *testing.T) {
	tests := []struct {
		name    string
		runsOn  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid: lowercase alphanumeric",
			runsOn:  "linux-amd64",
			wantErr: false,
		},
		{
			name:    "valid: with multiple hyphens",
			runsOn:  "web-servers-prod",
			wantErr: false,
		},
		{
			name:    "valid: numbers",
			runsOn:  "gpu-a100",
			wantErr: false,
		},
		{
			name:    "valid: uppercase",
			runsOn:  "Linux-AMD64",
			wantErr: false,
		},
		{
			name:    "valid: single character",
			runsOn:  "a",
			wantErr: false,
		},
		{
			name:    "valid: expression syntax",
			runsOn:  "${{ matrix.server }}",
			wantErr: false,
		},
		{
			name:    "valid: complex expression",
			runsOn:  "${{ matrix.server }}-agent",
			wantErr: false,
		},
		{
			name:    "invalid: empty",
			runsOn:  "",
			wantErr: true,
			errMsg:  "runs-on cannot be empty",
		},
		{
			name:    "invalid: underscore",
			runsOn:  "linux_amd64",
			wantErr: true,
			errMsg:  "invalid task queue name",
		},
		{
			name:    "invalid: space",
			runsOn:  "web servers",
			wantErr: true,
			errMsg:  "invalid task queue name",
		},
		{
			name:    "invalid: special char",
			runsOn:  "linux@amd64",
			wantErr: true,
			errMsg:  "invalid task queue name",
		},
		{
			name:    "invalid: starts with hyphen",
			runsOn:  "-linux",
			wantErr: true,
			errMsg:  "invalid task queue name",
		},
		{
			name:    "invalid: ends with hyphen",
			runsOn:  "linux-",
			wantErr: true,
			errMsg:  "invalid task queue name",
		},
		{
			name:    "invalid: too long",
			runsOn:  "a" + string(make([]byte, 255)), // 256 characters (1 + 255)
			wantErr: true,
			errMsg:  "task queue name too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dsl.ValidateRunsOn(tt.runsOn)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
