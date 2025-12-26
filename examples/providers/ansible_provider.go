// Package examples provides example implementations of custom ServerGroupProviders.
// This shows how to integrate Waterflow with external CMDB systems.
package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Websoft9/waterflow/pkg/provider"
)

// AnsibleInventoryProvider integrates with Ansible inventory.
// This is an example implementation showing how to create custom providers.
type AnsibleInventoryProvider struct {
	inventoryPath string
}

// NewAnsibleInventoryProvider creates a provider from Ansible inventory.
func NewAnsibleInventoryProvider(inventoryPath string) *AnsibleInventoryProvider {
	return &AnsibleInventoryProvider{
		inventoryPath: inventoryPath,
	}
}

// GetServers queries Ansible inventory for a specific group.
func (p *AnsibleInventoryProvider) GetServers(ctx context.Context, groupName string) ([]provider.ServerInfo, error) {
	// Execute: ansible-inventory -i <inventory> --list --export
	cmd := exec.CommandContext(ctx, "ansible-inventory",
		"-i", p.inventoryPath,
		"--list",
		"--export",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query ansible inventory: %w", err)
	}

	// Parse JSON output
	var inventory map[string]interface{}
	if err := json.Unmarshal(output, &inventory); err != nil {
		return nil, fmt.Errorf("failed to parse inventory: %w", err)
	}

	// Extract hosts from group
	groupData, ok := inventory[groupName].(map[string]interface{})
	if !ok {
		return []provider.ServerInfo{}, nil
	}

	hosts, ok := groupData["hosts"].([]interface{})
	if !ok {
		return []provider.ServerInfo{}, nil
	}

	// Convert to ServerInfo
	servers := make([]provider.ServerInfo, 0, len(hosts))
	for _, host := range hosts {
		hostname := host.(string)
		servers = append(servers, provider.ServerInfo{
			AgentID:    fmt.Sprintf("ansible-%s", hostname),
			Hostname:   hostname,
			TaskQueues: []string{groupName},
			Status:     "unknown",
			Metadata: map[string]string{
				"source": "ansible-inventory",
			},
		})
	}

	return servers, nil
}

// ListGroups returns all groups in Ansible inventory.
// Note: This is a simplified implementation for demonstration purposes.
func (p *AnsibleInventoryProvider) ListGroups(ctx context.Context) ([]string, error) {
	// Execute: ansible-inventory -i <inventory> --list --export
	cmd := exec.CommandContext(ctx, "ansible-inventory",
		"-i", p.inventoryPath,
		"--list",
		"--export",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query ansible inventory: %w", err)
	}

	// Parse JSON output
	var inventory map[string]interface{}
	if err := json.Unmarshal(output, &inventory); err != nil {
		return nil, fmt.Errorf("failed to parse inventory: %w", err)
	}

	// Extract group names (top-level keys)
	groups := make([]string, 0, len(inventory))
	for groupName := range inventory {
		// Skip meta groups like "_meta"
		if groupName != "_meta" {
			groups = append(groups, groupName)
		}
	}

	return groups, nil
}

// Close releases resources.
func (p *AnsibleInventoryProvider) Close() error {
	return nil
}
