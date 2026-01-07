package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

var (
	// ErrTemplateNotFound is returned when template is not found
	ErrTemplateNotFound = errors.New("template not found")

	// ErrTemplateFileTooLarge is returned when template file exceeds size limit
	ErrTemplateFileTooLarge = errors.New("template file too large")
)

const (
	// maxTemplateFileSize is the maximum template file size (1MB)
	maxTemplateFileSize = 1 * 1024 * 1024
)

// Template represents a workflow template
type Template struct {
	Name        string              `json:"name"`
	DisplayName string              `json:"display_name"`
	Description string              `json:"description"`
	Category    string              `json:"category"`
	Version     string              `json:"version,omitempty"`
	Author      string              `json:"author,omitempty"`
	Parameters  []TemplateParameter `json:"parameters"`
	Examples    []TemplateExample   `json:"examples,omitempty"`
}

// TemplateParameter represents a template parameter
type TemplateParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, integer, boolean, array, object
	Items       string      `json:"items,omitempty"`
	Required    bool        `json:"required"`
	Default     string      `json:"default,omitempty"`
	Description string      `json:"description"`
	Example     interface{} `json:"example,omitempty"`
}

// TemplateExample represents a usage example
type TemplateExample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Vars        map[string]interface{} `json:"vars"`
}

// TemplateListResponse for GET /v1/templates
type TemplateListResponse struct {
	Templates []Template `json:"templates"`
	Count     int        `json:"count"`
}

// TemplateDetailResponse for GET /v1/templates/{name}
type TemplateDetailResponse struct {
	Template
	Content string `json:"content,omitempty"` // YAML content as string (omitted if content=false)
}

// TemplateHandlers holds handler dependencies
type TemplateHandlers struct {
	logger          *zap.Logger
	templatesDir    string // Base directory for templates
	cachedTemplates []Template
	cacheMutex      sync.RWMutex // Protects cachedTemplates
	cacheOnce       sync.Once    // Ensures metadata loaded only once
	cacheLoadErr    error        // Stores cache load error
}

// NewTemplateHandlers creates a new TemplateHandlers instance
func NewTemplateHandlers(logger *zap.Logger) *TemplateHandlers {
	return &TemplateHandlers{
		logger:       logger,
		templatesDir: "examples/workflows", // Default path
	}
}

// SetTemplatesDir sets the templates directory (for testing)
func (h *TemplateHandlers) SetTemplatesDir(dir string) {
	h.templatesDir = dir
}

// ListTemplates handles GET /v1/templates
func (h *TemplateHandlers) ListTemplates(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	category := r.URL.Query().Get("category")

	// Get templates from metadata
	templates, err := h.getTemplateMetadata()
	if err != nil {
		h.logger.Error("Failed to load templates", zap.Error(err))
		h.sendError(w, r, http.StatusInternalServerError, "Failed to load templates", err)
		return
	}

	// Filter by category if specified
	if category != "" {
		templates = h.filterByCategory(templates, category)
	}

	// Build response
	response := TemplateListResponse{
		Templates: templates,
		Count:     len(templates),
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetTemplate handles GET /v1/templates/{name}
func (h *TemplateHandlers) GetTemplate(w http.ResponseWriter, r *http.Request) {
	// Get template name from path
	vars := mux.Vars(r)
	name := vars["name"]

	// Validate template name to prevent path traversal
	if err := validateTemplateName(name); err != nil {
		h.logger.Warn("Invalid template name", zap.String("name", name), zap.Error(err))
		h.sendError(w, r, http.StatusBadRequest, "Invalid template name", err)
		return
	}

	// Get template metadata
	metadata, err := h.getTemplateByName(name)
	if err != nil {
		if errors.Is(err, ErrTemplateNotFound) {
			h.logger.Info("Template not found", zap.String("name", name))
			h.sendError(w, r, http.StatusNotFound, fmt.Sprintf("Template not found: %s", name), err)
			return
		}
		h.logger.Error("Failed to load template", zap.String("name", name), zap.Error(err))
		h.sendError(w, r, http.StatusInternalServerError, "Failed to load template", err)
		return
	}

	// Build response
	response := TemplateDetailResponse{
		Template: *metadata,
	}

	// Read YAML content if requested (default: true)
	includeContent := r.URL.Query().Get("content") != "false"
	if includeContent {
		content, err := h.readTemplateContent(name)
		if err != nil {
			if errors.Is(err, ErrTemplateFileTooLarge) {
				h.logger.Warn("Template file too large", zap.String("name", name))
				h.sendError(w, r, http.StatusRequestEntityTooLarge, "Template file too large (max 1MB)", err)
				return
			}
			h.logger.Error("Failed to read template content", zap.String("name", name), zap.Error(err))
			h.sendError(w, r, http.StatusInternalServerError, "Failed to read template content", err)
			return
		}
		response.Content = content
	}

	h.sendJSON(w, http.StatusOK, response)
}

// getTemplateMetadata loads templates-metadata.json with concurrency-safe caching
func (h *TemplateHandlers) getTemplateMetadata() ([]Template, error) {
	// Load metadata only once using sync.Once
	h.cacheOnce.Do(func() {
		// Read metadata file
		metadataPath := fmt.Sprintf("%s/templates-metadata.json", h.templatesDir)
		// #nosec G304 - metadataPath is constructed from controlled templatesDir variable
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			h.cacheLoadErr = fmt.Errorf("read metadata file: %w", err)
			return
		}

		var metadata struct {
			Templates []Template `json:"templates"`
		}
		if err := json.Unmarshal(data, &metadata); err != nil {
			h.cacheLoadErr = fmt.Errorf("parse metadata: %w", err)
			return
		}

		// Cache templates (protected by sync.Once, no mutex needed here)
		h.cachedTemplates = metadata.Templates
	})

	// Return any error from loading
	if h.cacheLoadErr != nil {
		return nil, h.cacheLoadErr
	}

	// Read-lock for safe concurrent access
	h.cacheMutex.RLock()
	defer h.cacheMutex.RUnlock()

	return h.cachedTemplates, nil
}

// getTemplateByName finds a template by name
func (h *TemplateHandlers) getTemplateByName(name string) (*Template, error) {
	templates, err := h.getTemplateMetadata()
	if err != nil {
		return nil, err
	}

	for i := range templates {
		if templates[i].Name == name {
			return &templates[i], nil
		}
	}

	return nil, ErrTemplateNotFound
}

// readTemplateContent reads YAML file with size limit
func (h *TemplateHandlers) readTemplateContent(name string) (string, error) {
	path := fmt.Sprintf("%s/%s.yaml", h.templatesDir, name)

	// Check file size before reading (max 1MB)
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("template file not found: %w", err)
		}
		return "", fmt.Errorf("stat template file: %w", err)
	}

	if fileInfo.Size() > maxTemplateFileSize {
		return "", ErrTemplateFileTooLarge
	}

	// #nosec G304 - path is validated by validateTemplateName before this function is called
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read template file: %w", err)
	}
	return string(data), nil
}

// filterByCategory filters templates by category (case-insensitive)
func (h *TemplateHandlers) filterByCategory(templates []Template, category string) []Template {
	if category == "" {
		return templates
	}

	lowerCategory := strings.ToLower(category)
	var filtered []Template
	for _, t := range templates {
		if strings.ToLower(t.Category) == lowerCategory {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// validateTemplateName validates template name to prevent path traversal
func validateTemplateName(name string) error {
	if name == "" {
		return errors.New("template name cannot be empty")
	}

	// Only allow alphanumeric, hyphen, and underscore
	matched, err := regexp.MatchString(`^[a-zA-Z0-9-_]+$`, name)
	if err != nil {
		return fmt.Errorf("regex match error: %w", err)
	}
	if !matched {
		return errors.New("template name contains invalid characters")
	}

	return nil
}

// sendJSON sends JSON response
func (h *TemplateHandlers) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// sendError sends RFC 7807 error response
func (h *TemplateHandlers) sendError(w http.ResponseWriter, r *http.Request, statusCode int, detail string, err error) {
	// Map status code to type and title
	typeURL := "https://waterflow.io/errors/"
	title := http.StatusText(statusCode)

	switch statusCode {
	case http.StatusNotFound:
		typeURL += "template-not-found"
		title = "Template Not Found"
	case http.StatusBadRequest:
		typeURL += "invalid-request"
		title = "Invalid Request"
	case http.StatusRequestEntityTooLarge:
		typeURL += "payload-too-large"
		title = "Payload Too Large"
	case http.StatusInternalServerError:
		typeURL += "internal-error"
		title = "Internal Server Error"
	}

	// Use the shared ErrorResponse type from handlers.go
	errorResp := struct {
		Type     string `json:"type"`
		Title    string `json:"title"`
		Status   int    `json:"status"`
		Detail   string `json:"detail,omitempty"`
		Instance string `json:"instance,omitempty"`
	}{
		Type:     typeURL,
		Title:    title,
		Status:   statusCode,
		Detail:   detail,
		Instance: r.URL.Path,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.logger.Error("Failed to encode error response", zap.Error(err))
	}
}
