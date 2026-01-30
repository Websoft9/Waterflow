//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// E2E test environment configuration
var (
	serverURL   string
	temporalURL string
)

// TestMain sets up the E2E test environment
func TestMain(m *testing.M) {
	// Load environment variables
	serverURL = getEnv("SERVER_URL", "http://localhost:18080")
	temporalURL = getEnv("TEMPORAL_HOST", "localhost:17233")

	fmt.Printf("\n===========================================\n")
	fmt.Printf("E2E Test Environment Configuration\n")
	fmt.Printf("===========================================\n")
	fmt.Printf("Server URL:   %s\n", serverURL)
	fmt.Printf("Temporal URL: %s\n", temporalURL)
	fmt.Printf("===========================================\n\n")

	// Wait for services to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("Waiting for Waterflow Server to be ready...")
	if err := waitForServer(ctx, serverURL); err != nil {
		fmt.Printf("ERROR: Server not ready: %v\n", err)
		fmt.Println("Hint: Run 'docker compose -f test/e2e/docker-compose.e2e.yaml up -d'")
		os.Exit(1)
	}
	fmt.Println("✓ Server is ready")

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// waitForServer waits for the Waterflow server to be ready
func waitForServer(ctx context.Context, url string) error {
	healthURL := fmt.Sprintf("%s/health", url)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server at %s", url)
		case <-ticker.C:
			resp, err := http.Get(healthURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvOrDefault is an alias for getEnv (for compatibility)
func getEnvOrDefault(key, defaultValue string) string {
	return getEnv(key, defaultValue)
}
