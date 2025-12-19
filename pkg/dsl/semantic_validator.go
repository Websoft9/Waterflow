package dsl

import (
	"fmt"

	"github.com/Websoft9/waterflow/pkg/node"
)

// SemanticValidator 语义验证器
type SemanticValidator struct {
	nodeRegistry *node.Registry
	content      []byte // 保存原始内容用于提取 snippet
}

// NewSemanticValidator 创建语义验证器
func NewSemanticValidator(registry *node.Registry) *SemanticValidator {
	return &SemanticValidator{nodeRegistry: registry}
}

// Validate 验证工作流语义
func (v *SemanticValidator) Validate(workflow *Workflow, content []byte) error {
	v.content = content
	var errors []FieldError

	// 1. 验证节点存在性和参数
	for jobName, job := range workflow.Jobs {
		// Story 1.6: 验证Matrix配置
		matrixErrors := v.validateMatrix(jobName, job)
		errors = append(errors, matrixErrors...)

		for stepIdx, step := range job.Steps {
			stepErrors := v.validateStep(jobName, stepIdx, step)
			errors = append(errors, stepErrors...)
		}
	}

	// 2. 验证 Job 依赖
	depErrors := v.validateJobDependencies(workflow)
	errors = append(errors, depErrors...)

	if len(errors) > 0 {
		return &ValidationError{
			Type:   "semantic_validation_error",
			Detail: fmt.Sprintf("Found %d semantic errors", len(errors)),
			Errors: errors,
		}
	}

	return nil
}

// validateStep 验证步骤
func (v *SemanticValidator) validateStep(jobName string, stepIdx int, step *Step) []FieldError {
	var errors []FieldError

	// 检查节点是否存在
	n, err := v.nodeRegistry.Get(step.Uses)
	if err != nil {
		errors = append(errors, FieldError{
			Line:       step.LineNum,
			Field:      fmt.Sprintf("jobs.%s.steps[%d].uses", jobName, stepIdx),
			Error:      fmt.Sprintf("node '%s' not found", step.Uses),
			Value:      step.Uses,
			Snippet:    extractCodeSnippet(v.content, step.LineNum, 2),
			Suggestion: fmt.Sprintf("Available nodes: %v", v.nodeRegistry.List()),
		})
		return errors
	}

	// 验证参数
	paramSpecs := n.Params()

	// 检查必填参数
	for paramName, spec := range paramSpecs {
		if spec.Required {
			if _, exists := step.With[paramName]; !exists {
				errors = append(errors, FieldError{
					Line:       step.LineNum,
					Field:      fmt.Sprintf("jobs.%s.steps[%d].with.%s", jobName, stepIdx, paramName),
					Error:      "missing required parameter",
					Snippet:    extractCodeSnippet(v.content, step.LineNum, 2),
					Suggestion: fmt.Sprintf("Add '%s' parameter. %s", paramName, spec.Description),
				})
			}
		}
	}

	// 检查未知参数
	for paramName := range step.With {
		if _, exists := paramSpecs[paramName]; !exists {
			errors = append(errors, FieldError{
				Line:       step.LineNum,
				Field:      fmt.Sprintf("jobs.%s.steps[%d].with.%s", jobName, stepIdx, paramName),
				Error:      "unsupported parameter",
				Snippet:    extractCodeSnippet(v.content, step.LineNum, 2),
				Suggestion: fmt.Sprintf("Supported parameters: %v", getParamNames(paramSpecs)),
			})
		}
	}

	return errors
}

// validateJobDependencies 验证Job依赖
func (v *SemanticValidator) validateJobDependencies(workflow *Workflow) []FieldError {
	var errors []FieldError

	// 检查needs引用的Job是否存在
	for jobName, job := range workflow.Jobs {
		for _, neededJob := range job.Needs {
			if _, exists := workflow.Jobs[neededJob]; !exists {
				errors = append(errors, FieldError{
					Line:       job.LineNum,
					Field:      fmt.Sprintf("jobs.%s.needs", jobName),
					Error:      fmt.Sprintf("job '%s' not found in workflow", neededJob),
					Snippet:    extractCodeSnippet(v.content, job.LineNum, 2),
					Suggestion: fmt.Sprintf("Available jobs: %v", getJobNames(workflow.Jobs)),
				})
			}
		}
	}

	// 检查循环依赖
	if cycle := v.detectCyclicDependency(workflow); len(cycle) > 0 {
		errors = append(errors, FieldError{
			Field:      "jobs",
			Error:      fmt.Sprintf("cyclic dependency detected: %v", cycle),
			Suggestion: "Remove circular dependency between jobs",
		})
	}

	return errors
}

// validateMatrix 验证Matrix配置 (Story 1.6)
func (v *SemanticValidator) validateMatrix(jobName string, job *Job) []FieldError {
	if job.Strategy == nil {
		return nil
	}

	var errors []FieldError

	// 1. 检查matrix非空
	if len(job.Strategy.Matrix) == 0 {
		errors = append(errors, FieldError{
			Line:       job.LineNum,
			Field:      fmt.Sprintf("jobs.%s.strategy.matrix", jobName),
			Error:      "matrix is empty",
			Snippet:    extractCodeSnippet(v.content, job.LineNum, 2),
			Suggestion: "Define at least one matrix dimension with values",
		})
		return errors
	}

	// 2. 检查每个维度非空
	for dim, values := range job.Strategy.Matrix {
		if len(values) == 0 {
			errors = append(errors, FieldError{
				Line:       job.LineNum,
				Field:      fmt.Sprintf("jobs.%s.strategy.matrix.%s", jobName, dim),
				Error:      "matrix dimension is empty",
				Snippet:    extractCodeSnippet(v.content, job.LineNum, 2),
				Suggestion: fmt.Sprintf("Add at least one value to '%s' dimension", dim),
			})
		}
	}

	// 3. 检查组合数限制
	combinations := 1
	for _, values := range job.Strategy.Matrix {
		combinations *= len(values)
	}

	if combinations > 256 {
		errors = append(errors, FieldError{
			Line:       job.LineNum,
			Field:      fmt.Sprintf("jobs.%s.strategy.matrix", jobName),
			Error:      fmt.Sprintf("matrix combinations %d exceed limit 256", combinations),
			Snippet:    extractCodeSnippet(v.content, job.LineNum, 2),
			Suggestion: "Reduce matrix dimensions or split into multiple jobs",
		})
	}

	// 4. 检查include/exclude (MVP不支持)
	if len(job.Strategy.Include) > 0 {
		errors = append(errors, FieldError{
			Line:       job.LineNum,
			Field:      fmt.Sprintf("jobs.%s.strategy.include", jobName),
			Error:      "include not supported in MVP",
			Snippet:    extractCodeSnippet(v.content, job.LineNum, 2),
			Suggestion: "Use multiple matrix jobs instead",
		})
	}

	if len(job.Strategy.Exclude) > 0 {
		errors = append(errors, FieldError{
			Line:       job.LineNum,
			Field:      fmt.Sprintf("jobs.%s.strategy.exclude", jobName),
			Error:      "exclude not supported in MVP",
			Snippet:    extractCodeSnippet(v.content, job.LineNum, 2),
			Suggestion: "Use multiple matrix jobs instead",
		})
	}

	return errors
}

// detectCyclicDependency 检测循环依赖
func (v *SemanticValidator) detectCyclicDependency(workflow *Workflow) []string {
	// 使用DFS检测循环
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string

	var dfs func(string) bool
	dfs = func(jobName string) bool {
		visited[jobName] = true
		recStack[jobName] = true
		path = append(path, jobName)

		job, exists := workflow.Jobs[jobName]
		if !exists {
			return false
		}

		for _, dep := range job.Needs {
			if !visited[dep] {
				if dfs(dep) {
					return true
				}
			} else if recStack[dep] {
				// 找到循环
				path = append(path, dep)
				return true
			}
		}

		recStack[jobName] = false
		path = path[:len(path)-1]
		return false
	}

	for jobName := range workflow.Jobs {
		if !visited[jobName] {
			if dfs(jobName) {
				return path
			}
		}
	}

	return nil
}

// getParamNames 获取参数名列表
func getParamNames(specs map[string]node.ParamSpec) []string {
	names := make([]string, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	return names
}

// getJobNames 获取Job名列表
func getJobNames(jobs map[string]*Job) []string {
	names := make([]string, 0, len(jobs))
	for name := range jobs {
		names = append(names, name)
	}
	return names
}
