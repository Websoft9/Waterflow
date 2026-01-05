// Package main demonstrates the 5-minute quickstart for Waterflow Go SDK
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Websoft9/waterflow/pkg/sdk"
)

func main() {
	fmt.Println("=== Waterflow Go SDK Quickstart ===")

	// Step 1: Create client
	fmt.Println("Creating client...")
	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	fmt.Println("✓ Client created")

	// Step 2: Read workflow YAML
	yamlContent, err := os.ReadFile("workflow.yaml")
	if err != nil {
		log.Fatalf("Failed to read workflow.yaml: %v", err)
	}

	ctx := context.Background()

	// Step 3: Submit workflow
	fmt.Println("\nSubmitting workflow...")
	resp, err := client.SubmitWorkflow(ctx, &sdk.SubmitWorkflowRequest{
		YAML: string(yamlContent),
	})
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}
	fmt.Printf("✓ Workflow submitted: %s\n", resp.ID)

	// Step 4: Get status
	fmt.Println("\nChecking status...")
	status, err := client.GetWorkflowStatus(ctx, resp.ID)
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}
	fmt.Printf("✓ Status: %s\n", status.Status)

	// Step 5: Get logs
	fmt.Println("\nFetching logs...")
	logs, err := client.GetWorkflowLogs(ctx, &sdk.GetLogsRequest{
		WorkflowID: resp.ID,
		Tail:       50,
	})
	if err != nil {
		log.Fatalf("Failed to get logs: %v", err)
	}
	fmt.Printf("✓ Retrieved %d log entries\n", len(logs))
	for i, logEntry := range logs {
		if i >= 5 {
			fmt.Printf("... and %d more\n", len(logs)-5)
			break
		}
		fmt.Printf("  [%s] %s\n", logEntry.Level, logEntry.Message)
	}

	fmt.Println("\n✅ Quickstart complete!")
}
