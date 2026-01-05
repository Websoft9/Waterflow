// Package main provides a simple example of using Waterflow Go SDK
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Websoft9/waterflow/pkg/sdk"
)

func main() {
	// Create SDK client
	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("=== Waterflow Go SDK Example ===")

	// Example workflow YAML
	yamlContent := `
name: sdk-test-workflow
jobs:
  test-job:
    runs-on: local
    steps:
      - name: Echo test
        uses: exec/shell@v1
        with:
          command: echo "Hello from Go SDK"
`

	// Submit workflow
	fmt.Println("Submitting workflow...")
	resp, err := client.SubmitWorkflow(context.Background(), &sdk.SubmitWorkflowRequest{
		YAML: yamlContent,
	})
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("✓ Workflow submitted: %s\n", resp.ID)
	fmt.Printf("  Name: %s, Status: %s\n", resp.Name, resp.Status)
}
