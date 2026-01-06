package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/validator"
	"github.com/Websoft9/waterflow/pkg/dsl"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	validateRemote    bool
	validateVerbose   bool
	validateFormat    string
	validateRecursive bool
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <workflow-file>...",
		Short: "Validate workflow YAML syntax",
		Long: `Validate workflow YAML syntax and structure.

By default, performs local validation without connecting to the server.
Use --remote flag to validate against the server (checks node availability).

Examples:
  # Validate single file (local)
  waterflow validate workflow.yaml
  
  # Validate multiple files
  waterflow validate workflow1.yaml workflow2.yaml
  
  # Validate with server
  waterflow validate --remote workflow.yaml
  
  # Validate all YAML files in directory
  waterflow validate --recursive ./workflows/
  
  # JSON output for automation
  waterflow validate --format json workflow.yaml`,
		Args: cobra.MinimumNArgs(1),
		RunE: runValidate,
	}

	cmd.Flags().BoolVar(&validateRemote, "remote", false, "Validate against server (checks node availability)")
	cmd.Flags().BoolVarP(&validateVerbose, "verbose", "v", false, "Show detailed validation information")
	cmd.Flags().StringVar(&validateFormat, "format", "text", "Output format (text, json, yaml)")
	cmd.Flags().BoolVarP(&validateRecursive, "recursive", "r", false, "Recursively validate YAML files in directory")

	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Create logger
	var logger *zap.Logger
	if debugMode {
		logger, _ = zap.NewDevelopment()
	} else {
		logger = zap.NewNop()
	}
	defer func() { _ = logger.Sync() }()

	// Collect files
	files, err := collectValidationFiles(args, validateRecursive)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no workflow files found")
	}

	// Create validator
	var v validator.Validator
	if validateRemote {
		// Server validation mode
		serverAddr := serverURL
		if serverAddr == "" {
			serverAddr = "http://localhost:8080"
		}

		client := &http.Client{
			Timeout: 30 * time.Second,
		}
		remoteValidator := validator.NewRemoteValidator(client, serverAddr, logger)

		// Try remote validation, fallback to local on network error
		v = &fallbackValidator{
			remote: remoteValidator,
			logger: logger,
		}
	} else {
		// Local validation mode
		dslValidator, err := dsl.NewValidator(logger)
		if err != nil {
			return fmt.Errorf("failed to create validator: %w", err)
		}
		v = validator.NewLocalValidator(dslValidator, logger)
	}

	// Validate files
	results := &validator.ValidationResults{
		Total: len(files),
	}

	if len(files) > 1 && validateFormat == "text" {
		fmt.Printf("Validating %d files...\n\n", len(files))
	}

	for _, file := range files {
		result, err := v.Validate(file)
		if err != nil {
			return fmt.Errorf("validation error for %s: %w", file, err)
		}

		results.Files = append(results.Files, result)

		if result.Valid {
			results.Passed++
		} else {
			results.Failed++
		}

		// Real-time progress for text format
		if validateFormat == "text" && len(files) > 1 && !validateVerbose {
			if result.Valid {
				fmt.Printf("✓ %s\n", file)
			} else {
				fmt.Printf("✗ %s\n", file)
			}
		}
	}

	// Output results
	if err := printValidationResults(results, validateFormat, validateVerbose); err != nil {
		return err
	}

	// Return exit code
	if results.HasErrors() {
		os.Exit(1)
	}

	return nil
}

func collectValidationFiles(args []string, recursive bool) ([]string, error) {
	var files []string

	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("file not found: %s", arg)
			}
			if os.IsPermission(err) {
				return nil, fmt.Errorf("permission denied: %s", arg)
			}
			return nil, fmt.Errorf("failed to access file: %w", err)
		}

		if info.IsDir() {
			if !recursive {
				return nil, fmt.Errorf("%s is a directory, use --recursive flag", arg)
			}
			dirFiles, err := scanDirectory(arg)
			if err != nil {
				return nil, err
			}
			files = append(files, dirFiles...)
		} else {
			if !isYAMLFile(arg) {
				return nil, fmt.Errorf("not a YAML file: %s", arg)
			}
			files = append(files, arg)
		}
	}

	return files, nil
}

func scanDirectory(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isYAMLFile(path) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	return files, nil
}

func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

// fallbackValidator tries remote validation first, falls back to local on network error
type fallbackValidator struct {
	remote *validator.RemoteValidator
	logger *zap.Logger
}

func (v *fallbackValidator) Validate(filepath string) (*validator.ValidationResult, error) {
	// Try remote validation
	result, err := v.remote.Validate(filepath)
	if err != nil {
		// Network error - fallback to local validation
		if strings.Contains(err.Error(), "failed to connect to server") {
			fmt.Fprintf(os.Stderr, "✗ Failed to connect to server\n")
			fmt.Fprintf(os.Stderr, "Error: %s\n\n", err.Error())
			fmt.Fprintf(os.Stderr, "Falling back to local validation...\n\n")

			// Create local validator
			dslValidator, localErr := dsl.NewValidator(v.logger)
			if localErr != nil {
				return nil, fmt.Errorf("failed to create local validator: %w", localErr)
			}

			localVal := validator.NewLocalValidator(dslValidator, v.logger)
			return localVal.Validate(filepath)
		}
		return nil, err
	}
	return result, nil
}

func printValidationResults(results *validator.ValidationResults, format string, verbose bool) error {
	switch format {
	case "json":
		return printValidationJSON(results)
	case "yaml":
		return printValidationYAML(results)
	default:
		return printValidationText(results, verbose)
	}
}

func printValidationText(results *validator.ValidationResults, verbose bool) error {
	if results.Total == 1 {
		result := results.Files[0]
		if result.Valid {
			if verbose {
				// Verbose mode - show detailed information
				fmt.Println("✓ Workflow is valid")
				fmt.Println()
				fmt.Println("Workflow Details:")
				fmt.Printf("  Name:    %s\n", result.WorkflowName)
				fmt.Printf("  Jobs:    %d\n", result.JobsCount)
				fmt.Printf("  Steps:   %d\n", result.StepsCount)
				if result.Remote {
					fmt.Println("  Verified: Server")
				}
				fmt.Println()
				fmt.Printf("Validation passed in %dms\n", result.ValidationTimeMS())
			} else {
				// Normal mode - compact output
				fmt.Println("✓ Workflow is valid")
				fmt.Println()
				fmt.Printf("Workflow: %s\n", result.WorkflowName)
				fmt.Printf("Jobs:     %d\n", result.JobsCount)
				fmt.Printf("Steps:    %d\n", result.StepsCount)
				if result.Remote {
					fmt.Println("Verified: Server")
				}
			}
		} else {
			fmt.Println("✗ Validation failed")
			fmt.Println()
			printValidationErrors(result)
		}
	} else {
		fmt.Println()
		fmt.Println("Summary:")
		fmt.Printf("  Total:   %d\n", results.Total)
		fmt.Printf("  Passed:  %d\n", results.Passed)
		fmt.Printf("  Failed:  %d\n", results.Failed)
	}

	return nil
}

func printValidationErrors(result *validator.ValidationResult) {
	fmt.Printf("File: %s\n", result.File)

	if len(result.Errors) == 1 {
		err := result.Errors[0]
		fmt.Printf("Error: %s\n\n", err.Type)
		if err.Line > 0 {
			fmt.Printf("  Line %d: %s\n", err.Line, err.Message)
		} else {
			fmt.Printf("  %s\n", err.Message)
		}
		if err.Suggestion != "" {
			fmt.Printf("\n  Suggestion: %s\n", err.Suggestion)
		}
	} else {
		fmt.Printf("Found %d validation errors:\n\n", len(result.Errors))
		for i, err := range result.Errors {
			fmt.Printf("  %d. ", i+1)
			if err.Line > 0 {
				fmt.Printf("Line %d: ", err.Line)
			}
			fmt.Printf("%s\n", err.Message)
			if err.Field != "" {
				fmt.Printf("     Field: %s\n", err.Field)
			}
			if err.Suggestion != "" {
				fmt.Printf("     Suggestion: %s\n", err.Suggestion)
			}
			fmt.Println()
		}
	}
}

func printValidationJSON(results *validator.ValidationResults) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if results.Total == 1 {
		return enc.Encode(results.Files[0])
	}

	return enc.Encode(results)
}

func printValidationYAML(results *validator.ValidationResults) error {
	enc := yaml.NewEncoder(os.Stdout)
	defer func() { _ = enc.Close() }()

	if results.Total == 1 {
		return enc.Encode(results.Files[0])
	}

	return enc.Encode(results)
}
