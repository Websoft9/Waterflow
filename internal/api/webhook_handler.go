package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Websoft9/waterflow/pkg/trigger"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// WebhookHandlers handles incoming webhook requests
type WebhookHandlers struct {
	logger  *zap.Logger
	manager *trigger.Manager
}

// NewWebhookHandlers creates new WebhookHandlers instance
func NewWebhookHandlers(logger *zap.Logger, manager *trigger.Manager) *WebhookHandlers {
	return &WebhookHandlers{
		logger:  logger,
		manager: manager,
	}
}

// HandleWebhook handles POST /api/v1/webhooks/{trigger_id}/trigger
func (h *WebhookHandlers) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	triggerID := vars["trigger_id"]

	// Read raw payload
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read webhook payload", zap.Error(err))
		http.Error(w, "Failed to read payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Get signature from header
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		// Try alternative header names
		signature = r.Header.Get("X-Webhook-Signature")
	}

	// Get source IP
	sourceIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		sourceIP = strings.Split(forwarded, ",")[0]
	}

	// Parse webhook event from payload
	event, err := parseWebhookPayload(payload, r.Header)
	if err != nil {
		h.logger.Error("Failed to parse webhook payload",
			zap.String("trigger_id", triggerID),
			zap.Error(err),
		)
		http.Error(w, "Invalid payload format", http.StatusBadRequest)
		return
	}

	// Set source IP
	event.SourceIP = sourceIP

	// Collect headers for logging
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Handle webhook
	response, err := h.manager.HandleWebhook(
		r.Context(),
		triggerID,
		payload,
		signature,
		headers,
		event,
	)

	if err != nil {
		h.logger.Error("Webhook handling failed",
			zap.String("trigger_id", triggerID),
			zap.Error(err),
		)

		// HIGH-2 fix: use errors.Is instead of string matching
		switch {
		case errors.Is(err, trigger.ErrInvalidSignature):
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		case errors.Is(err, trigger.ErrNotFound):
			http.Error(w, "Trigger not found", http.StatusNotFound)
		case errors.Is(err, trigger.ErrWorkflowNotFound):
			http.Error(w, "Workflow not found", http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return successful response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
	}
}

// parseWebhookPayload parses webhook payload into WebhookEvent
// Supports GitHub, GitLab, and generic webhook formats
func parseWebhookPayload(payload []byte, headers http.Header) (*trigger.WebhookEvent, error) {
	event := &trigger.WebhookEvent{}

	// Try to parse as JSON
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}

	// Detect event type from headers or payload
	eventType := headers.Get("X-GitHub-Event")
	if eventType == "" {
		eventType = headers.Get("X-GitLab-Event")
	}
	if eventType == "" {
		if t, ok := data["event_type"].(string); ok {
			eventType = t
		} else {
			eventType = "unknown"
		}
	}
	event.Type = eventType

	// Parse GitHub-style payload
	if ref, ok := data["ref"].(string); ok {
		event.Ref = ref
		// Extract branch/tag from ref
		if strings.HasPrefix(ref, "refs/heads/") {
			event.Branch = strings.TrimPrefix(ref, "refs/heads/")
		} else if strings.HasPrefix(ref, "refs/tags/") {
			event.Tag = strings.TrimPrefix(ref, "refs/tags/")
		}
	}

	// Parse commits
	if commits, ok := data["commits"].([]interface{}); ok && len(commits) > 0 {
		if commit, ok := commits[0].(map[string]interface{}); ok {
			if id, ok := commit["id"].(string); ok {
				event.CommitID = id
			}
			if msg, ok := commit["message"].(string); ok {
				event.CommitMsg = msg
			}
			if author, ok := commit["author"].(map[string]interface{}); ok {
				if name, ok := author["name"].(string); ok {
					event.Author = name
				} else if email, ok := author["email"].(string); ok {
					event.Author = email
				}
			}
			// Parse changed files
			if added, ok := commit["added"].([]interface{}); ok {
				for _, file := range added {
					if f, ok := file.(string); ok {
						event.ChangedFiles = append(event.ChangedFiles, f)
					}
				}
			}
			if modified, ok := commit["modified"].([]interface{}); ok {
				for _, file := range modified {
					if f, ok := file.(string); ok {
						event.ChangedFiles = append(event.ChangedFiles, f)
					}
				}
			}
		}
	}

	// Parse repository
	if repo, ok := data["repository"].(map[string]interface{}); ok {
		if name, ok := repo["full_name"].(string); ok {
			event.Repository = name
		} else if name, ok := repo["name"].(string); ok {
			event.Repository = name
		}
	}

	// Parse pusher/author for fallback
	if event.Author == "" {
		if pusher, ok := data["pusher"].(map[string]interface{}); ok {
			if name, ok := pusher["name"].(string); ok {
				event.Author = name
			}
		}
	}

	// Support direct field overrides (for generic webhooks)
	if branch, ok := data["branch"].(string); ok && branch != "" {
		event.Branch = branch
	}
	if tag, ok := data["tag"].(string); ok && tag != "" {
		event.Tag = tag
	}
	if commitID, ok := data["commit_id"].(string); ok && commitID != "" {
		event.CommitID = commitID
	}

	return event, nil
}
