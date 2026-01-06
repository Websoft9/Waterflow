package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Global flags
	serverURL    string
	apiKey       string
	debugMode    bool
	configFile   string
	outputFormat string

	// Version information (set by main.go)
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "waterflow",
	Short: "Waterflow - Declarative workflow orchestration engine",
	Long: `Waterflow CLI provides command-line interface for workflow management.

The CLI tool allows you to:
  - Validate workflow YAML syntax
  - Submit workflows to Waterflow server
  - Query workflow status and execution history
  - Retrieve workflow logs
  - Manage workflow nodes

For more information, visit: https://github.com/Websoft9/waterflow`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle --version flag
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			fmt.Printf("Waterflow CLI %s\n", version)
			return
		}
		// No subcommand provided, show help
		_ = cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		if err := initConfig(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Merge viper config with flags (flags take precedence)
		if !cmd.Flags().Changed("server") && viper.IsSet("server") {
			serverURL = viper.GetString("server")
		}
		if !cmd.Flags().Changed("api-key") && viper.IsSet("api_key") {
			apiKey = viper.GetString("api_key")
		}
		if !cmd.Flags().Changed("debug") && viper.IsSet("debug") {
			debugMode = viper.GetBool("debug")
		}
		if !cmd.Flags().Changed("output-format") && viper.IsSet("output_format") {
			outputFormat = viper.GetString("output_format")
		}

		if debugMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Server URL: %s\n", serverURL)
			fmt.Fprintf(os.Stderr, "DEBUG: Debug mode: %v\n", debugMode)
			fmt.Fprintf(os.Stderr, "DEBUG: Output format: %s\n", outputFormat)
		}

		return nil
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// SetVersionInfo sets version information from main.go
func SetVersionInfo(v, c, t string) {
	version = v
	commit = c
	buildTime = t
}

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "", "Waterflow server URL (env: WATERFLOW_SERVER)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication (env: WATERFLOW_API_KEY)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (default: ~/.waterflow/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output-format", "", "Output format: text, json, or yaml (default: text)")

	// --version flag (handled in PersistentPreRunE)
	rootCmd.Flags().BoolP("version", "v", false, "Show version information")

	// Bind environment variables
	_ = viper.BindEnv("server", "WATERFLOW_SERVER")
	_ = viper.BindEnv("api_key", "WATERFLOW_API_KEY")

	// Register subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newLogsCmd())
	rootCmd.AddCommand(newNodeCmd())
}

// initConfig loads configuration from file
func initConfig() error {
	// Determine config file path
	cfgFile := configFile
	if cfgFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		cfgFile = filepath.Join(home, ".waterflow", "config.yaml")
	}

	// Set config file
	viper.SetConfigFile(cfgFile)
	viper.SetConfigType("yaml")

	// Set defaults (following Story 1.1 configuration pattern)
	viper.SetDefault("server", "http://localhost:8080")
	viper.SetDefault("debug", false)
	viper.SetDefault("timeout", 30)
	viper.SetDefault("output_format", "text")

	// Read config file (ignore file not found error)
	if err := viper.ReadInConfig(); err != nil {
		// Ignore file not found errors - use defaults
		if os.IsNotExist(err) {
			if debugMode {
				fmt.Fprintf(os.Stderr, "DEBUG: Config file not found: %s (using defaults)\n", cfgFile)
			}
		} else if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if debugMode {
				fmt.Fprintf(os.Stderr, "DEBUG: Config file not found: %s (using defaults)\n", cfgFile)
			}
		} else {
			// Config file exists but failed to read (e.g., invalid YAML)
			return fmt.Errorf("failed to read config file %s: %w", cfgFile, err)
		}
	} else {
		if debugMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Loaded config from: %s\n", cfgFile)
		}
	}

	return nil
}

// GetServerURL returns the configured server URL
func GetServerURL() string {
	return serverURL
}

// GetAPIKey returns the configured API key
func GetAPIKey() string {
	return apiKey
}

// IsDebugMode returns whether debug mode is enabled
func IsDebugMode() bool {
	return debugMode
}

// GetOutputFormat returns the configured output format
func GetOutputFormat() string {
	if outputFormat == "" {
		return "text"
	}
	return outputFormat
}
