package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Websoft9/waterflow/pkg/dsl/node"
	"github.com/Websoft9/waterflow/pkg/errors"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// NodeHandlers handles node-related requests
type NodeHandlers struct {
	logger       *zap.Logger
	nodeRegistry *node.Registry
}

// NewNodeHandlers creates a new NodeHandlers instance
// If nodeRegistry is nil, an empty registry is created for graceful degradation
func NewNodeHandlers(logger *zap.Logger, nodeRegistry *node.Registry) *NodeHandlers {
	if nodeRegistry == nil {
		nodeRegistry = node.NewRegistry() // Empty registry for graceful degradation
	}
	return &NodeHandlers{
		logger:       logger,
		nodeRegistry: nodeRegistry,
	}
}

// ListNodes handles GET /v1/nodes
// Returns all registered nodes with their metadata
func (h *NodeHandlers) ListNodes(w http.ResponseWriter, r *http.Request) {
	// Get query parameters for filtering
	query := r.URL.Query()
	category := query.Get("category")
	search := query.Get("search")

	// Get all known nodes from NodeRegistry (dynamic loading)
	nodes := h.getKnownNodes()

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

	// Find node in registry
	nodes := h.getKnownNodes()
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
	if len(details) > 0 {
		if baseErr, ok := err.(*errors.BaseError); ok {
			baseErr.Context = details
		}
	}
	h.writeError(w, r, err)
}

// getKnownNodes retrieves all registered nodes from NodeRegistry
// Converts Node interface to API response format
func (h *NodeHandlers) getKnownNodes() []map[string]interface{} {
	// Get all registered nodes from registry
	nodeKeys := h.nodeRegistry.List()
	result := make([]map[string]interface{}, 0, len(nodeKeys))

	for _, nodeKey := range nodeKeys {
		node, err := h.nodeRegistry.Get(nodeKey)
		if err != nil {
			h.logger.Warn("Failed to get node from registry", zap.String("key", nodeKey), zap.Error(err))
			continue
		}

		metadata := node.Metadata()

		// Convert to API response format (backward compatible)
		nodeInfo := map[string]interface{}{
			"name":          nodeKey, // name@version format
			"version":       node.Version(),
			"category":      metadata.Category,
			"description":   metadata.Description,
			"input_schema":  metadata.InputSchema,
			"output_schema": metadata.OutputSchema,
		}
		result = append(result, nodeInfo)
	}

	return result
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
