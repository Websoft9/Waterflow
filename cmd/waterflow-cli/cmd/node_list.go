package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

var (
	nodeListCategory string
	nodeListSearch   string
	nodeListFormat   string
	nodeListNoGroup  bool
)

func newNodeListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [node-name]",
		Short: "List available workflow nodes",
		Long: `List all available workflow nodes or show details for a specific node.

Examples:
  # List all nodes
  waterflow node list
  
  # Show details for a specific node
  waterflow node list exec/shell
  
  # Filter by category
  waterflow node list --category exec
  
  # Search by name
  waterflow node list --search docker
  
  # JSON output
  waterflow node list --format json`,
		Args: cobra.MaximumNArgs(1),
		RunE: runNodeList,
	}

	cmd.Flags().StringVar(&nodeListCategory, "category", "", "Filter by category")
	cmd.Flags().StringVar(&nodeListSearch, "search", "", "Search nodes by name")
	cmd.Flags().StringVar(&nodeListFormat, "format", "text", "Output format (text, json, yaml, simple)")
	cmd.Flags().BoolVar(&nodeListNoGroup, "no-group", false, "Disable category grouping")

	return cmd
}

func runNodeList(cmd *cobra.Command, args []string) error {
	// Create HTTP client
	httpClient := client.New(serverURL, apiKey, 30*1000000000, debugMode)

	// Query nodes
	nodes, err := httpClient.ListNodes(nodeListCategory, nodeListSearch)
	if err != nil {
		return formatNodeError(err)
	}

	// Single node detail
	if len(args) == 1 {
		return displayNodeDetail(args[0], nodes)
	}

	// Display node list
	return displayNodeList(nodes, nodeListFormat, nodeListNoGroup)
}

func displayNodeList(nodes []client.NodeInfo, format string, noGroup bool) error {
	if len(nodes) == 0 {
		fmt.Println("No nodes found")
		return nil
	}

	switch format {
	case "json":
		data, _ := json.MarshalIndent(map[string]interface{}{
			"nodes": nodes,
			"total": len(nodes),
		}, "", "  ")
		fmt.Println(string(data))

	case "yaml":
		data, _ := yaml.Marshal(map[string]interface{}{
			"nodes": nodes,
			"total": len(nodes),
		})
		fmt.Println(string(data))

	case "simple":
		for _, node := range nodes {
			fmt.Printf("%s@%s\n", node.Name, node.Version)
		}

	default: // text
		if noGroup {
			for _, node := range nodes {
				fmt.Printf("  %-24s %s\n", node.Name+"@"+node.Version, node.Description)
			}
		} else {
			displayNodeListGrouped(nodes)
		}
	}

	return nil
}

func displayNodeListGrouped(nodes []client.NodeInfo) {
	// Group by category
	categories := make(map[string][]client.NodeInfo)
	for _, node := range nodes {
		cat := node.Category
		categories[cat] = append(categories[cat], node)
	}

	// Sort categories
	catNames := make([]string, 0, len(categories))
	for cat := range categories {
		catNames = append(catNames, cat)
	}
	sort.Strings(catNames)

	fmt.Printf("Available Nodes (%d):\n\n", len(nodes))

	categoryTitles := map[string]string{
		"exec":   "Execution",
		"flow":   "Flow Control",
		"http":   "HTTP",
		"file":   "File Transfer",
		"docker": "Docker",
	}

	for _, cat := range catNames {
		title := categoryTitles[cat]
		if title == "" {
			title = cases.Title(language.English).String(cat)
		}

		fmt.Printf("%s:\n", title)
		for _, node := range categories[cat] {
			fmt.Printf("  %-24s %s\n", node.Name+"@"+node.Version, node.Description)
		}
		fmt.Println()
	}

	fmt.Printf("Use 'waterflow node list <name>' to see details for a specific node.\n")
}

func displayNodeDetail(nodeName string, nodes []client.NodeInfo) error {
	// Find node
	var found *client.NodeInfo
	for _, node := range nodes {
		if node.Name == nodeName || node.Name+"@"+node.Version == nodeName {
			found = &node
			break
		}
	}

	if found == nil {
		fmt.Printf("Error: Node not found\n")
		fmt.Printf("  Node: %s\n\n", nodeName)
		fmt.Printf("Suggestion: Use 'waterflow node list' to see all available nodes\n")
		os.Exit(1)
	}

	// Display details
	fmt.Printf("Node: %s@%s\n", found.Name, found.Version)
	fmt.Printf("Category: %s\n", found.Category)
	fmt.Printf("Description: %s\n\n", found.Description)

	if len(found.InputSchema) > 0 {
		fmt.Printf("Input Parameters:\n")
		for name, param := range found.InputSchema {
			required := ""
			if req, ok := param["required"].(bool); ok && req {
				required = ", required"
			} else {
				required = ", optional"
			}
			fmt.Printf("  %s (%s%s)\n", name, param["type"], required)
			if desc, ok := param["description"].(string); ok {
				fmt.Printf("    Description: %s\n", desc)
			}
			if def, ok := param["default"]; ok {
				fmt.Printf("    Default: %v\n", def)
			}
		}
		fmt.Println()
	}

	if len(found.OutputSchema) > 0 {
		fmt.Printf("Output:\n")
		for name, typ := range found.OutputSchema {
			fmt.Printf("  %s (%v)\n", name, typ)
		}
		fmt.Println()
	}

	fmt.Printf("Usage Example:\n")
	fmt.Printf("  - name: Run command\n")
	fmt.Printf("    uses: %s@%s\n", found.Name, found.Version)
	if len(found.InputSchema) > 0 {
		fmt.Printf("    with:\n")
		// Show all required parameters with example values
		hasRequired := false
		for name, param := range found.InputSchema {
			if req, ok := param["required"].(bool); ok && req {
				exampleValue := getExampleValue(param)
				fmt.Printf("      %s: %s\n", name, exampleValue)
				hasRequired = true
			}
		}
		// If no required params, show first param
		if !hasRequired {
			for name := range found.InputSchema {
				fmt.Printf("      %s: \"...\"\n", name)
				break
			}
		}
	}

	return nil
}

func formatNodeError(err error) error {
	if serverErr, ok := err.(*client.ServerError); ok {
		fmt.Fprintf(os.Stderr, "Error: Failed to query nodes\n")
		fmt.Fprintf(os.Stderr, "  Status: %d\n", serverErr.StatusCode)
		fmt.Fprintf(os.Stderr, "  Message: %s\n\n", serverErr.Message)
		fmt.Fprintf(os.Stderr, "Suggestion: Check server connectivity\n")
		os.Exit(1)
	}
	return err
}

// getExampleValue generates example value for a parameter
func getExampleValue(param map[string]interface{}) string {
	// Use default value if available
	if def, ok := param["default"]; ok {
		return fmt.Sprintf("%v", def)
	}

	// Use first enum value if available
	if enum, ok := param["enum"].([]interface{}); ok && len(enum) > 0 {
		return fmt.Sprintf("%v", enum[0])
	}

	// Generate based on type
	paramType, _ := param["type"].(string)
	switch paramType {
	case "string":
		return `"example"`
	case "int", "integer":
		return "0"
	case "bool", "boolean":
		return "true"
	case "map", "object":
		return `{"key": "value"}`
	case "array", "list":
		return `["item1", "item2"]`
	default:
		return `"..."`
	}
}
