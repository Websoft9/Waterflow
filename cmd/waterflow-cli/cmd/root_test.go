package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRootCmdStructure tests root command is properly set up
func TestRootCmdStructure(t *testing.T) {
	require.NotNil(t, rootCmd)
	assert.Equal(t, "waterflow", rootCmd.Use)
	assert.Contains(t, rootCmd.Short, "Waterflow")
}

// TestGetServerURL tests GetServerURL function
func TestGetServerURL(t *testing.T) {
	// Save and restore
	old := serverURL
	defer func() { serverURL = old }()

	serverURL = "http://test-server:8080"
	assert.Equal(t, "http://test-server:8080", GetServerURL())

	serverURL = ""
	assert.Equal(t, "", GetServerURL())
}

// TestGetAPIKey tests GetAPIKey function
func TestGetAPIKey(t *testing.T) {
	// Save and restore
	old := apiKey
	defer func() { apiKey = old }()

	apiKey = "test-api-key"
	assert.Equal(t, "test-api-key", GetAPIKey())

	apiKey = ""
	assert.Equal(t, "", GetAPIKey())
}

// TestIsDebugMode tests IsDebugMode function
func TestIsDebugMode(t *testing.T) {
	// Save and restore
	old := debugMode
	defer func() { debugMode = old }()

	debugMode = true
	assert.True(t, IsDebugMode())

	debugMode = false
	assert.False(t, IsDebugMode())
}

// TestGetOutputFormat tests GetOutputFormat function
func TestGetOutputFormat(t *testing.T) {
	// Save and restore
	old := outputFormat
	defer func() { outputFormat = old }()

	outputFormat = "json"
	assert.Equal(t, "json", GetOutputFormat())

	outputFormat = "yaml"
	assert.Equal(t, "yaml", GetOutputFormat())

	// Default case
	outputFormat = ""
	assert.Equal(t, "text", GetOutputFormat())
}

// TestInitConfig tests config initialization
func TestInitConfig(t *testing.T) {
	// Test with non-existent config file (should not error, uses defaults)
	configFile = "/nonexistent/path/config.yaml"
	debugMode = false

	err := initConfig()
	assert.NoError(t, err)

	// Reset
	configFile = ""
}

// TestInitConfigWithInvalidPath tests config with invalid home directory scenario
func TestInitConfigWithInvalidPath(t *testing.T) {
	// Test default config loading - should use defaults if file doesn't exist
	oldConfigFile := configFile
	defer func() { configFile = oldConfigFile }()

	configFile = ""
	err := initConfig()
	// Should succeed (uses defaults when config not found)
	assert.NoError(t, err)
}

// TestExecute tests Execute function (basic structure)
func TestExecute(t *testing.T) {
	// We can't fully test Execute without it parsing os.Args,
	// but we can verify the function exists and rootCmd is set up
	require.NotNil(t, rootCmd)
	assert.NotNil(t, rootCmd.Execute)
}

// TestRootCmdFlags tests that required flags are defined
func TestRootCmdFlags(t *testing.T) {
	// Check persistent flags
	flags := rootCmd.PersistentFlags()

	serverFlag := flags.Lookup("server")
	require.NotNil(t, serverFlag)

	apiKeyFlag := flags.Lookup("api-key")
	require.NotNil(t, apiKeyFlag)

	debugFlag := flags.Lookup("debug")
	require.NotNil(t, debugFlag)

	configFlag := flags.Lookup("config")
	require.NotNil(t, configFlag)

	outputFormatFlag := flags.Lookup("output-format")
	require.NotNil(t, outputFormatFlag)
}

// TestRootCmdSubcommands tests that subcommands are registered
func TestRootCmdSubcommands(t *testing.T) {
	commands := rootCmd.Commands()
	require.NotEmpty(t, commands)

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	// Check expected subcommands exist
	assert.True(t, commandNames["version"], "version command should exist")
	assert.True(t, commandNames["validate"], "validate command should exist")
	assert.True(t, commandNames["status"], "status command should exist")
	assert.True(t, commandNames["logs"], "logs command should exist")
	assert.True(t, commandNames["node"], "node command should exist")
}
