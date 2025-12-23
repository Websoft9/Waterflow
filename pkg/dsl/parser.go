package dsl

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Parser YAML 解析器
type Parser struct {
	logger *zap.Logger
}

// NewParser 创建解析器
func NewParser(logger *zap.Logger) *Parser {
	return &Parser{logger: logger}
}

// Parse 解析 YAML 内容为 Workflow 结构
func (p *Parser) Parse(content []byte) (*Workflow, error) {
	var workflow Workflow

	// 使用 yaml.Node 解析以保留行号信息
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, p.wrapYAMLError(err, content)
	}

	// 解析为结构体
	if err := node.Decode(&workflow); err != nil {
		return nil, p.wrapYAMLError(err, content)
	}

	// 提取行号信息
	if err := p.extractLineNumbers(&workflow, &node, content); err != nil {
		return nil, err
	}

	// 填充内部字段
	for jobName, job := range workflow.Jobs {
		job.Name = jobName

		// 设置 runs-on 默认值
		if job.RunsOn == "" {
			job.RunsOn = "default"
		}

		for i, step := range job.Steps {
			step.Index = i
		}
	}

	p.logger.Info("YAML parsed successfully",
		zap.String("workflow", workflow.Name),
		zap.Int("jobs", len(workflow.Jobs)),
	)

	return &workflow, nil
}

// wrapYAMLError 包装 YAML 错误为友好格式
func (p *Parser) wrapYAMLError(err error, content []byte) error {
	yamlErr := &ValidationError{
		Type:   "yaml_syntax_error",
		Detail: "YAML syntax error",
		Errors: []FieldError{},
	}

	// 解析 yaml 错误信息提取行号
	// 典型错误: "yaml: line 5: mapping values are not allowed in this context"
	errMsg := err.Error()

	// 提取行号
	lineRegex := regexp.MustCompile(`line\s+(\d+)`)
	matches := lineRegex.FindStringSubmatch(errMsg)

	var lineNum int
	if len(matches) > 1 {
		_, _ = fmt.Sscanf(matches[1], "%d", &lineNum)
	}

	// 生成代码片段
	snippet := ""
	suggestion := ""
	if lineNum > 0 {
		snippet = p.extractCodeSnippet(content, lineNum, 2)
		suggestion = p.generateSuggestion(errMsg)
	}

	yamlErr.Errors = append(yamlErr.Errors, FieldError{
		Line:       lineNum,
		Error:      errMsg,
		Snippet:    snippet,
		Suggestion: suggestion,
	})

	return yamlErr
}

// extractCodeSnippet 提取代码片段 (包含上下文)
func (p *Parser) extractCodeSnippet(content []byte, lineNum int, contextLines int) string {
	lines := bytes.Split(content, []byte("\n"))
	if lineNum <= 0 || lineNum > len(lines) {
		return ""
	}

	// 计算起始和结束行
	start := lineNum - contextLines - 1
	end := lineNum + contextLines
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	var buf strings.Builder
	for i := start; i < end; i++ {
		// 标记错误行
		marker := "  "
		if i == lineNum-1 {
			marker = "→ "
		}
		_, _ = buf.WriteString(fmt.Sprintf("%s%3d | %s\n", marker, i+1, lines[i]))
	}

	return buf.String()
}

// generateSuggestion 根据错误消息生成修复建议
func (p *Parser) generateSuggestion(errMsg string) string {
	errMsg = strings.ToLower(errMsg)

	if strings.Contains(errMsg, "mapping values are not allowed") {
		return "Add ':' after key name. Example: 'name: Checkout Code'"
	}
	if strings.Contains(errMsg, "did not find expected key") {
		return "Check YAML indentation. Use spaces (not tabs) for indentation."
	}
	if strings.Contains(errMsg, "could not find expected") {
		return "Check for missing closing quotes or brackets."
	}
	if strings.Contains(errMsg, "found character that cannot start") {
		return "Check for invalid characters or missing quotes around special characters."
	}

	return "Check YAML syntax. Refer to https://yaml.org/spec/1.2/spec.html"
}

// extractLineNumbers 提取字段行号映射
func (p *Parser) extractLineNumbers(workflow *Workflow, node *yaml.Node, content []byte) error {
	workflow.LineMap = make(map[string]int)

	// 遍历 YAML 节点树提取行号
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		return nil
	}

	rootNode := node.Content[0]
	if rootNode.Kind != yaml.MappingNode {
		return nil
	}

	// 解析顶层字段
	for i := 0; i < len(rootNode.Content); i += 2 {
		keyNode := rootNode.Content[i]
		valueNode := rootNode.Content[i+1]

		key := keyNode.Value
		workflow.LineMap[key] = keyNode.Line

		// 特殊处理 jobs
		if key == "jobs" && valueNode.Kind == yaml.MappingNode {
			p.extractJobLineNumbers(workflow, valueNode)
		}
	}

	return nil
}

// extractJobLineNumbers 提取 Job 行号
func (p *Parser) extractJobLineNumbers(workflow *Workflow, jobsNode *yaml.Node) {
	for i := 0; i < len(jobsNode.Content); i += 2 {
		jobKeyNode := jobsNode.Content[i]
		jobValueNode := jobsNode.Content[i+1]

		jobName := jobKeyNode.Value
		job, exists := workflow.Jobs[jobName]
		if !exists {
			continue
		}

		job.LineNum = jobKeyNode.Line
		workflow.LineMap[fmt.Sprintf("jobs.%s", jobName)] = jobKeyNode.Line

		// 提取 job 的字段行号
		if jobValueNode.Kind == yaml.MappingNode {
			for j := 0; j < len(jobValueNode.Content); j += 2 {
				fieldKeyNode := jobValueNode.Content[j]
				fieldKey := fieldKeyNode.Value
				workflow.LineMap[fmt.Sprintf("jobs.%s.%s", jobName, fieldKey)] = fieldKeyNode.Line

				// 提取 steps 行号
				if fieldKey == "steps" {
					fieldValueNode := jobValueNode.Content[j+1]
					if fieldValueNode.Kind == yaml.SequenceNode {
						p.extractStepLineNumbers(workflow, jobName, job, fieldValueNode)
					}
				}
			}
		}
	}
}

// extractStepLineNumbers 提取 Step 行号
func (p *Parser) extractStepLineNumbers(workflow *Workflow, jobName string, job *Job, stepsNode *yaml.Node) {
	for i, stepNode := range stepsNode.Content {
		if i >= len(job.Steps) {
			break
		}

		step := job.Steps[i]
		step.LineNum = stepNode.Line
		workflow.LineMap[fmt.Sprintf("jobs.%s.steps[%d]", jobName, i)] = stepNode.Line

		// 提取 step 的字段行号
		if stepNode.Kind == yaml.MappingNode {
			for j := 0; j < len(stepNode.Content); j += 2 {
				fieldKeyNode := stepNode.Content[j]
				fieldKey := fieldKeyNode.Value
				workflow.LineMap[fmt.Sprintf("jobs.%s.steps[%d].%s", jobName, i, fieldKey)] = fieldKeyNode.Line
			}
		}
	}
}
