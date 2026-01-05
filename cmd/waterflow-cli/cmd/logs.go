package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	logsFollow       bool
	logsTail         int
	logsLevel        string
	logsJob          string
	logsStep         string
	logsFormat       string
	logsNoColor      bool
	logsNoTimestamps bool
	logsPollInterval string
)

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <workflow-id>",
		Short: "View workflow execution logs",
		Long: `View execution logs from a workflow.

Displays logs from all jobs and steps in chronological order.
Use --follow to stream logs in real-time.

Examples:
  # View logs
  waterflow logs 550e8400-e29b-41d4-a716-446655440000
  
  # Follow logs in real-time
  waterflow logs --follow <workflow-id>
  
  # Show only errors
  waterflow logs --level error <workflow-id>
  
  # Filter by job
  waterflow logs --job deploy <workflow-id>
  
  # Filter by step
  waterflow logs --step checkout <workflow-id>
  
  # Combine filters
  waterflow logs --job deploy --level error <workflow-id>
  
  # Show last 50 lines
  waterflow logs --tail 50 <workflow-id>
  
  # JSON output
  waterflow logs --format json <workflow-id>`,
		Args: cobra.ExactArgs(1),
		RunE: runLogs,
	}

	cmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow logs in real-time (like tail -f)")
	cmd.Flags().IntVar(&logsTail, "tail", 100, "Number of lines to show from the end (1-1000, -1 for all)")
	cmd.Flags().StringVar(&logsLevel, "level", "", "Filter by log level (info,warn,error,debug)")
	cmd.Flags().StringVar(&logsJob, "job", "", "Filter by job name")
	cmd.Flags().StringVar(&logsStep, "step", "", "Filter by step name")
	cmd.Flags().StringVar(&logsFormat, "format", "text", "Output format (text, json, compact)")
	cmd.Flags().BoolVar(&logsNoColor, "no-color", false, "Disable colored output")
	cmd.Flags().BoolVar(&logsNoTimestamps, "no-timestamps", false, "Hide timestamps")
	cmd.Flags().StringVar(&logsPollInterval, "poll-interval", "2s", "Polling interval for --follow")

	return cmd
}

func runLogs(cmd *cobra.Command, args []string) error {
	workflowID := args[0]

	// 验证参数
	if err := validateLogsParams(); err != nil {
		return err
	}

	if logsTail < -1 || logsTail == 0 || logsTail > 1000 {
		return fmt.Errorf("invalid --tail value: %d (must be -1 or 1-1000)", logsTail)
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
	noColor := logsNoColor || os.Getenv("NO_COLOR") != "" || !isTerminal()

	// 创建日志格式化器
	formatter := newLogsFormatter(logsFormat, noColor, logsNoTimestamps)

	// 跟踪模式
	if logsFollow {
		return followLogs(httpClient, workflowID, formatter)
	}

	// 一次性查询
	logs, err := httpClient.GetWorkflowLogs(workflowID, buildLogsQuery())
	if err != nil {
		return formatLogsError(err, logsFormat)
	}

	// 输出日志
	if len(logs) == 0 {
		return displayNoLogsMessage()
	}

	for _, log := range logs {
		if err := formatter.PrintLog(&log); err != nil {
			return err
		}
	}

	return nil
}

// validateLogsParams 验证日志参数
func validateLogsParams() error {
	if logsLevel != "" {
		levels := strings.Split(logsLevel, ",")
		validLevels := map[string]bool{
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
		}

		for _, level := range levels {
			level = strings.TrimSpace(strings.ToLower(level))
			if !validLevels[level] {
				return fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error)", level)
			}
		}
	}
	return nil
}

// buildLogsQuery 构建查询参数
func buildLogsQuery() client.LogsQuery {
	tail := logsTail
	if tail == -1 {
		tail = 1000 // API 最大值
	}

	return client.LogsQuery{
		Tail:  tail,
		Level: logsLevel,
		Job:   logsJob,
		Step:  logsStep,
	}
}

// followLogs 实时跟踪日志 (SSE 优先,降级为轮询)
func followLogs(c *client.Client, workflowID string, formatter *LogsFormatter) error {
	// 尝试 SSE 流式获取 (优先方案)
	logChan, errChan, err := c.StreamWorkflowLogs(workflowID, buildLogsQuery())
	if err == nil {
		// SSE 可用,使用流式跟踪
		return followLogsWithSSE(c, workflowID, logChan, errChan, formatter)
	}

	// SSE 不可用,降级为轮询模式
	if logsFormat != "json" {
		fmt.Fprintf(os.Stderr, "WARN: SSE not available, using polling mode\n")
	}
	return followLogsWithPolling(c, workflowID, formatter)
}

// followLogsWithSSE 使用 SSE 流式跟踪日志
func followLogsWithSSE(c *client.Client, workflowID string, logChan <-chan client.LogEntry, errChan <-chan error, formatter *LogsFormatter) error {
	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nInterrupted by user")
		cancel()
	}()

	// 定期检查工作流状态
	statusTicker := time.NewTicker(5 * time.Second)
	defer statusTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case log, ok := <-logChan:
			if !ok {
				// SSE stream 关闭,检查工作流状态
				return checkWorkflowCompletion(c, workflowID)
			}
			if err := formatter.PrintLog(&log); err != nil {
				return err
			}

		case err := <-errChan:
			if err != nil {
				return fmt.Errorf("SSE stream error: %w", err)
			}

		case <-statusTicker.C:
			// 定期检查工作流是否完成
			status, err := c.GetWorkflowStatus(workflowID)
			if err == nil && isTerminalStatusState(status.Status) {
				// 等待剩余日志
				time.Sleep(1 * time.Second)
				return checkWorkflowCompletion(c, workflowID)
			}
		}
	}
}

// followLogsWithPolling 轮询模式跟踪日志 (降级方案)
func followLogsWithPolling(c *client.Client, workflowID string, formatter *LogsFormatter) error {
	// 解析轮询间隔
	interval, err := time.ParseDuration(logsPollInterval)
	if err != nil {
		return fmt.Errorf("invalid poll-interval format: %s (expected: 2s, 5s, 1m)", logsPollInterval)
	}

	if interval < 1*time.Second {
		return fmt.Errorf("poll-interval too short: %s (minimum: 1s)", logsPollInterval)
	}

	// 设置信号处理 (Ctrl+C 退出)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\n\nInterrupted by user")
		cancel()
	}()

	// 记录已显示的日志条目数
	lastCount := 0

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 首次查询
	logs, err := c.GetWorkflowLogs(workflowID, buildLogsQuery())
	if err != nil {
		return formatLogsError(err, logsFormat)
	}

	for _, log := range logs {
		if err := formatter.PrintLog(&log); err != nil {
			return err
		}
	}
	lastCount = len(logs)

	// 轮询新日志
	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			// 查询状态判断是否完成
			status, err := c.GetWorkflowStatus(workflowID)
			if err != nil {
				// 忽略状态查询错误,继续尝试获取日志
				continue
			}

			// 获取所有日志(不使用tail限制,获取新增的)
			query := buildLogsQuery()
			query.Tail = 1000 // 获取更多日志
			logs, err := c.GetWorkflowLogs(workflowID, query)
			if err != nil {
				continue
			}

			// 只显示新日志
			if len(logs) > lastCount {
				for i := lastCount; i < len(logs); i++ {
					if err := formatter.PrintLog(&logs[i]); err != nil {
						return err
					}
				}
				lastCount = len(logs)
			}

			// 工作流完成后退出
			if isTerminalStatusState(status.Status) {
				// 根据工作流状态返回退出码
				if status.Status == "failed" {
					os.Exit(1)
				}
				return nil
			}
		}
	}
}

// checkWorkflowCompletion 检查工作流完成状态并显示消息
func checkWorkflowCompletion(c *client.Client, workflowID string) error {
	status, err := c.GetWorkflowStatus(workflowID)
	if err != nil {
		return err
	}

	if logsFormat == "json" {
		return nil // JSON 模式不显示完成消息
	}

	fmt.Fprintln(os.Stderr, "")
	if status.Status == "completed" {
		fmt.Fprintf(os.Stderr, "✓ Workflow completed successfully\n")
		return nil
	} else {
		fmt.Fprintf(os.Stderr, "✗ Workflow %s\n", status.Status)
		os.Exit(1)
		return nil
	}
}

// displayNoLogsMessage 显示无日志提示
func displayNoLogsMessage() error {
	fmt.Println("No logs found matching filter criteria")
	fmt.Println()

	filters := []string{}
	if logsLevel != "" {
		filters = append(filters, fmt.Sprintf("  Level: %s", logsLevel))
	}
	if logsJob != "" {
		filters = append(filters, fmt.Sprintf("  Job: %s", logsJob))
	}
	if logsStep != "" {
		filters = append(filters, fmt.Sprintf("  Step: %s", logsStep))
	}

	if len(filters) > 0 {
		fmt.Println("Filters applied:")
		for _, f := range filters {
			fmt.Println(f)
		}
		fmt.Println()
		fmt.Println("Suggestion: Try removing filters or check workflow status")
	}

	return nil
}

// formatLogsError 格式化日志查询错误
func formatLogsError(err error, format string) error {
	if format == "json" {
		return formatLogsErrorJSON(err)
	}

	if serverErr, ok := err.(*client.ServerError); ok {
		return formatLogsServerError(serverErr)
	}

	// 其他错误
	return err
}

func formatLogsServerError(err *client.ServerError) error {
	fmt.Fprintf(os.Stderr, "Error: Workflow logs query failed\n")
	fmt.Fprintf(os.Stderr, "  Status: %d\n\n", err.StatusCode)

	switch err.StatusCode {
	case 404:
		fmt.Fprintln(os.Stderr, "Workflow not found")
		if workflowID, ok := err.Details["workflow_id"].(string); ok {
			fmt.Fprintf(os.Stderr, "  Workflow ID: %s\n", workflowID)
		}
		fmt.Fprintln(os.Stderr, "\nSuggestion: Check workflow ID or use 'waterflow status' to verify workflow exists")

	case 400:
		fmt.Fprintf(os.Stderr, "Invalid parameter\n")
		if details, ok := err.Details["field"].(string); ok {
			fmt.Fprintf(os.Stderr, "  Parameter: %s\n", details)
		}
		if value, ok := err.Details["value"]; ok {
			fmt.Fprintf(os.Stderr, "  Value: %v\n", value)
		}
		if reason, ok := err.Details["reason"].(string); ok {
			fmt.Fprintf(os.Stderr, "  Reason: %s\n", reason)
		}

	default:
		fmt.Fprintf(os.Stderr, "Message: %s\n", err.Message)
		fmt.Fprintln(os.Stderr, "\nSuggestion: Check server logs for details or contact administrator")
	}

	os.Exit(1)
	return nil
}

func formatLogsErrorJSON(err error) error {
	if serverErr, ok := err.(*client.ServerError); ok {
		errorObj := map[string]interface{}{
			"error": map[string]interface{}{
				"code":    serverErr.Code,
				"message": serverErr.Message,
				"details": serverErr.Details,
			},
		}
		fmt.Println(toJSON(errorObj))
	}
	os.Exit(1)
	return nil
}

// LogsFormatter 日志格式化器
type LogsFormatter struct {
	format       string
	noColor      bool
	noTimestamps bool
}

func newLogsFormatter(format string, noColor, noTimestamps bool) *LogsFormatter {
	return &LogsFormatter{
		format:       format,
		noColor:      noColor,
		noTimestamps: noTimestamps,
	}
}

// PrintLog 打印单条日志
func (f *LogsFormatter) PrintLog(log *client.LogEntry) error {
	switch f.format {
	case "json":
		return f.printJSON(log)
	case "compact":
		return f.printCompact(log)
	default:
		return f.printText(log)
	}
}

func (f *LogsFormatter) printText(log *client.LogEntry) error {
	var parts []string

	// 时间戳
	if !f.noTimestamps {
		timestamp := formatTimestamp(log.Timestamp)
		parts = append(parts, fmt.Sprintf("[%s]", timestamp))
	}

	// 级别(带颜色)
	levelColor := f.getLevelColor(log.Level)
	parts = append(parts, levelColor.Sprintf("%-5s", strings.ToUpper(log.Level)))

	// Job.Step 或 Job
	if log.Job != "" {
		if log.Step != "" {
			parts = append(parts, fmt.Sprintf("[%s.%s]", log.Job, log.Step))
		} else {
			parts = append(parts, fmt.Sprintf("[%s]", log.Job))
		}
	}

	// 消息
	parts = append(parts, log.Message)

	// 错误信息
	if log.Error != "" {
		errorColor := color.New(color.FgRed)
		if f.noColor {
			errorColor = color.New()
		}
		parts = append(parts, errorColor.Sprintf("Error: %s", log.Error))
	}

	fmt.Println(strings.Join(parts, " "))
	return nil
}

func (f *LogsFormatter) printJSON(log *client.LogEntry) error {
	fmt.Println(toJSON(log))
	return nil
}

func (f *LogsFormatter) printCompact(log *client.LogEntry) error {
	var parts []string

	// Job.Step 或 Job
	if log.Job != "" {
		if log.Step != "" {
			parts = append(parts, fmt.Sprintf("[%s.%s]", log.Job, log.Step))
		} else {
			parts = append(parts, fmt.Sprintf("[%s]", log.Job))
		}
	}

	parts = append(parts, log.Message)

	if log.Error != "" {
		parts = append(parts, fmt.Sprintf("Error: %s", log.Error))
	}

	fmt.Println(strings.Join(parts, " "))
	return nil
}

func (f *LogsFormatter) getLevelColor(level string) *color.Color {
	if f.noColor {
		return color.New()
	}

	switch strings.ToLower(level) {
	case "error":
		return color.New(color.FgRed)
	case "warn", "warning":
		return color.New(color.FgYellow)
	case "info":
		return color.New(color.FgBlue)
	case "debug":
		return color.New(color.FgHiBlack)
	default:
		return color.New()
	}
}

func toJSON(data interface{}) string {
	b, _ := json.Marshal(data)
	return string(b)
}

// formatTimestamp 格式化时间戳 (ISO 8601 → 本地时间)
func formatTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		// Fallback: 简单字符串处理
		if len(ts) > 19 {
			ts = ts[:19]
		}
		return strings.Replace(ts, "T", " ", 1)
	}

	// 格式: 2006-01-02 15:04:05
	return t.Local().Format("2006-01-02 15:04:05")
}
