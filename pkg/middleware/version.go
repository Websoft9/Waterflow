package middleware

import (
	"net/http"
)

// Version middleware adds X-Server-Version header to all responses
func Version(version string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Server-Version", version)
			next.ServeHTTP(w, r)
		})
	}
}
