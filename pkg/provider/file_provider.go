package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// FileProvider loads server groups from a YAML configuration file.
type FileProvider struct {
	filePath string
	groups   map[string][]ServerInfo
}

// ServerGroupConfig represents the YAML structure for server groups.
type ServerGroupConfig struct {
	Groups map[string]GroupConfig `yaml:"groups"`
}

type GroupConfig struct {
	Servers []ServerConfig `yaml:"servers"`
}

type ServerConfig struct {
	AgentID    string            `yaml:"agent_id"`
	Hostname   string            `yaml:"hostname"`
	IPAddress  string            `yaml:"ip_address,omitempty"`
	TaskQueues []string          `yaml:"task_queues"`
	Metadata   map[string]string `yaml:"metadata,omitempty"`
}

// NewFileProvider creates a provider from a YAML file.
func NewFileProvider(filePath string) (*FileProvider, error) {
	// Security: Clean and validate file path
	cleanPath := filepath.Clean(filePath)
	if !filepath.IsAbs(cleanPath) {
		return nil, fmt.Errorf("file path must be absolute: %s", filePath)
	}

	// Check file size to prevent DoS
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	if fileInfo.Size() > 10*1024*1024 { // 10MB limit
		return nil, fmt.Errorf("file too large: %d bytes (max 10MB)", fileInfo.Size())
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config ServerGroupConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert to internal format
	groups := make(map[string][]ServerInfo)
	for groupName, groupConfig := range config.Groups {
		servers := make([]ServerInfo, 0, len(groupConfig.Servers))
		for _, sc := range groupConfig.Servers {
			servers = append(servers, ServerInfo{
				AgentID:       sc.AgentID,
				Hostname:      sc.Hostname,
				IPAddress:     sc.IPAddress,
				TaskQueues:    sc.TaskQueues,
				Status:        "unknown", // File doesn't track real-time status
				LastHeartbeat: time.Time{},
				Metadata:      sc.Metadata,
			})
		}
		groups[groupName] = servers
	}

	return &FileProvider{
		filePath: filePath,
		groups:   groups,
	}, nil
}

// GetServers returns servers in the specified group.
func (p *FileProvider) GetServers(ctx context.Context, groupName string) ([]ServerInfo, error) {
	servers, ok := p.groups[groupName]
	if !ok {
		return []ServerInfo{}, nil
	}

	// Return a copy
	result := make([]ServerInfo, len(servers))
	copy(result, servers)
	return result, nil
}

// ListGroups returns all group names.
func (p *FileProvider) ListGroups(ctx context.Context) ([]string, error) {
	groups := make([]string, 0, len(p.groups))
	for name := range p.groups {
		groups = append(groups, name)
	}
	return groups, nil
}

// Close is a no-op for file provider.
func (p *FileProvider) Close() error {
	return nil
}
