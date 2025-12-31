package node

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
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
// This function performs runtime validation of all parameters, including:
// - Required parameter presence check
// - Type validation (string, int, float, bool, object, array)
// - Constraint validation (Pattern, Enum, MinValue, MaxValue)
// - Default value application for optional parameters
//
// Returns InputValidationError if validation fails, nil if all parameters are valid.
func ValidateInputs(inputs map[string]interface{}, specs map[string]ParamSpec) error {
	var errors []ParameterError

	// 1. Apply default values
	applyDefaultValues(inputs, specs)

	// 2. Check required parameters
	errors = append(errors, checkRequiredParameters(inputs, specs)...)

	// 3. Validate parameter types and constraints
	errors = append(errors, validateAllParameters(inputs, specs)...)

	if len(errors) > 0 {
		return &InputValidationError{
			Errors: errors,
		}
	}

	return nil
}

// applyDefaultValues applies default values for optional parameters.
func applyDefaultValues(inputs map[string]interface{}, specs map[string]ParamSpec) {
	for paramName, spec := range specs {
		if !spec.Required && spec.Default != nil {
			if _, exists := inputs[paramName]; !exists {
				inputs[paramName] = spec.Default
			}
		}
	}
}

// checkRequiredParameters validates that all required parameters are present.
func checkRequiredParameters(inputs map[string]interface{}, specs map[string]ParamSpec) []ParameterError {
	var errors []ParameterError
	for paramName, spec := range specs {
		if spec.Required {
			if _, exists := inputs[paramName]; !exists {
				errors = append(errors, ParameterError{
					ParamName: paramName,
					ErrorType: "Missing",
					Expected:  "required",
					Actual:    nil,
					Message:   fmt.Sprintf("required parameter '%s' is missing", paramName),
				})
			}
		}
	}
	return errors
}

// validateAllParameters validates types and constraints for all parameters.
func validateAllParameters(inputs map[string]interface{}, specs map[string]ParamSpec) []ParameterError {
	var errors []ParameterError
	for paramName, value := range inputs {
		spec, exists := specs[paramName]
		if !exists {
			// Unknown parameter - allow extra parameters (backward compatibility)
			continue
		}

		// Type validation
		if err := validateType(paramName, value, spec.Type); err != nil {
			errors = append(errors, *err)
			continue // Skip constraint validation if type is wrong
		}

		// Constraint validations
		errors = append(errors, validateConstraints(paramName, value, spec)...)
	}
	return errors
}

// validateConstraints validates pattern, enum, and range constraints.
func validateConstraints(paramName string, value interface{}, spec ParamSpec) []ParameterError {
	var errors []ParameterError

	// Pattern validation (only for string type)
	if spec.Pattern != "" && spec.Type == "string" {
		if err := validatePattern(paramName, value.(string), spec.Pattern); err != nil {
			errors = append(errors, *err)
		}
	}

	// Enum validation
	if len(spec.Enum) > 0 {
		if err := validateEnumConstraint(paramName, value, spec.Enum); err != nil {
			errors = append(errors, *err)
		}
	}

	// Numeric range validation
	if (spec.Type == "int" || spec.Type == "float") && (spec.MinValue != nil || spec.MaxValue != nil) {
		if err := validateRange(paramName, value, spec.MinValue, spec.MaxValue); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

// validateType validates parameter type matches expected type.
// Handles JSON number compatibility (all numbers parsed as float64).
func validateType(paramName string, value interface{}, expectedType string) *ParameterError {
	switch expectedType {
	case "string":
		return validateStringType(paramName, value)
	case "int":
		return validateIntType(paramName, value)
	case "float":
		return validateFloatType(paramName, value)
	case "bool":
		return validateBoolType(paramName, value)
	case "object":
		return validateObjectType(paramName, value)
	case "array":
		return validateArrayType(paramName, value)
	default:
		return nil
	}
}

// validateStringType validates string type.
func validateStringType(paramName string, value interface{}) *ParameterError {
	if _, ok := value.(string); !ok {
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "TypeMismatch",
			Expected:  "string",
			Actual:    getGoType(value),
			Message:   fmt.Sprintf("expected string, got %s", getGoType(value)),
		}
	}
	return nil
}

// validateIntType validates integer type with JSON compatibility.
func validateIntType(paramName string, value interface{}) *ParameterError {
	switch v := value.(type) {
	case int, int32, int64:
		return nil // Native int types
	case float64:
		// JSON-parsed numbers, check if it's a whole number
		if v == float64(int64(v)) {
			return nil // ✅ 30.0 is valid int
		}
		// Reject floats with decimal parts
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "TypeMismatch",
			Expected:  "int (whole number)",
			Actual:    fmt.Sprintf("float64(%v)", v),
			Message:   fmt.Sprintf("expected integer, got float with decimal: %v", v),
		}
	default:
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "TypeMismatch",
			Expected:  "int",
			Actual:    getGoType(value),
			Message:   fmt.Sprintf("expected int, got %s", getGoType(value)),
		}
	}
}

// validateFloatType validates float type.
func validateFloatType(paramName string, value interface{}) *ParameterError {
	switch value.(type) {
	case float32, float64:
		return nil
	case int, int32, int64:
		return nil // ✅ int can be implicitly converted to float
	default:
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "TypeMismatch",
			Expected:  "float",
			Actual:    getGoType(value),
			Message:   fmt.Sprintf("expected float, got %s", getGoType(value)),
		}
	}
}

// validateBoolType validates boolean type.
func validateBoolType(paramName string, value interface{}) *ParameterError {
	if _, ok := value.(bool); !ok {
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "TypeMismatch",
			Expected:  "bool",
			Actual:    getGoType(value),
			Message:   fmt.Sprintf("expected bool, got %s", getGoType(value)),
		}
	}
	return nil
}

// validateObjectType validates object (map) type.
func validateObjectType(paramName string, value interface{}) *ParameterError {
	if _, ok := value.(map[string]interface{}); !ok {
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "TypeMismatch",
			Expected:  "object (map[string]interface{})",
			Actual:    getGoType(value),
			Message:   fmt.Sprintf("expected object, got %s", getGoType(value)),
		}
	}
	return nil
}

// validateArrayType validates array (slice) type.
func validateArrayType(paramName string, value interface{}) *ParameterError {
	if _, ok := value.([]interface{}); !ok {
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "TypeMismatch",
			Expected:  "array ([]interface{})",
			Actual:    getGoType(value),
			Message:   fmt.Sprintf("expected array, got %s", getGoType(value)),
		}
	}
	return nil
}

// getGoType returns the Go type string for a value.
func getGoType(value interface{}) string {
	if value == nil {
		return "nil"
	}
	switch value.(type) {
	case string:
		return "string"
	case int, int32, int64:
		return "int"
	case float32, float64:
		return "float64"
	case bool:
		return "bool"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	default:
		return fmt.Sprintf("%T", value)
	}
}

// patternCache caches compiled regex patterns for performance.
// Uses sync.Map for lock-free reads in concurrent scenarios.
var patternCache sync.Map // map[string]*regexp.Regexp

// validatePattern validates string value against regex pattern.
// Uses cached compiled regex for performance.
func validatePattern(paramName, value, pattern string) *ParameterError {
	// Try to load from cache (lock-free read)
	if cached, ok := patternCache.Load(pattern); ok {
		re := cached.(*regexp.Regexp)
		if !re.MatchString(value) {
			return &ParameterError{
				ParamName: paramName,
				ErrorType: "PatternMismatch",
				Expected:  pattern,
				Actual:    value,
				Message:   fmt.Sprintf("value '%s' does not match pattern '%s'", value, pattern),
			}
		}
		return nil
	}

	// Cache miss - compile and store
	re, err := regexp.Compile(pattern)
	if err != nil {
		// Compilation failure - should have been caught in ValidateNode
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "PatternMismatch",
			Message:   fmt.Sprintf("invalid regex pattern: %v", err),
		}
	}

	// Store to cache (may store multiple times for same key, but harmless)
	patternCache.Store(pattern, re)

	// Validate
	if !re.MatchString(value) {
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "PatternMismatch",
			Expected:  pattern,
			Actual:    value,
			Message:   fmt.Sprintf("value '%s' does not match pattern '%s'", value, pattern),
		}
	}

	return nil
}

// validateEnumConstraint validates value is in enum list.
func validateEnumConstraint(paramName string, value interface{}, enum []interface{}) *ParameterError {
	for _, allowed := range enum {
		if value == allowed {
			return nil
		}
	}

	return &ParameterError{
		ParamName: paramName,
		ErrorType: "EnumViolation",
		Expected:  enum,
		Actual:    value,
		Message:   fmt.Sprintf("value '%v' is not in allowed enum: %v", value, enum),
	}
}

// validateRange validates numeric value is within min/max range.
func validateRange(paramName string, value interface{}, minValue, maxValue *float64) *ParameterError {
	numVal, err := toFloat64(value)
	if err != nil {
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "RangeViolation",
			Message:   fmt.Sprintf("expected numeric type, got %T", value),
		}
	}

	if minValue != nil && numVal < *minValue {
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "RangeViolation",
			Expected:  fmt.Sprintf(">= %f", *minValue),
			Actual:    numVal,
			Message:   fmt.Sprintf("value %f is less than minimum %f", numVal, *minValue),
		}
	}

	if maxValue != nil && numVal > *maxValue {
		return &ParameterError{
			ParamName: paramName,
			ErrorType: "RangeViolation",
			Expected:  fmt.Sprintf("<= %f", *maxValue),
			Actual:    numVal,
			Message:   fmt.Sprintf("value %f exceeds maximum %f", numVal, *maxValue),
		}
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
