package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/output"
	"github.com/spf13/cobra"
)

var (
	statusWatch    bool
	statusInterval string
	statusCompact  bool
	statusFormat   string
	statusQuiet    bool
	statusNoColor  bool
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <workflow-id>",
		Short: "Get workflow execution status",
		Long: `Query the status of a workflow execution.

Displays workflow status, execution progress, jobs, and steps.
Use --watch to continuously monitor a running workflow.

Examples:
  # Get current status
  waterflow status 550e8400-e29b-41d4-a716-446655440000
  
  # Watch status updates
  waterflow status --watch 550e8400-e29b-41d4-a716-446655440000
  
  # Custom refresh interval
  waterflow status --watch --interval 5s <workflow-id>
  
  # Compact mode (no steps)
  waterflow status --compact <workflow-id>
  
  # JSON output for automation
  waterflow status --format json <workflow-id>
  
  # Quiet mode (only status value)
  STATUS=$(waterflow status --quiet <workflow-id>)`,
		Args: cobra.ExactArgs(1),
		RunE: runStatus,
	}

	cmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, "Watch status updates continuously")
	cmd.Flags().StringVar(&statusInterval, "interval", "2s", "Refresh interval for --watch (e.g., 2s, 5s)")
	cmd.Flags().BoolVar(&statusCompact, "compact", false, "Compact mode (hide steps)")
	cmd.Flags().StringVar(&statusFormat, "format", "text", "Output format (text, json, yaml)")
	cmd.Flags().BoolVarP(&statusQuiet, "quiet", "q", false, "Only output status value (for scripts)")
	cmd.Flags().BoolVar(&statusNoColor, "no-color", false, "Disable colored output")

	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	workflowID := args[0]

	// 验证工作流 ID 格式
	if err := validateWorkflowID(workflowID); err != nil {
		return err
	}

	// 确定最终输出格式（优先使用命令特定的格式）
	format := statusFormat
	if format == "text" && outputFormat != "" {
		format = outputFormat
	}

	// 确定超时时间
	timeout := 30 * time.Second

	// 创建 HTTP 客户端
	httpClient := client.New(
		serverURL,
		apiKey,
		timeout,
		debugMode,
	)

	// 检查颜色支持
	noColor := statusNoColor || os.Getenv("NO_COLOR") != "" || !isTerminal()

	// 创建输出格式化器
	formatter := output.NewFormatter(format, noColor)

	// 监控模式
	if statusWatch {
		return watchStatus(httpClient, workflowID, statusInterval, formatter)
	}

	// 一次性查询
	status, err := httpClient.GetWorkflowStatus(workflowID)
	if err != nil {
		return formatStatusErrorMessage(err, format)
	}

	// 输出结果
	if statusQuiet {
		fmt.Println(status.Status)
		return nil
	}

	if err := formatter.PrintWorkflowStatus(status, statusCompact); err != nil {
		return err
	}

	return nil
}

// validateWorkflowID 验证工作流 ID 格式
func validateWorkflowID(id string) error {
	if id == "" {
		return fmt.Errorf("workflow ID is required")
	}

	// UUID v4 格式验证 (友好提示,Server 端会进一步验证)
	if !uuidRegex.MatchString(id) {
		fmt.Fprintf(os.Stderr, "Warning: Workflow ID '%s' is not a valid UUID format\n", id)
		fmt.Fprintf(os.Stderr, "Expected format: 550e8400-e29b-41d4-a716-446655440000\n\n")
	}

	return nil
}

// watchStatus 持续监控工作流状态
func watchStatus(c *client.Client, workflowID string, intervalStr string, formatter *output.Formatter) error {
	// 解析刷新间隔
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return fmt.Errorf("invalid interval format: %s (expected: 2s, 5s, 1m)", intervalStr)
	}

	if interval < 1*time.Second {
		return fmt.Errorf("interval too short: %s (minimum: 1s)", intervalStr)
	}

	// 设置信号处理 (Ctrl+C 退出)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nInterrupted by user")
		cancel()
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 首次查询
	completed, err := displayStatus(c, workflowID, formatter, true, intervalStr)
	if err != nil {
		return err
	}
	if completed {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			// 清屏 (仅在 TTY 时)
			if isTerminal() {
				clearScreen()
			}

			// 查询并显示状态
			completed, err := displayStatus(c, workflowID, formatter, true, intervalStr)
			if err != nil {
				// 判断是否为致命错误
				if serverErr, ok := err.(*client.ServerError); ok {
					if serverErr.StatusCode == 404 {
						// 工作流不存在,终止
						return err
					}
				}
				// 临时错误,显示警告并继续
				fmt.Fprintf(os.Stderr, "⚠ Temporary error: %v (retrying...)\n", err)
				continue
			}

			// 工作流完成后退出
			if completed {
				return nil
			}
		}
	}
}

// displayStatus 查询并显示状态
func displayStatus(c *client.Client, workflowID string, formatter *output.Formatter, showRefreshHint bool, interval string) (bool, error) {
	status, err := c.GetWorkflowStatus(workflowID)
	if err != nil {
		return false, formatStatusErrorMessage(err, formatter.GetFormat())
	}

	if err := formatter.PrintWorkflowStatus(status, statusCompact); err != nil {
		return false, err
	}

	// 显示刷新提示
	if showRefreshHint && !isTerminalStatusState(status.Status) {
		fmt.Printf("\nRefreshing every %s... (Press Ctrl+C to stop)\n", interval)
	}

	// 判断是否完成
	return isTerminalStatusState(status.Status), nil
}

// clearScreen 清屏 (ANSI 转义码)
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// isTerminal 判断是否为终端
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// isTerminalStatusState 判断是否为终止状态
func isTerminalStatusState(status string) bool {
	return status == "completed" || status == "failed" || status == "cancelled" || status == "timeout"
}

// formatStatusErrorMessage 格式化状态查询错误
func formatStatusErrorMessage(err error, format string) error {
	if format == "json" {
		return formatStatusErrorJSON(err)
	}

	if serverErr, ok := err.(*client.ServerError); ok {
		return formatStatusServerError(serverErr)
	}

	// 其他错误
	return err
}

func formatStatusServerError(err *client.ServerError) error {
	fmt.Fprintf(os.Stderr, "Error: Workflow status query failed\n")
	fmt.Fprintf(os.Stderr, "  Status: %d\n\n", err.StatusCode)

	switch err.StatusCode {
	case 404:
		fmt.Fprintln(os.Stderr, "Workflow not found")
		if workflowID, ok := err.Details["workflow_id"].(string); ok {
			fmt.Fprintf(os.Stderr, "  Workflow ID: %s\n", workflowID)
		}
		fmt.Fprintln(os.Stderr, "\nSuggestion: Check workflow ID or use 'waterflow list' to see all workflows")

	case 401:
		fmt.Fprintf(os.Stderr, "Message: %s\n", err.Message)
		fmt.Fprintln(os.Stderr, "\nSuggestion: Set API key using --api-key flag or WATERFLOW_API_KEY env variable")

	default:
		fmt.Fprintf(os.Stderr, "Message: %s\n", err.Message)
		fmt.Fprintln(os.Stderr, "\nSuggestion: Check server logs for details or contact administrator")
	}

	os.Exit(1)
	return nil
}

func formatStatusErrorJSON(err error) error {
	if serverErr, ok := err.(*client.ServerError); ok {
		errorObj := map[string]interface{}{
			"error": map[string]interface{}{
				"code":    serverErr.Code,
				"message": serverErr.Message,
				"details": serverErr.Details,
			},
		}
		fmt.Println(output.ToJSON(errorObj))
	}
	os.Exit(1)
	return nil
}
