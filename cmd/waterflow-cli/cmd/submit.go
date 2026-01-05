package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
	clierrors "github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/errors"
	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/output"
	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/validator"
	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	submitWait     bool
	submitFollow   bool
	submitValidate bool
	submitQuiet    bool
	submitFormat   string
	submitVars     []string
)

// ExitError represents an error with an exit code
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}

func newSubmitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit <workflow-file>",
		Short: "Submit workflow for execution",
		Long: `Submit a workflow to the Waterflow server for execution.

By default, the workflow is submitted and the command returns immediately.
Use --wait to wait for completion or --follow to stream logs in real-time.

Examples:
  # Quick submit (returns workflow ID)
  waterflow submit workflow.yaml
  
  # Submit with variable overrides
  waterflow submit deployment.yaml --var env=production
  
  # Wait for completion
  waterflow submit --wait workflow.yaml
  
  # Follow logs in real-time
  waterflow submit --follow workflow.yaml
  
  # Validate before submitting
  waterflow submit --validate workflow.yaml
  
  # JSON output for automation
  waterflow submit --format json workflow.yaml
  
  # Quiet mode (only workflow ID)
  WORKFLOW_ID=$(waterflow submit --quiet workflow.yaml)`,
		Args: cobra.ExactArgs(1),
		RunE: runSubmit,
	}

	cmd.Flags().BoolVarP(&submitWait, "wait", "w", false, "Wait for workflow completion")
	cmd.Flags().BoolVarP(&submitFollow, "follow", "f", false, "Follow logs in real-time (implies --wait)")
	cmd.Flags().BoolVar(&submitValidate, "validate", false, "Validate workflow before submitting")
	cmd.Flags().BoolVarP(&submitQuiet, "quiet", "q", false, "Only output workflow ID (for scripts)")
	cmd.Flags().StringVar(&submitFormat, "format", "text", "Output format (text, json)")
	cmd.Flags().StringArrayVar(&submitVars, "var", []string{}, "Override variables (key=value)")

	return cmd
}

func runSubmit(cmd *cobra.Command, args []string) error {
	filepath := args[0]

	// Create logger
	var loggerInstance *zap.Logger
	if debugMode {
		loggerInstance, _ = zap.NewDevelopment()
	} else {
		loggerInstance = zap.NewNop()
	}
	defer func() { _ = loggerInstance.Sync() }()

	// 1. Optional: Local validation (AC5)
	if submitValidate {
		if !submitQuiet && submitFormat == "text" {
			fmt.Println("Validating workflow...")
		}

		if err := validateWorkflowLocal(filepath, loggerInstance, submitQuiet); err != nil {
			return err
		}

		if !submitQuiet && submitFormat == "text" {
			fmt.Println("✓ Workflow is valid")
			fmt.Println("\nSubmitting to server...")
		}
	}

	// 2. Read file
	content, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: File not found\n")
			fmt.Fprintf(os.Stderr, "  Path: %s\n\n", filepath)
			fmt.Fprintf(os.Stderr, "Suggestion: Check file path or use 'waterflow validate --help'\n")
			return &ExitError{Code: clierrors.ExitUsageError}
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 3. Parse variable overrides (AC2)
	vars, err := parseVars(submitVars)
	if err != nil {
		return err
	}

	// 4. Create HTTP client
	timeout := 30 * time.Second
	httpClient := client.New(serverURL, apiKey, timeout, debugMode)

	// 5. Submit workflow
	result, err := httpClient.SubmitWorkflow(context.Background(), string(content), vars)
	if err != nil {
		return formatSubmitError(err, submitFormat)
	}

	// 6. Validate workflow ID format
	if !isValidWorkflowID(result.ID) {
		return fmt.Errorf("invalid workflow ID format received from server: %s", result.ID)
	}

	// 7. Output result
	formatter := output.New(submitFormat)

	if submitQuiet {
		// Quiet mode: only output ID
		fmt.Println(result.ID)
		return nil
	}

	if err := printSubmitResult(result, formatter); err != nil {
		return err
	}

	// 8. Optional: Wait for completion (AC3)
	if submitWait || submitFollow {
		return waitForCompletion(httpClient, result.ID, submitFollow, formatter)
	}

	return nil
}

// parseVars parses --var parameters
func parseVars(varArgs []string) (map[string]interface{}, error) {
	if len(varArgs) == 0 {
		return nil, nil
	}

	vars := make(map[string]interface{})

	for _, arg := range varArgs {
		key, value, err := parseVarArg(arg)
		if err != nil {
			return nil, err
		}
		vars[key] = value
	}

	return vars, nil
}

// parseVarArg parses a single variable argument (key=value)
func parseVarArg(arg string) (string, interface{}, error) {
	parts := strings.SplitN(arg, "=", 2)
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "Error: Invalid variable format\n")
		fmt.Fprintf(os.Stderr, "  Value: '%s'\n", arg)
		fmt.Fprintf(os.Stderr, "  Expected: key=value\n\n")
		fmt.Fprintf(os.Stderr, "Example: --var env=production --var timeout=300\n")
		return "", nil, &ExitError{Code: clierrors.ExitUsageError}
	}

	key := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSpace(parts[1])

	if key == "" {
		fmt.Fprintf(os.Stderr, "Error: Empty variable key\n")
		fmt.Fprintf(os.Stderr, "  Value: '%s'\n\n", arg)
		fmt.Fprintf(os.Stderr, "Example: --var key=value\n")
		return "", nil, &ExitError{Code: clierrors.ExitUsageError}
	}

	// Limit key length to prevent abuse
	if len(key) > 256 {
		fmt.Fprintf(os.Stderr, "Error: Variable key too long (max 256 characters)\n")
		fmt.Fprintf(os.Stderr, "  Key: '%s...'\n\n", key[:50])
		return "", nil, &ExitError{Code: clierrors.ExitUsageError}
	}

	// Try to parse value to appropriate type
	value := parseValue(valueStr)

	return key, value, nil
}

// parseValue intelligently parses value type
// Type inference priority: JSON > bool > int > float > string
// Note: Plain numbers like "123" will be parsed as numbers.
// To force string type, use JSON string syntax: --var version='"123"'
func parseValue(s string) interface{} {
	// Try parsing as JSON (supports complex types)
	var jsonValue interface{}
	if err := json.Unmarshal([]byte(s), &jsonValue); err == nil {
		return jsonValue
	}

	// Try parsing as boolean
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}

	// Try parsing as integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	// Try parsing as float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Default to string
	return s
}

// validateWorkflowLocal validates workflow locally before submitting
func validateWorkflowLocal(filepath string, logger *zap.Logger, quiet bool) error {
	// Create DSL validator
	dslValidator, err := dsl.NewValidator(logger)
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}

	// Create local validator
	localValidator := validator.NewLocalValidator(dslValidator, logger)

	// Validate file
	result, err := localValidator.Validate(filepath)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid {
		// Display validation errors (suppress in quiet mode)
		if !quiet {
			fmt.Println("✗ Validation failed")
			fmt.Println()
			for i, err := range result.Errors {
				fmt.Printf("  %d. ", i+1)
				if err.Line > 0 {
					fmt.Printf("Line %d: ", err.Line)
				}
				fmt.Printf("%s\n", err.Message)
				if err.Suggestion != "" {
					fmt.Printf("     Suggestion: %s\n", err.Suggestion)
				}
			}
			fmt.Println()
			fmt.Println("Workflow not submitted.")
		}
		return &ExitError{Code: clierrors.ExitGeneralError}
	}

	return nil
}

// printSubmitResult prints submit result in text format
func printSubmitResult(result *client.SubmitWorkflowResult, formatter *output.Formatter) error {
	if submitFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Text format
	fmt.Println("✓ Workflow submitted successfully")
	fmt.Println()

	fmt.Printf("Workflow ID:   %s\n", result.ID)
	fmt.Printf("Run ID:        %s\n", result.RunID)
	fmt.Printf("Name:          %s\n", result.Name)
	fmt.Printf("Status:        %s\n", result.Status)
	fmt.Printf("Created:       %s\n", result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))

	fmt.Println()
	fmt.Printf("View status:   waterflow status %s\n", result.ID)
	fmt.Printf("View logs:     waterflow logs %s\n", result.ID)

	return nil
}

// formatSubmitError formats submission errors
func formatSubmitError(err error, format string) error {
	if format == "json" {
		return formatSubmitErrorJSON(err)
	}

	// Check if it's a server error
	if serverErr, ok := err.(*client.ServerError); ok {
		return formatSubmitServerError(serverErr)
	}

	// Network or other errors
	return err
}

// formatSubmitServerError formats server errors with helpful suggestions
func formatSubmitServerError(err *client.ServerError) error {
	fmt.Println("Error: Workflow submission failed")
	fmt.Printf("  Status: %d\n\n", err.StatusCode)

	switch err.Code {
	case "validation_error":
		fmt.Println("Validation errors:")
		if errors, ok := err.Details["errors"].([]interface{}); ok {
			for i, e := range errors {
				errMap, ok := e.(map[string]interface{})
				if !ok {
					continue
				}
				fmt.Printf("  %d. ", i+1)
				if line, ok := errMap["line"].(float64); ok {
					fmt.Printf("Line %d: ", int(line))
				}
				if msg, ok := errMap["message"].(string); ok {
					fmt.Printf("%s\n", msg)
				}
				if suggestion, ok := errMap["suggestion"].(string); ok && suggestion != "" {
					fmt.Printf("     Suggestion: %s\n", suggestion)
				}
			}
		}
		fmt.Println("\nSuggestion: Run 'waterflow validate <file>' to check syntax locally")

	case "invalid_request":
		fmt.Printf("Message: %s\n", err.Message)
		fmt.Println("\nSuggestion: Check request format and try again")

	default:
		fmt.Printf("Message: %s\n", err.Message)
		if err.StatusCode >= 500 {
			fmt.Println("\nSuggestion: Check server logs for details or contact administrator")
		}
	}

	return &ExitError{Code: clierrors.ExitGeneralError}
}

// formatSubmitErrorJSON formats error as JSON
func formatSubmitErrorJSON(err error) error {
	if serverErr, ok := err.(*client.ServerError); ok {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    serverErr.Code,
				"message": serverErr.Message,
				"details": serverErr.Details,
			},
		})
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "client_error",
				"message": err.Error(),
			},
		})
	}
	return &ExitError{Code: clierrors.ExitGeneralError}
}

// waitForCompletion waits for workflow completion with timeout
func waitForCompletion(c *client.Client, workflowID string, followLogs bool, formatter *output.Formatter) error {
	if !followLogs {
		fmt.Println("\nWaiting for completion...")
	}

	// Default timeout: 30 minutes (can be made configurable later)
	timeout := 30 * time.Minute
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	ticker := time.NewTicker(2 * time.Second) // Poll every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-timeoutTimer.C:
			fmt.Fprintf(os.Stderr, "\nError: Workflow execution timeout (exceeded %v)\n", timeout)
			fmt.Fprintf(os.Stderr, "Workflow may still be running on the server.\n")
			fmt.Fprintf(os.Stderr, "Check status: waterflow status %s\n", workflowID)
			return &ExitError{Code: clierrors.ExitGeneralError}
		case <-ticker.C:
			// Query workflow status
			status, err := c.GetWorkflowStatus(workflowID)
			if err != nil {
				return fmt.Errorf("failed to get workflow status: %w", err)
			}

			// Display progress
			if !followLogs {
				displayProgress(status)
			}

			// Check if completed
			if isTerminalStatusInSubmit(status.Status) {
				return handleCompletion(status)
			}
		}
	}
}

// displayProgress displays execution progress
func displayProgress(status *client.WorkflowStatus) {
	timestamp := time.Now().Format("15:04:05")

	// Display current status
	fmt.Printf("[%s] %s", timestamp, status.Status)

	// Display current Job/Step from Jobs list
	for _, job := range status.Jobs {
		if job.Status == "running" {
			fmt.Printf(" - Job: %s", job.Name)
			for _, step := range job.Steps {
				if step.Status == "running" {
					fmt.Printf(", Step: %s", step.Name)
					break
				}
			}
			break
		}
	}

	fmt.Println()
}

// isTerminalStatusInSubmit checks if status is terminal
func isTerminalStatusInSubmit(status string) bool {
	return status == "completed" || status == "failed" || status == "cancelled"
}

// handleCompletion handles workflow completion
func handleCompletion(status *client.WorkflowStatus) error {
	fmt.Println()

	if status.Status == "completed" {
		fmt.Println("✓ Workflow completed successfully")
	} else {
		fmt.Println("✗ Workflow failed")
	}
	fmt.Println()

	// Display final status
	fmt.Printf("Final Status:    %s\n", status.Status)

	// Display error on failure
	if status.Status == "failed" {
		fmt.Printf("\nView logs:       waterflow logs %s\n", status.ID)
	}

	// Return exit code (only completed returns 0)
	if status.Status != "completed" {
		return &ExitError{Code: clierrors.ExitGeneralError}
	}

	return nil
}

// isValidWorkflowID validates workflow ID format (UUID)
func isValidWorkflowID(id string) bool {
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (hexadecimal)
	if len(id) != 36 {
		return false
	}
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}
	// Validate hexadecimal characters in each segment
	for i, c := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // Skip hyphens
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func init() {
	// Register submit command
	rootCmd.AddCommand(newSubmitCmd())
}
