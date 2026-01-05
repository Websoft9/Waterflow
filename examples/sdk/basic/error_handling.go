// Package main demonstrates error handling with Waterflow Go SDK
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Websoft9/waterflow/pkg/sdk"
)

func main() {
	client, err := sdk.NewClient(&sdk.ClientConfig{
		ServerURL: "http://localhost:8080",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Error Handling Examples ===")
	fmt.Println()

	// Example 1: Validation error (invalid YAML)
	fmt.Println("Example 1: Validation Error")
	invalidYAML := `
name: invalid-workflow
jobs:
  build:
    # Missing required 'runs-on' field
    steps:
      - name: test
        uses: invalid-format  # Invalid node format
`
	_, err = client.SubmitWorkflow(context.Background(), &sdk.SubmitWorkflowRequest{
		YAML: invalidYAML,
	})
	handleError(err, "validation")

	// Example 2: Not found error
	fmt.Println("\nExample 2: Not Found Error")
	_, err = client.GetWorkflowStatus(context.Background(), "non-existent-id")
	handleError(err, "not found")

	// Example 3: Empty workflow ID
	fmt.Println("\nExample 3: Client-side Validation")
	_, err = client.GetWorkflowStatus(context.Background(), "")
	handleError(err, "empty ID")

	// Example 4: Successfully handling errors
	fmt.Println("\nExample 4: Correct Usage")
	yamlContent := `
name: valid-workflow
jobs:
  test:
    runs-on: default
    steps:
      - name: echo
        uses: exec/shell@v1
        with:
          command: echo "test"
`
	resp, err := client.SubmitWorkflow(context.Background(), &sdk.SubmitWorkflowRequest{
		YAML: yamlContent,
	})
	if err != nil {
		handleError(err, "submit")
	} else {
		fmt.Printf("   ✅ Success! Workflow ID: %s\n", resp.ID)
	}

	fmt.Println("\n=== Error Handling Complete ===")
}

func handleError(err error, context string) {
	if err == nil {
		fmt.Printf("   ✅ No error in %s\n", context)
		return
	}

	// Check for specific error types
	if sdk.IsNotFound(err) {
		fmt.Printf("   ❌ Not Found Error (%s)\n", context)
		fmt.Printf("      Message: %v\n", err)
		return
	}

	if sdk.IsValidationError(err) {
		fmt.Printf("   ❌ Validation Error (%s)\n", context)
		fmt.Printf("      Message: %v\n", err)
		if serverErr, ok := err.(*sdk.ServerError); ok && serverErr.Details != nil {
			fmt.Printf("      Details: %+v\n", serverErr.Details)
		}
		return
	}

	// Generic error
	fmt.Printf("   ❌ Error (%s): %v\n", context, err)
}
