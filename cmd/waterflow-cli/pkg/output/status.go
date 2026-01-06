package output

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Websoft9/waterflow/cmd/waterflow-cli/pkg/client"
	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

// PrintWorkflowStatus prints workflow status in configured format
func (f *Formatter) PrintWorkflowStatus(status *client.WorkflowStatus, compact bool) error {
	switch f.format {
	case "json":
		return f.printStatusJSON(status)
	case "yaml":
		return f.printStatusYAML(status)
	default:
		return f.printStatusText(status, compact)
	}
}

func (f *Formatter) printStatusText(status *client.WorkflowStatus, compact bool) error {
	// 标题
	fmt.Printf("Workflow: %s\n", status.Name)
	fmt.Printf("ID:       %s\n", status.ID)
	if status.RunID != "" {
		fmt.Printf("Run ID:   %s\n", status.RunID)
	}
	fmt.Println()

	// 状态
	statusColor := f.getStatusColor(status.Status)
	fmt.Printf("Status:      %s\n", statusColor.Sprint(status.Status))

	// 失败时显示错误信息
	if status.Error != "" {
		errorColor := color.New(color.FgRed)
		if f.noColor {
			errorColor = color.New()
		}
		fmt.Printf("Error:       %s\n", errorColor.Sprint(status.Error))
	}

	// 时间信息
	if status.CreatedAt != "" {
		fmt.Printf("Created:     %s\n", status.CreatedAt)
	}
	if status.StartedAt != "" {
		fmt.Printf("Started:     %s\n", status.StartedAt)
	}
	if status.CompletedAt != "" {
		fmt.Printf("Completed:   %s\n", status.CompletedAt)
	}
	if status.DurationSeconds != nil {
		fmt.Printf("Duration:    %s\n", formatDuration(*status.DurationSeconds))
	}

	// Jobs
	if len(status.Jobs) > 0 {
		fmt.Println()
		fmt.Println("Jobs:")
		for _, job := range status.Jobs {
			f.printJob(job, compact)
		}

		if !compact {
			fmt.Println()
			f.printLegend()
		}
	}

	// 失败提示
	if status.Status == "failed" {
		fmt.Printf("\nView logs:   waterflow logs %s\n", status.ID)
	}

	return nil
}

func (f *Formatter) printJob(job client.JobStatus, compact bool) {
	symbol := f.getStatusSymbol(job.Status)
	statusColor := f.getStatusColor(job.Status)

	duration := ""
	if job.StartedAt != "" && job.CompletedAt != "" {
		d := calculateDuration(job.StartedAt, job.CompletedAt)
		duration = fmt.Sprintf("  (%s)", formatDuration(d))
	} else if job.StartedAt != "" && job.Status == "running" {
		d := calculateDurationSince(job.StartedAt)
		duration = fmt.Sprintf("  (%s)", formatDuration(d))
	}

	fmt.Printf("  %s %-20s", symbol, truncateName(job.Name, 20))
	fmt.Printf("%s", statusColor.Sprintf("%-10s", job.Status))
	fmt.Printf("%s\n", duration)

	// Steps (仅在非紧凑模式下)
	if !compact && len(job.Steps) > 0 {
		for _, step := range job.Steps {
			f.printStep(step)
		}
		fmt.Println()
	}
}

func (f *Formatter) printStep(step client.StepStatus) {
	symbol := f.getStatusSymbol(step.Status)
	statusColor := f.getStatusColor(step.Status)

	fmt.Printf("    %s %-13s", symbol, step.Name)
	fmt.Printf("%s\n", statusColor.Sprint(step.Status))
}

func (f *Formatter) getStatusSymbol(status string) string {
	if f.noColor {
		// 无颜色模式使用简单字符
		switch status {
		case "completed":
			return "[✓]"
		case "running":
			return "[→]"
		case "failed":
			return "[✗]"
		case "cancelled", "timeout":
			return "[⊗]"
		default:
			return "[○]"
		}
	}

	// 带颜色的符号
	switch status {
	case "completed":
		return color.GreenString("✓")
	case "running":
		return color.BlueString("→")
	case "failed":
		return color.RedString("✗")
	case "cancelled", "timeout":
		return color.YellowString("⊗")
	default:
		return color.New(color.FgHiBlack).Sprint("○")
	}
}

func (f *Formatter) getStatusColor(status string) *color.Color {
	if f.noColor {
		return color.New()
	}

	switch status {
	case "completed":
		return color.New(color.FgGreen)
	case "running":
		return color.New(color.FgBlue)
	case "failed":
		return color.New(color.FgRed)
	case "cancelled", "timeout":
		return color.New(color.FgYellow)
	default:
		return color.New(color.FgHiBlack)
	}
}

func (f *Formatter) printLegend() {
	if f.noColor {
		fmt.Println("Legend: [✓] completed  [→] running  [✗] failed  [○] pending  [⊗] cancelled/timeout")
	} else {
		fmt.Printf("Legend: %s completed  %s running  %s failed  %s pending  %s cancelled/timeout\n",
			color.GreenString("✓"),
			color.BlueString("→"),
			color.RedString("✗"),
			color.New(color.FgHiBlack).Sprint("○"),
			color.YellowString("⊗"),
		)
	}
}

// formatDuration 格式化持续时间 (秒 → 人类可读)
func formatDuration(seconds int) string {
	d := time.Duration(seconds) * time.Second

	if d < time.Minute {
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := int(d.Minutes())
	remainingSeconds := seconds - (minutes * 60)

	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", minutes, remainingSeconds)
	}

	hours := int(d.Hours())
	remainingMinutes := minutes - (hours * 60)
	return fmt.Sprintf("%dh %dm", hours, remainingMinutes)
}

// truncateName 截断过长的名称并添加省略号
func truncateName(name string, maxLen int) string {
	if len(name) <= maxLen {
		return name
	}
	if maxLen <= 3 {
		return name[:maxLen]
	}
	return name[:maxLen-3] + "..."
}

// calculateDuration 计算两个时间戳之间的持续时间
func calculateDuration(startStr, endStr string) int {
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return 0
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return 0
	}
	return int(end.Sub(start).Seconds())
}

// calculateDurationSince 计算从开始到现在的持续时间
func calculateDurationSince(startStr string) int {
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return 0
	}
	return int(time.Since(start).Seconds())
}

func (f *Formatter) printStatusJSON(status *client.WorkflowStatus) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(status)
}

func (f *Formatter) printStatusYAML(status *client.WorkflowStatus) error {
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	defer func() { _ = enc.Close() }()
	return enc.Encode(status)
}

// ToJSON converts data to JSON string (for error formatting)
func ToJSON(data interface{}) string {
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b)
}
