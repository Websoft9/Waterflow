package dsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// TestStrategy_Parsing 测试 Strategy 结构解析
func TestStrategy_Parsing(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    *Strategy
		wantErr bool
	}{
		{
			name: "simple matrix",
			yaml: "matrix:\n  server: [web1, web2, web3]\n",
			want: &Strategy{
				Matrix: map[string][]interface{}{
					"server": {"web1", "web2", "web3"},
				},
			},
			wantErr: false,
		},
		{
			name: "multi-dimension matrix",
			yaml: `
matrix:
  server: [web1, web2]
  env: [prod, staging]
`,
			want: &Strategy{
				Matrix: map[string][]interface{}{
					"server": {"web1", "web2"},
					"env":    {"prod", "staging"},
				},
			},
			wantErr: false,
		},
		{
			name: "matrix with different types",
			yaml: `
matrix:
  version: [1.20, 1.21, 1.22]
  os: [ubuntu, debian]
  enabled: [true, false]
`,
			want: &Strategy{
				Matrix: map[string][]interface{}{
					"version": {1.20, 1.21, 1.22},
					"os":      {"ubuntu", "debian"},
					"enabled": {true, false},
				},
			},
			wantErr: false,
		},
		{
			name: "matrix with max-parallel",
			yaml: `
matrix:
  server: [web1, web2, web3]
max-parallel: 2
`,
			want: &Strategy{
				Matrix: map[string][]interface{}{
					"server": {"web1", "web2", "web3"},
				},
				MaxParallel: 2,
			},
			wantErr: false,
		},
		{
			name: "matrix with fail-fast true",
			yaml: `
matrix:
  server: [web1, web2]
fail-fast: true
`,
			want: &Strategy{
				Matrix: map[string][]interface{}{
					"server": {"web1", "web2"},
				},
				FailFast: boolPtr(true),
			},
			wantErr: false,
		},
		{
			name: "matrix with fail-fast false",
			yaml: `
matrix:
  server: [web1, web2]
fail-fast: false
`,
			want: &Strategy{
				Matrix: map[string][]interface{}{
					"server": {"web1", "web2"},
				},
				FailFast: boolPtr(false),
			},
			wantErr: false,
		},
		{
			name: "matrix with all options",
			yaml: `
matrix:
  server: [web1, web2]
  env: [prod, staging]
max-parallel: 2
fail-fast: false
`,
			want: &Strategy{
				Matrix: map[string][]interface{}{
					"server": {"web1", "web2"},
					"env":    {"prod", "staging"},
				},
				MaxParallel: 2,
				FailFast:    boolPtr(false),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Strategy
			err := yaml.Unmarshal([]byte(tt.yaml), &got)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want.Matrix, got.Matrix)
			assert.Equal(t, tt.want.MaxParallel, got.MaxParallel)
			assert.Equal(t, tt.want.FailFast, got.FailFast)
		})
	}
}

// TestJob_WithStrategy 测试 Job 包含 Strategy 字段
func TestJob_WithStrategy(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    *Job
		wantErr bool
	}{
		{
			name: "job with matrix strategy",
			yaml: `
runs-on: linux-amd64
strategy:
  matrix:
    server: [web1, web2, web3]
steps:
  - name: Deploy
    uses: deploy@v1
`,
			want: &Job{
				RunsOn: "linux-amd64",
				Strategy: &Strategy{
					Matrix: map[string][]interface{}{
						"server": {"web1", "web2", "web3"},
					},
				},
				Steps: []*Step{
					{
						Name: "Deploy",
						Uses: "deploy@v1",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "job without strategy",
			yaml: `
runs-on: linux-amd64
steps:
  - name: Deploy
    uses: deploy@v1
`,
			want: &Job{
				RunsOn: "linux-amd64",
				Steps: []*Step{
					{
						Name: "Deploy",
						Uses: "deploy@v1",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Job
			err := yaml.Unmarshal([]byte(tt.yaml), &got)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want.RunsOn, got.RunsOn)
			if tt.want.Strategy != nil {
				assert.NotNil(t, got.Strategy)
				assert.Equal(t, tt.want.Strategy.Matrix, got.Strategy.Matrix)
			} else {
				assert.Nil(t, got.Strategy)
			}
		})
	}
}

// TestWorkflow_WithMatrixJob 测试完整工作流包含 Matrix Job
func TestWorkflow_WithMatrixJob(t *testing.T) {
	yamlContent := `
name: matrix-test
on: push
jobs:
  deploy:
    runs-on: linux-amd64
    strategy:
      matrix:
        server: [web1, web2, web3]
        env: [prod, staging]
      max-parallel: 2
      fail-fast: false
    steps:
      - name: Deploy to Server
        uses: deploy@v1
        with:
          server: ${{ matrix.server }}
          environment: ${{ matrix.env }}
`

	var workflow Workflow
	err := yaml.Unmarshal([]byte(yamlContent), &workflow)
	assert.NoError(t, err)

	assert.Equal(t, "matrix-test", workflow.Name)
	assert.Contains(t, workflow.Jobs, "deploy")

	job := workflow.Jobs["deploy"]
	assert.NotNil(t, job.Strategy)
	assert.Equal(t, map[string][]interface{}{
		"server": {"web1", "web2", "web3"},
		"env":    {"prod", "staging"},
	}, job.Strategy.Matrix)
	assert.Equal(t, 2, job.Strategy.MaxParallel)
	assert.Equal(t, false, *job.Strategy.FailFast)
}

// boolPtr 辅助函数
func boolPtr(b bool) *bool {
	return &b
}
