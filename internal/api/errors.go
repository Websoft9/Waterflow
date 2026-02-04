package api

import (
	"encoding/json"
	"net/http"
)

// writeError writes unified error response (AC13)
func writeError(w http.ResponseWriter, r *http.Request, statusCode int, code, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")

	// Add request ID if available
	if reqID := r.Context().Value("request_id"); reqID != nil {
		w.Header().Set("X-Request-ID", reqID.(string))
	}

	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	if details != nil {
		response["error"].(map[string]interface{})["details"] = details
	}

	json.NewEncoder(w).Encode(response)
}
