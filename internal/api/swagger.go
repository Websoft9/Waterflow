package api

import (
	"net/http"
)

// ServeSwaggerUI serves the Swagger UI for API documentation
// Uses CDN-hosted Swagger UI for simplicity and auto-updates
func ServeSwaggerUI(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Waterflow API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js" crossorigin></script>
    <script>
        window.onload = () => {
            window.ui = SwaggerUIBundle({
                url: '/api/openapi.yaml',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                // Enable Try it out by default
                tryItOutEnabled: true,
                // Display request duration
                displayRequestDuration: true,
                // Persist authorization data
                persistAuthorization: true,
                // Default models expansion
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                // Show extensions
                showExtensions: true,
                // Show common extensions
                showCommonExtensions: true
            });
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}

// ServeOpenAPISpec serves the OpenAPI specification file
func ServeOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	// In production, this should serve the actual openapi.yaml file
	// For now, redirect to the file system
	http.ServeFile(w, r, "api/openapi.yaml")
}
