package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewVersionCmd tests version command creation
func TestNewVersionCmd(t *testing.T) {
	cmd := newVersionCmd()
	require.NotNil(t, cmd)

	assert.Equal(t, "version", cmd.Use)
	assert.Contains(t, cmd.Short, "version")
}

// TestRunVersion tests version output
func TestRunVersion(t *testing.T) {
	// Set test version info
	oldVersion := version
	oldCommit := commit
	oldBuildTime := buildTime

	version = "v1.0.0-test"
	commit = "abc123"
	buildTime = "2024-01-01"

	defer func() {
		version = oldVersion
		commit = oldCommit
		buildTime = oldBuildTime
	}()

	cmd := newVersionCmd()

	// Should not panic
	runVersion(cmd, []string{})
}

// TestSetVersionInfo tests setting version info
func TestSetVersionInfo(t *testing.T) {
	// Save original values
	oldVersion := version
	oldCommit := commit
	oldBuildTime := buildTime

	defer func() {
		version = oldVersion
		commit = oldCommit
		buildTime = oldBuildTime
	}()

	SetVersionInfo("v2.0.0", "def456", "2024-06-01")

	assert.Equal(t, "v2.0.0", version)
	assert.Equal(t, "def456", commit)
	assert.Equal(t, "2024-06-01", buildTime)
}
