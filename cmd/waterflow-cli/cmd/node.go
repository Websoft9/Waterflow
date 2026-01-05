package cmd

import (
	"github.com/spf13/cobra"
)

func newNodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage workflow nodes",
		Long: `Commands for discovering and inspecting workflow nodes.

Nodes are plugins that provide execution capabilities for workflow steps.`,
	}

	// Add subcommands
	cmd.AddCommand(newNodeListCmd())

	return cmd
}
