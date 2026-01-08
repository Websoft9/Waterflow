package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Websoft9/waterflow/pkg/errors"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// NodeHandlers handles node-related requests
type NodeHandlers struct {
	logger *zap.Logger
}

// NewNodeHandlers creates a new NodeHandlers instance
func NewNodeHandlers(logger *zap.Logger) *NodeHandlers {
	return &NodeHandlers{
		logger: logger,
	}
}

// ListNodes handles GET /v1/nodes
// Returns all registered nodes with their metadata
func (h *NodeHandlers) ListNodes(w http.ResponseWriter, r *http.Request) {
	// Get query parameters for filtering
	query := r.URL.Query()
	category := query.Get("category")
	search := query.Get("search")

	// Get all known nodes (hardcoded for now, will be dynamic in future)
	nodes := getKnownNodes()

	// Filter nodes if needed
	filtered := filterNodeList(nodes, category, search)

	// Prepare response
	response := map[string]interface{}{
		"nodes": filtered,
		"total": len(filtered),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode nodes response", zap.Error(err))
	}
}

// GetNode handles GET /v1/nodes/{name}
// Returns detailed information about a specific node
func (h *NodeHandlers) GetNode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["name"]

	if nodeName == "" {
		h.writeErrorLegacy(w, r, http.StatusBadRequest, "invalid_request", "Node name is required", nil)
		return
	}

	// Find node in known nodes
	nodes := getKnownNodes()
	var foundNode map[string]interface{}
	for _, node := range nodes {
		if name, ok := node["name"].(string); ok {
			if name == nodeName || name+"@"+node["version"].(string) == nodeName {
				foundNode = node
				break
			}
		}
	}

	if foundNode == nil {
		h.writeErrorLegacy(w, r, http.StatusNotFound, "not_found", "Node not found", map[string]interface{}{
			"node_name": nodeName,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(foundNode); err != nil {
		h.logger.Error("Failed to encode node response", zap.Error(err))
	}
}

// writeError writes RFC 7807 error response using pkg/errors
func (h *NodeHandlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	rfc7807 := errors.ToRFC7807(err, r.URL.Path)

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(rfc7807.Status)

	if encodeErr := json.NewEncoder(w).Encode(rfc7807); encodeErr != nil {
		h.logger.Error("Failed to encode error response", zap.Error(encodeErr))
	}
}

// writeErrorLegacy provides backward compatibility during migration
func (h *NodeHandlers) writeErrorLegacy(w http.ResponseWriter, r *http.Request, statusCode int, code, message string, details map[string]interface{}) {
	var err error
	switch code {
	case "not_found":
		err = &errors.BaseError{Type: "not_found", Message: message, Retryable: false}
	case "invalid_request", "invalid_argument":
		err = &errors.BaseError{Type: "invalid_argument", Message: message, Retryable: false}
	default:
		err = &errors.BaseError{Type: code, Message: message, Retryable: false}
	}
	if details != nil && len(details) > 0 {
		if baseErr, ok := err.(*errors.BaseError); ok {
			baseErr.Context = details
		}
	}
	h.writeError(w, r, err)
}

// getKnownNodes returns hardcoded list of known nodes
// TODO: In future, this should query NodeRegistry or database
func getKnownNodes() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "exec/shell",
			"version":     "v1",
			"category":    "exec",
			"description": "Execute shell commands",
			"input_schema": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Shell command to execute",
				},
				"shell": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"default":     "/bin/bash",
					"description": "Shell interpreter",
				},
				"timeout": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"default":     "5m",
					"description": "Maximum execution time",
				},
			},
			"output_schema": map[string]interface{}{
				"stdout":    "string",
				"stderr":    "string",
				"exit_code": "int",
			},
		},
		{
			"name":        "exec/script",
			"version":     "v1",
			"category":    "exec",
			"description": "Run script files (bash, python, node)",
			"input_schema": map[string]interface{}{
				"script_path": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Path to script file",
				},
				"interpreter": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"description": "Script interpreter",
				},
			},
			"output_schema": map[string]interface{}{
				"stdout":    "string",
				"exit_code": "int",
			},
		},
		{
			"name":        "flow/sleep",
			"version":     "v1",
			"category":    "flow",
			"description": "Delay execution for specified duration",
			"input_schema": map[string]interface{}{
				"duration": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Sleep duration (e.g., 5s, 1m, 2h)",
				},
			},
			"output_schema": map[string]interface{}{
				"slept": "duration",
			},
		},
		{
			"name":        "http/request",
			"version":     "v1",
			"category":    "http",
			"description": "Make HTTP requests (GET, POST, PUT, DELETE)",
			"input_schema": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Request URL",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"default":     "GET",
					"description": "HTTP method",
				},
				"headers": map[string]interface{}{
					"type":        "map",
					"required":    false,
					"description": "HTTP headers",
				},
			},
			"output_schema": map[string]interface{}{
				"status_code": "int",
				"body":        "string",
				"headers":     "map",
			},
		},
		{
			"name":        "file/transfer",
			"version":     "v1",
			"category":    "file",
			"description": "Transfer files via SCP/SFTP",
			"input_schema": map[string]interface{}{
				"source": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Source file path",
				},
				"destination": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Destination path",
				},
			},
			"output_schema": map[string]interface{}{
				"bytes_transferred": "int",
			},
		},
		{
			"name":        "docker/exec",
			"version":     "v1",
			"category":    "docker",
			"description": "Execute Docker commands",
			"input_schema": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Docker command",
				},
			},
			"output_schema": map[string]interface{}{
				"stdout": "string",
			},
		},
		{
			"name":        "docker/compose",
			"version":     "v1",
			"category":    "docker",
			"description": "Manage Docker Compose services",
			"input_schema": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"required":    true,
					"description": "Compose action (up, down, restart)",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"required":    false,
					"description": "Compose file path",
				},
			},
			"output_schema": map[string]interface{}{
				"status": "string",
			},
		},
	}
}

// filterNodeList filters nodes by category and search term
func filterNodeList(nodes []map[string]interface{}, category, search string) []map[string]interface{} {
	if category == "" && search == "" {
		return nodes
	}

	filtered := make([]map[string]interface{}, 0)
	for _, node := range nodes {
		// Category filter
		if category != "" {
			if nodeCategory, ok := node["category"].(string); ok {
				// Support multiple categories separated by comma
				categories := strings.Split(category, ",")
				matched := false
				for _, cat := range categories {
					if strings.TrimSpace(cat) == nodeCategory {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			} else {
				continue
			}
		}

		// Search filter (case-insensitive, searches in name and description)
		if search != "" {
			if nodeName, ok := node["name"].(string); ok {
				nodeDesc, _ := node["description"].(string)
				searchLower := strings.ToLower(search)
				nameLower := strings.ToLower(nodeName)
				descLower := strings.ToLower(nodeDesc)

				if !strings.Contains(nameLower, searchLower) && !strings.Contains(descLower, searchLower) {
					continue
				}
			} else {
				continue
			}
		}

		filtered = append(filtered, node)
	}

	return filtered
}
