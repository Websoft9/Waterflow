package dsl

import "fmt"

// ExpressionError represents an error during expression evaluation
type ExpressionError struct {
	Expression string `json:"expression"`
	Message    string `json:"error"`
	Type       string `json:"type"`
	Position   int    `json:"position,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

func (e *ExpressionError) Error() string {
	if e.Position > 0 {
		return fmt.Sprintf("expression error at position %d: %s in '%s'", e.Position, e.Message, e.Expression)
	}
	return fmt.Sprintf("expression error: %s in '%s'", e.Message, e.Expression)
}

// NewExpressionError creates a new expression error
func NewExpressionError(expression, message, errorType string) *ExpressionError {
	return &ExpressionError{
		Expression: expression,
		Message:    message,
		Type:       errorType,
	}
}

// WithPosition adds position information to the error
func (e *ExpressionError) WithPosition(pos int) *ExpressionError {
	e.Position = pos
	return e
}

// WithSuggestion adds a suggestion to the error
func (e *ExpressionError) WithSuggestion(suggestion string) *ExpressionError {
	e.Suggestion = suggestion
	return e
}
