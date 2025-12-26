package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestServerInfo tests the ServerInfo struct
func TestServerInfo_Struct(t *testing.T) {
	info := ServerInfo{
		AgentID:    "agent-1",
		Hostname:   "server1.example.com",
		IPAddress:  "192.168.1.10",
		TaskQueues: []string{"linux-amd64", "linux-common"},
		Status:     "healthy",
		Metadata: map[string]string{
			"os":   "linux",
			"arch": "amd64",
		},
	}

	assert.Equal(t, "agent-1", info.AgentID)
	assert.Equal(t, "server1.example.com", info.Hostname)
	assert.Equal(t, "192.168.1.10", info.IPAddress)
	assert.Equal(t, 2, len(info.TaskQueues))
	assert.Equal(t, "healthy", info.Status)
	assert.Equal(t, "linux", info.Metadata["os"])
}

// TestServerGroupProvider_Interface ensures the interface is properly defined
func TestServerGroupProvider_Interface(t *testing.T) {
	var _ ServerGroupProvider = (*mockProvider)(nil)
}

// mockProvider is a test implementation
type mockProvider struct{}

func (m *mockProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	return []ServerInfo{}, nil
}

func (m *mockProvider) ListGroups(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *mockProvider) Close() error {
	return nil
}
