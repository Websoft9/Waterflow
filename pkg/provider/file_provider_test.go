package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileProvider_Load(t *testing.T) {
	// Create temp test file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "servers.yaml")

	yaml := `groups:
  linux-amd64:
    servers:
      - agent_id: agent-001
        hostname: build-server-1.example.com
        ip_address: 192.168.1.10
        task_queues:
          - linux-amd64
          - linux-common
        metadata:
          os: linux
          arch: amd64
  web-servers:
    servers:
      - agent_id: agent-web-1
        hostname: web-1.example.com
        ip_address: 10.0.1.20
        task_queues:
          - web-servers
        metadata:
          role: web
`

	err := os.WriteFile(configFile, []byte(yaml), 0644)
	require.NoError(t, err)

	// Load provider
	provider, err := NewFileProvider(configFile)
	require.NoError(t, err)
	defer provider.Close()

	// Test GetServers
	servers, err := provider.GetServers(context.Background(), "linux-amd64")
	require.NoError(t, err)
	assert.Len(t, servers, 1)
	assert.Equal(t, "agent-001", servers[0].AgentID)
	assert.Equal(t, "build-server-1.example.com", servers[0].Hostname)
	assert.Equal(t, "192.168.1.10", servers[0].IPAddress)
	assert.Equal(t, "linux", servers[0].Metadata["os"])

	// Test ListGroups
	groups, err := provider.ListGroups(context.Background())
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"linux-amd64", "web-servers"}, groups)
}

func TestFileProvider_GetServers_EmptyGroup(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "servers.yaml")

	yaml := `groups:
  linux-amd64:
    servers: []
`

	err := os.WriteFile(configFile, []byte(yaml), 0644)
	require.NoError(t, err)

	provider, err := NewFileProvider(configFile)
	require.NoError(t, err)
	defer provider.Close()

	servers, err := provider.GetServers(context.Background(), "non-existent")
	require.NoError(t, err)
	assert.Empty(t, servers)
}

func TestFileProvider_InvalidFile(t *testing.T) {
	provider, err := NewFileProvider("/non/existent/file.yaml")
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func TestFileProvider_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "bad.yaml")

	err := os.WriteFile(configFile, []byte("invalid: [yaml: structure"), 0644)
	require.NoError(t, err)

	provider, err := NewFileProvider(configFile)
	assert.Error(t, err)
	assert.Nil(t, provider)
}

func BenchmarkFileProvider_GetServers(b *testing.B) {
	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "servers.yaml")

	yaml := `groups:
  linux-amd64:
    servers:
      - agent_id: agent-001
        hostname: server1.example.com
        task_queues:
          - linux-amd64
`

	os.WriteFile(configFile, []byte(yaml), 0644)

	provider, _ := NewFileProvider(configFile)
	defer provider.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.GetServers(context.Background(), "linux-amd64")
		if err != nil {
			b.Fatal(err)
		}
	}
	// Expected: < 100ms per operation
}
