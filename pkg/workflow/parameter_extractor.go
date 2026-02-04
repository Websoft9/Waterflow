package workflow

import (
	"regexp"
	"strings"

	"github.com/Websoft9/waterflow/pkg/dsl"
)

// ParameterExtractor extracts parameters from workflow YAML
type ParameterExtractor struct {
	varPattern *regexp.Regexp
}

// NewParameterExtractor creates a new parameter extractor
func NewParameterExtractor() *ParameterExtractor {
	return &ParameterExtractor{
		varPattern: regexp.MustCompile(`\$\{\{\s*vars\.(\w+)\s*\}\}`),
	}
}

// ExtractParameters extracts parameters from workflow definition
func (e *ParameterExtractor) ExtractParameters(workflow *dsl.Workflow) []Parameter {
	params := make([]Parameter, 0)
	seen := make(map[string]bool)

	// Extract from vars section
	for name, value := range workflow.Vars {
		if !seen[name] {
			params = append(params, Parameter{
				Name:    name,
				Type:    inferType(value),
				Default: value,
			})
			seen[name] = true
		}
	}

	// Extract from expressions in the content
	content := workflow.Name
	for _, job := range workflow.Jobs {
		content += " " + job.RunsOn
		for _, step := range job.Steps {
			content += " " + step.Name
			if step.Uses != "" {
				content += " " + step.Uses
			}
			// Add step.With values
			for _, v := range step.With {
				if s, ok := v.(string); ok {
					content += " " + s
				}
			}
		}
	}

	// Find all ${{ vars.xxx }} patterns
	matches := e.varPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !seen[varName] {
				// Parameter used in expression but not defined in vars
				params = append(params, Parameter{
					Name: varName,
					Type: "string",
				})
				seen[varName] = true
			}
		}
	}

	return params
}

// inferType infers the parameter type from its default value
func inferType(value interface{}) string {
	if value == nil {
		return "string"
	}

	switch value.(type) {
	case int, int32, int64, float32, float64:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "string"
	}
}

// NormalizeVarName normalizes a variable name (removes special characters)
func NormalizeVarName(name string) string {
	// Remove ${{ vars. and }}
	name = strings.ReplaceAll(name, "${{", "")
	name = strings.ReplaceAll(name, "}}", "")
	name = strings.ReplaceAll(name, "vars.", "")
	name = strings.TrimSpace(name)
	return name
}
