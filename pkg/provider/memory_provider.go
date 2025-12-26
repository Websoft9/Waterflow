package provider

import (
	"context"
	"sort"
	"sync"
	"time"
)

// InMemoryProvider is a simple in-memory implementation of ServerGroupProvider.
// Useful for testing and simple deployments without external CMDB.
type InMemoryProvider struct {
	mu     sync.RWMutex
	groups map[string][]ServerInfo // groupName -> servers
}

// NewInMemoryProvider creates a new in-memory provider.
func NewInMemoryProvider() *InMemoryProvider {
	return &InMemoryProvider{
		groups: make(map[string][]ServerInfo),
	}
}

// RegisterServer registers a server to one or more groups.
// This is typically called when an agent starts up.
func (p *InMemoryProvider) RegisterServer(server ServerInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, queue := range server.TaskQueues {
		if p.groups[queue] == nil {
			p.groups[queue] = []ServerInfo{}
		}

		// Check if server already registered (by AgentID)
		found := false
		for i, existing := range p.groups[queue] {
			if existing.AgentID == server.AgentID {
				// Update existing entry
				p.groups[queue][i] = server
				found = true
				break
			}
		}

		if !found {
			p.groups[queue] = append(p.groups[queue], server)
		}
	}

	return nil
}

// GetServers returns all servers in the specified group.
// Note: Health status is computed on-the-fly based on last heartbeat time.
// For high-traffic scenarios, consider caching health status or using a
// background goroutine to periodically update server status.
func (p *InMemoryProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	servers, ok := p.groups[groupName]
	if !ok {
		return []ServerInfo{}, nil // Empty list, not an error
	}

	// Create a copy with health status check
	result := make([]ServerInfo, len(servers))
	now := time.Now()

	for i, server := range servers {
		result[i] = server

		// Auto-detect unhealthy: heartbeat > 90s ago
		if !server.LastHeartbeat.IsZero() {
			timeSinceHeartbeat := now.Sub(server.LastHeartbeat)
			if timeSinceHeartbeat > 90*time.Second {
				result[i].Status = "unhealthy"
			}
		}
	}

	return result, nil
}

// ListGroups returns all available group names.
func (p *InMemoryProvider) ListGroups(ctx context.Context) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	groups := make([]string, 0, len(p.groups))
	for name := range p.groups {
		groups = append(groups, name)
	}

	// Sort to ensure stable ordering
	sort.Strings(groups)

	return groups, nil
}

// Close is a no-op for in-memory provider.
func (p *InMemoryProvider) Close() error {
	return nil
}

// UpdateHeartbeat updates the last heartbeat time for a server.
func (p *InMemoryProvider) UpdateHeartbeat(agentID string, status string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	for _, servers := range p.groups {
		for i, server := range servers {
			if server.AgentID == agentID {
				servers[i].Status = status
				servers[i].LastHeartbeat = now
			}
		}
	}

	return nil
}
