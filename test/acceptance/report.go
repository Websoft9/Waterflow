//go:build acceptance

package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReportGenerator generates acceptance test reports
type ReportGenerator struct {
	OutputDir   string
	ProjectName string
	Version     string
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(outputDir string) *ReportGenerator {
	return &ReportGenerator{
		OutputDir:   outputDir,
		ProjectName: "Waterflow",
		Version:     getVersion(),
	}
}

// getVersion returns the current version
func getVersion() string {
	// Try to get from environment or default
	if v := os.Getenv("VERSION"); v != "" {
		return v
	}
	return "dev"
}

// GenerateReport creates a Markdown report from scenario results
func (r *ReportGenerator) GenerateReport(results []ScenarioResult) (string, error) {
	if err := os.MkdirAll(r.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("acceptance-report-%s.md", timestamp)
	filepath := filepath.Join(r.OutputDir, filename)

	report := r.buildReport(results)

	if err := os.WriteFile(filepath, []byte(report), 0644); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}

	return filepath, nil
}

// buildReport constructs the Markdown report content
func (r *ReportGenerator) buildReport(results []ScenarioResult) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Waterflow Acceptance Test Report\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s\n", time.Now().UTC().Format("2006-01-02 15:04:05 UTC")))
	sb.WriteString(fmt.Sprintf("**Version:** %s\n", r.Version))
	sb.WriteString("**Environment:** Docker Compose (Acceptance)\n\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Scenario | Status | Duration |\n")
	sb.WriteString("|----------|--------|----------|\n")

	passed := 0
	failed := 0
	var totalDuration time.Duration

	for _, result := range results {
		statusIcon := "✅ PASSED"
		if result.Status == "failed" {
			statusIcon = "❌ FAILED"
			failed++
		} else if result.Status == "skipped" {
			statusIcon = "⏭️ SKIPPED"
		} else {
			passed++
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
			result.Name, statusIcon, formatDuration(result.Duration)))
		totalDuration += result.Duration
	}

	sb.WriteString("\n")
	percentage := 0
	if len(results) > 0 {
		percentage = (passed * 100) / len(results)
	}
	sb.WriteString(fmt.Sprintf("**Total:** %d/%d passed (%d%%)\n",
		passed, len(results), percentage))
	sb.WriteString(fmt.Sprintf("**Total Duration:** %s\n\n", formatDuration(totalDuration)))

	// Scenario Details
	sb.WriteString("## Scenario Details\n\n")

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, result.Name))

		statusText := "✅ PASSED"
		if result.Status == "failed" {
			statusText = "❌ FAILED"
		} else if result.Status == "skipped" {
			statusText = "⏭️ SKIPPED"
		}

		sb.WriteString(fmt.Sprintf("- **Status:** %s\n", statusText))
		sb.WriteString(fmt.Sprintf("- **Duration:** %s\n", formatDuration(result.Duration)))

		if result.WorkflowID != "" {
			sb.WriteString(fmt.Sprintf("- **Workflow ID:** %s\n", result.WorkflowID))
		}

		// Add details
		if len(result.Details) > 0 {
			for key, value := range result.Details {
				sb.WriteString(fmt.Sprintf("- **%s:** %v\n", key, value))
			}
		}

		// Add error if failed
		if result.Error != nil {
			sb.WriteString(fmt.Sprintf("\n**Error:**\n```\n%s\n```\n", result.Error.Error()))
		}

		// Add logs if available
		if result.Logs != "" {
			sb.WriteString(fmt.Sprintf("\n**Logs:**\n```\n%s\n```\n", truncateLogs(result.Logs, 2000)))
		}

		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("## Environment Information\n\n")
	sb.WriteString("- **Server URL:** http://localhost:18080\n")
	sb.WriteString("- **Temporal:** localhost:17233\n")
	sb.WriteString("- **Agents:** 3 (web-1, web-2, db-1)\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*Report generated at %s*\n", time.Now().UTC().Format(time.RFC3339)))

	return sb.String()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

// truncateLogs truncates logs to a maximum length
func truncateLogs(logs string, maxLen int) string {
	if len(logs) <= maxLen {
		return logs
	}
	return logs[:maxLen] + "\n... (truncated)"
}

// PrintSummary prints a summary to stdout
func PrintSummary(results []ScenarioResult) {
	fmt.Println("\n========================================")
	fmt.Println("Acceptance Test Summary")
	fmt.Println("========================================")

	passed := 0
	failed := 0

	for _, result := range results {
		status := "✅ PASSED"
		if result.Status == "failed" {
			status = "❌ FAILED"
			failed++
		} else if result.Status == "skipped" {
			status = "⏭️ SKIPPED"
		} else {
			passed++
		}

		fmt.Printf("  %s: %s (%s)\n", result.Name, status, formatDuration(result.Duration))
	}

	fmt.Println("----------------------------------------")
	fmt.Printf("Total: %d passed, %d failed\n", passed, failed)
	fmt.Println("========================================")
}
