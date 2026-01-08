// Package middleware provides HTTP middleware functions.
package middleware

import (
	"net/http"
	"strings"
)

// RequireAuth is a middleware that requires authentication via Bearer token or API key.
// For MVP, this is a placeholder that allows all requests.
// In production, this should validate against a token store or API key database.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract authorization header
		authHeader := r.Header.Get("Authorization")

		// Check for Bearer token
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// MVP: Accept any non-empty token
			// TODO (Story 9.1): Validate token against authentication service
			if token != "" {
				// Store user info in context for audit logging
				// For MVP, use token as username
				// ctx := context.WithValue(r.Context(), "user", token)
				next.ServeHTTP(w, r)
				return
			}
		}

		// Check for API Key header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			// MVP: Accept any non-empty API key
			// TODO (Story 9.1): Validate API key
			// ctx := context.WithValue(r.Context(), "user", apiKey)
			next.ServeHTTP(w, r)
			return
		}

		// No valid authentication found
		http.Error(w, "Unauthorized: Bearer token or X-API-Key required", http.StatusUnauthorized)
	})
}
