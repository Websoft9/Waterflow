package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected Format
	}{
		{"text format", "text", FormatText},
		{"json format", "json", FormatJSON},
		{"yaml format", "yaml", FormatYAML},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New(tt.format)
			if f.format != tt.expected {
				t.Errorf("New(%q).format = %v, want %v", tt.format, f.format, tt.expected)
			}
		})
	}
}

func TestNewFormatter(t *testing.T) {
	f := NewFormatter("json", true)
	if f.format != FormatJSON {
		t.Errorf("format = %v, want %v", f.format, FormatJSON)
	}
	if !f.noColor {
		t.Error("noColor should be true")
	}
}

func TestFormatter_GetFormat(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"text", "text"},
		{"json", "json"},
		{"yaml", "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			f := New(tt.format)
			if got := f.GetFormat(); got != tt.expected {
				t.Errorf("GetFormat() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestFormatter_Print_JSON(t *testing.T) {
	// Redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f := New("json")
	data := map[string]interface{}{
		"name":   "test",
		"status": "success",
		"count":  42,
	}

	err := f.Print(data)
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}

	// Verify content
	if result["name"] != "test" {
		t.Errorf("name = %v, want test", result["name"])
	}
	if result["status"] != "success" {
		t.Errorf("status = %v, want success", result["status"])
	}
	// JSON numbers are float64
	if count, ok := result["count"].(float64); !ok || count != 42 {
		t.Errorf("count = %v, want 42", result["count"])
	}
}

func TestFormatter_Print_YAML(t *testing.T) {
	// Redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f := New("yaml")
	data := map[string]interface{}{
		"name":   "test",
		"status": "success",
		"count":  42,
	}

	err := f.Print(data)
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify it's valid YAML
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Output is not valid YAML: %v", err)
	}

	// Verify content
	if result["name"] != "test" {
		t.Errorf("name = %v, want test", result["name"])
	}
	if result["status"] != "success" {
		t.Errorf("status = %v, want success", result["status"])
	}
	if result["count"] != 42 {
		t.Errorf("count = %v, want 42", result["count"])
	}
}

func TestFormatter_Print_Text(t *testing.T) {
	// Redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f := New("text")
	data := map[string]string{
		"name":   "test",
		"status": "success",
	}

	err := f.Print(data)
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Text format just uses %+v, verify it contains the data
	if !contains(output, "name") || !contains(output, "test") {
		t.Errorf("Output should contain data fields: %s", output)
	}
}

func TestFormatter_PrintString(t *testing.T) {
	// Redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f := New("text")
	message := "Test message"

	f.PrintString(message)

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !contains(output, message) {
		t.Errorf("Output should contain %q, got: %s", message, output)
	}
}

func TestFormatConstants(t *testing.T) {
	if FormatText != "text" {
		t.Errorf("FormatText = %q, want %q", FormatText, "text")
	}
	if FormatJSON != "json" {
		t.Errorf("FormatJSON = %q, want %q", FormatJSON, "json")
	}
	if FormatYAML != "yaml" {
		t.Errorf("FormatYAML = %q, want %q", FormatYAML, "yaml")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
