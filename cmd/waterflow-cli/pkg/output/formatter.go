// Package output provides output formatting utilities
package output

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Format represents output format type
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// Formatter handles output formatting
type Formatter struct {
	format  Format
	noColor bool
}

// New creates a new formatter
func New(format string) *Formatter {
	return &Formatter{
		format: Format(format),
	}
}

// NewFormatter creates a new formatter with color support control
func NewFormatter(format string, noColor bool) *Formatter {
	return &Formatter{
		format:  Format(format),
		noColor: noColor,
	}
}

// GetFormat returns the current format as string
func (f *Formatter) GetFormat() string {
	return string(f.format)
}

// Print prints data in the configured format
func (f *Formatter) Print(data interface{}) error {
	switch f.format {
	case FormatJSON:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case FormatYAML:
		enc := yaml.NewEncoder(os.Stdout)
		defer func() { _ = enc.Close() }()
		return enc.Encode(data)
	case FormatText:
		fallthrough
	default:
		// Text format - simple representation
		fmt.Printf("%+v\n", data)
		return nil
	}
}

// PrintString prints a simple string message
func (f *Formatter) PrintString(message string) {
	fmt.Println(message)
}
