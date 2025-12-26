package provider

import (
	"context"
	"time"
)

// ServerInfo represents information about a single agent/server.
type ServerInfo struct {
	// AgentID is the unique identifier of the agent worker.
	AgentID string `json:"agent_id"`

	// Hostname is the server's hostname.
	Hostname string `json:"hostname"`

	// IPAddress is the server's IP address.
	IPAddress string `json:"ip_address,omitempty"`

	// TaskQueues is the list of task queues this agent polls.
	TaskQueues []string `json:"task_queues"`

	// Status indicates the agent's health status.
	// Values: "healthy", "unhealthy", "unknown"
	Status string `json:"status"`

	// LastHeartbeat is the timestamp of the last heartbeat.
	LastHeartbeat time.Time `json:"last_heartbeat"`

	// Metadata contains additional server attributes (OS, arch, tags, etc.)
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ServerGroupProvider defines the interface for querying server groups.
// Implementations can integrate with CMDB systems, Ansible inventories,
// configuration files, or other sources.
type ServerGroupProvider interface {
	// GetServers returns a list of servers in the specified group.
	// Returns empty list if group doesn't exist or has no servers.
	GetServers(ctx context.Context, groupName string) ([]ServerInfo, error)

	// ListGroups returns all available server group names.
	// This is used for discovery and validation.
	ListGroups(ctx context.Context) ([]string, error)

	// Close releases any resources held by the provider.
	Close() error
}
