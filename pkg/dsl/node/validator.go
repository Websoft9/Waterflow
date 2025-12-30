package node

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateNode validates that a node conforms to all interface requirements.
// It checks name format, version format, metadata completeness, and schema validity.
//
// Returns NodeValidationError if validation fails, nil if node is valid.
func ValidateNode(n Node) error {
	var errors []string

	// Validate Name
	name := n.Name()
	if name == "" {
		errors = append(errors, "Name() returned empty string")
	} else if !isValidNodeName(name) {
		errors = append(errors, fmt.Sprintf("Name() format invalid: %s (expected: category/name)", name))
	}

	// Validate Version
	version := n.Version()
	if version == "" {
		errors = append(errors, "Version() returned empty string")
	} else if !isValidVersion(version) {
		errors = append(errors, fmt.Sprintf("Version() format invalid: %s (expected: vX.Y.Z or vX)", version))
	}

	// Validate Metadata
	metadata := n.Metadata()
	if metadata.Description == "" {
		errors = append(errors, "Metadata().Description is empty")
	}
	if metadata.Category == "" {
		errors = append(errors, "Metadata().Category is empty")
	} else if !isValidCategory(metadata.Category) {
		errors = append(errors, fmt.Sprintf("Metadata().Category invalid: %s (expected: exec/docker/http/file/flow)", metadata.Category))
	}

	// Validate InputSchema
	if metadata.InputSchema == nil {
		errors = append(errors, "Metadata().InputSchema is nil")
	} else {
		for paramName, spec := range metadata.InputSchema {
			if err := validateParamSpec(paramName, spec); err != nil {
				errors = append(errors, err.Error())
			}
		}
	}

	// Validate OutputSchema
	if metadata.OutputSchema == nil {
		errors = append(errors, "Metadata().OutputSchema is nil")
	}

	if len(errors) > 0 {
		return &NodeValidationError{
			NodeName: name,
			Errors:   errors,
		}
	}

	return nil
}

// isValidNodeName checks if the node name follows the category/name format.
func isValidNodeName(name string) bool {
	// Format: category/name
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return false
	}
	// Both parts must be non-empty and contain only alphanumeric/dash/underscore
	namePattern := regexp.MustCompile(`^[a-z0-9_-]+$`)
	return namePattern.MatchString(parts[0]) && namePattern.MatchString(parts[1])
}

// isValidVersion checks if version follows semantic versioning (vX.Y.Z or vX).
func isValidVersion(version string) bool {
	// Format: vX, vX.Y, or vX.Y.Z
	versionPattern := regexp.MustCompile(`^v\d+(\.\d+)?(\.\d+)?$`)
	return versionPattern.MatchString(version)
}

// isValidCategory checks if category is one of the allowed values.
func isValidCategory(category string) bool {
	validCategories := map[string]bool{
		"exec":   true,
		"docker": true,
		"http":   true,
		"file":   true,
		"flow":   true,
	}
	return validCategories[category]
}

// validateParamSpec validates a single parameter specification.
func validateParamSpec(paramName string, spec ParamSpec) error {
	// Check Type is valid
	validTypes := map[string]bool{
		"string": true,
		"int":    true,
		"float":  true,
		"bool":   true,
		"object": true,
		"array":  true,
	}
	if !validTypes[spec.Type] {
		return fmt.Errorf("param %s: invalid type %s", paramName, spec.Type)
	}

	// Validate Pattern (must be valid regex if provided)
	if spec.Pattern != "" {
		if _, err := regexp.Compile(spec.Pattern); err != nil {
			return fmt.Errorf("param %s: invalid regex pattern: %w", paramName, err)
		}
	}

	// Validate MinValue/MaxValue only apply to numeric types
	if spec.MinValue != nil || spec.MaxValue != nil {
		if spec.Type != "int" && spec.Type != "float" {
			return fmt.Errorf("param %s: MinValue/MaxValue only valid for int/float types, got %s", paramName, spec.Type)
		}
	}

	// Validate MinValue <= MaxValue
	if spec.MinValue != nil && spec.MaxValue != nil {
		if *spec.MinValue > *spec.MaxValue {
			return fmt.Errorf("param %s: MinValue (%f) > MaxValue (%f)", paramName, *spec.MinValue, *spec.MaxValue)
		}
	}

	// Validate Enum values are non-empty
	if len(spec.Enum) > 0 {
		for i, val := range spec.Enum {
			if val == nil {
				return fmt.Errorf("param %s: Enum[%d] is nil", paramName, i)
			}
		}
	}

	return nil
}

// ValidateInputs validates that input values match the node's parameter specifications.
// This is a helper function for node implementations to use in their Execute method.
func ValidateInputs(inputs map[string]interface{}, specs map[string]ParamSpec) error {
	var errors []string

	for name, spec := range specs {
		value, exists := inputs[name]

		if spec.Required && !exists {
			errors = append(errors, fmt.Sprintf("required parameter missing: %s", name))
			continue
		}

		if !exists {
			continue // Optional parameter not provided
		}

		// Validate value against spec
		if err := validateValue(name, value, spec); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("input validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateValue validates a single input value against its spec
func validateValue(name string, value interface{}, spec ParamSpec) error {
	// Validate enum constraints
	if err := validateEnum(name, value, spec.Enum); err != nil {
		return err
	}

	// Validate numeric ranges
	if spec.Type == "int" || spec.Type == "float" {
		if err := validateNumericRange(name, value, spec); err != nil {
			return err
		}
	}

	// Validate string patterns
	if spec.Type == "string" && spec.Pattern != "" {
		if err := validateStringPattern(name, value, spec.Pattern); err != nil {
			return err
		}
	}

	return nil
}

// validateEnum validates enum constraints
func validateEnum(name string, value interface{}, enum []interface{}) error {
	if len(enum) == 0 {
		return nil
	}

	for _, allowed := range enum {
		if value == allowed {
			return nil
		}
	}

	return fmt.Errorf("parameter %s: value not in enum: %v", name, value)
}

// validateNumericRange validates numeric range constraints
func validateNumericRange(name string, value interface{}, spec ParamSpec) error {
	numVal, err := toFloat64(value)
	if err != nil {
		return fmt.Errorf("parameter %s: expected numeric type, got %T", name, value)
	}

	if spec.MinValue != nil && numVal < *spec.MinValue {
		return fmt.Errorf("parameter %s: value %f < minimum %f", name, numVal, *spec.MinValue)
	}
	if spec.MaxValue != nil && numVal > *spec.MaxValue {
		return fmt.Errorf("parameter %s: value %f > maximum %f", name, numVal, *spec.MaxValue)
	}

	return nil
}

// validateStringPattern validates string pattern constraints
func validateStringPattern(name string, value interface{}, pattern string) error {
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("parameter %s: expected string, got %T", name, value)
	}

	matched, _ := regexp.MatchString(pattern, strVal)
	if !matched {
		return fmt.Errorf("parameter %s: value does not match pattern %s", name, pattern)
	}

	return nil
}

// toFloat64 converts various numeric types to float64
func toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("not a numeric type: %T", value)
	}
}
