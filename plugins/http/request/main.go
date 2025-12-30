// Package main implements an HTTP request node for API integration
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/websoft9/waterflow/pkg/dsl/node"
	"go.temporal.io/sdk/temporal"
)

// HTTPRequestNode implements HTTP request functionality for API integration
type HTTPRequestNode struct{}

// Name returns the node identifier
func (n *HTTPRequestNode) Name() string {
	return "http/request"
}

// Version returns the node version
func (n *HTTPRequestNode) Version() string {
	return "v1"
}

// Params returns the parameter specifications
func (n *HTTPRequestNode) Params() map[string]node.ParamSpec {
	return n.Metadata().InputSchema
}

// Metadata returns the node metadata including input/output schemas
func (n *HTTPRequestNode) Metadata() node.NodeMetadata {
	return node.NodeMetadata{
		Description: "Send HTTP requests to external APIs (GET, POST, PUT, DELETE, PATCH)",
		Category:    "http",
		InputSchema: map[string]node.ParamSpec{
			"url": {
				Type:        "string",
				Required:    true,
				Description: "Target URL (http:// or https://)",
			},
			"method": {
				Type:        "string",
				Required:    false,
				Default:     "GET",
				Description: "HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)",
			},
			"headers": {
				Type:        "object",
				Required:    false,
				Description: "Request headers as key-value pairs",
			},
			"body": {
				Type:        "string|object",
				Required:    false,
				Description: "Request body (string or JSON object)",
			},
			"timeout": {
				Type:        "string",
				Required:    false,
				Default:     "30s",
				Description: "Request timeout (e.g., '30s', '1m', max 5 minutes)",
			},
			"verify_ssl": {
				Type:        "bool",
				Required:    false,
				Default:     true,
				Description: "Verify SSL certificates (default: true)",
			},
		},
		OutputSchema: map[string]interface{}{
			"status_code":    "int - HTTP status code",
			"headers":        "object - Response headers",
			"body":           "string|object - Response body (auto-parsed if JSON)",
			"content_type":   "string - Content-Type header value",
			"content_length": "int - Response body length in bytes",
			"elapsed_ms":     "int - Request duration in milliseconds",
		},
	}
}

// Execute performs the HTTP request
func (n *HTTPRequestNode) Execute(
	ctx context.Context,
	inputs map[string]interface{},
) (*node.NodeResult, error) {
	startTime := time.Now()

	// 1. Parse and validate URL
	urlStr, ok := inputs["url"].(string)
	if !ok || urlStr == "" {
		return nil, temporal.NewApplicationError("url is required", "InvalidParameter")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, temporal.NewApplicationError(
			fmt.Sprintf("invalid url: %v", err),
			"InvalidParameter",
		)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, temporal.NewApplicationError(
			"url must use http or https scheme",
			"InvalidParameter",
		)
	}

	// 2. Get HTTP method (normalized to uppercase)
	method := "GET"
	if m, ok := inputs["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	// Validate method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true,
		"DELETE": true, "PATCH": true, "HEAD": true,
		"OPTIONS": true,
	}
	if !validMethods[method] {
		return nil, temporal.NewApplicationError(
			fmt.Sprintf("invalid HTTP method: %s", method),
			"InvalidParameter",
		)
	}

	// 3. Prepare request body
	var bodyReader io.Reader
	var contentType string

	if body, ok := inputs["body"]; ok && body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
			contentType = "text/plain"
		case map[string]interface{}:
			// JSON object - auto-marshal
			jsonData, err := json.Marshal(v)
			if err != nil {
				return nil, temporal.NewApplicationError(
					fmt.Sprintf("failed to marshal body: %v", err),
					"InvalidParameter",
				)
			}
			bodyReader = bytes.NewReader(jsonData)
			contentType = "application/json"
		default:
			return nil, temporal.NewApplicationError(
				"body must be string or object",
				"InvalidParameter",
			)
		}
	}

	// 4. Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, temporal.NewApplicationError(
			fmt.Sprintf("failed to create request: %v", err),
			"RequestCreationFailed",
		)
	}

	// 5. Set headers
	req.Header.Set("User-Agent", "Waterflow/1.0")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Apply custom headers from inputs
	if headers, ok := inputs["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if strVal, ok := v.(string); ok {
				req.Header.Set(k, strVal)
			}
		}
	}

	// 6. Parse timeout
	timeout := 30 * time.Second
	if timeoutStr, ok := inputs["timeout"].(string); ok && timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	// Validate timeout range (max 5 minutes)
	if timeout > 5*time.Minute {
		return nil, temporal.NewApplicationError(
			"timeout must not exceed 5 minutes",
			"InvalidParameter",
		)
	}

	// 7. Configure HTTP client with SSL verification setting
	verifySsl := true
	if v, ok := inputs["verify_ssl"].(bool); ok {
		verifySsl = v
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !verifySsl,
			},
		},
	}

	// 8. Send HTTP request
	resp, err := client.Do(req)
	if err != nil {
		// Classify error (network, timeout, DNS, etc.)
		return nil, classifyHTTPError(err, 0)
	}
	defer resp.Body.Close()

	// 9. Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, temporal.NewApplicationError(
			fmt.Sprintf("failed to read response: %v", err),
			"ResponseReadFailed",
		)
	}

	elapsed := time.Since(startTime)

	// 10. Parse response body (auto-detect JSON)
	var bodyOutput interface{} = string(respBody)
	respContentType := resp.Header.Get("Content-Type")

	if strings.Contains(respContentType, "application/json") && len(respBody) > 0 {
		var jsonBody interface{}
		if err := json.Unmarshal(respBody, &jsonBody); err == nil {
			bodyOutput = jsonBody
		}
		// If JSON parsing fails, keep as string
	}

	// 11. Check status code for errors
	// 4xx/5xx are errors, classified by classifyHTTPError
	if resp.StatusCode >= 400 {
		return nil, classifyHTTPError(
			fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)),
			resp.StatusCode,
		)
	}

	// 12. Return successful result
	logs := []string{
		fmt.Sprintf("HTTP %s %s", method, urlStr),
		fmt.Sprintf("Response: %d in %dms", resp.StatusCode, elapsed.Milliseconds()),
	}

	return &node.NodeResult{
		Outputs: map[string]interface{}{
			"status_code":    resp.StatusCode,
			"headers":        resp.Header,
			"body":           bodyOutput,
			"content_type":   respContentType,
			"content_length": len(respBody),
			"elapsed_ms":     elapsed.Milliseconds(),
		},
		Logs:     logs,
		Duration: elapsed,
	}, nil
}

// classifyHTTPError categorizes HTTP errors for Temporal retry logic
// - 4xx client errors: Permanent (non-retryable)
// - 5xx server errors: Temporary (retryable)
// - Network/timeout errors: Temporary (retryable)
// - TLS certificate errors: Permanent (non-retryable)
func classifyHTTPError(err error, statusCode int) error {
	errMsg := err.Error()

	// 4xx client errors - permanent, do not retry
	if statusCode >= 400 && statusCode < 500 {
		return temporal.NewNonRetryableApplicationError(
			errMsg,
			"HTTPClientError",
			nil, // no underlying cause
		)
	}

	// 5xx server errors - temporary, can retry
	if statusCode >= 500 {
		return temporal.NewApplicationError(
			errMsg,
			"HTTPServerError",
			// Retryable by default
		)
	}

	// TLS certificate errors - permanent
	if strings.Contains(errMsg, "certificate") ||
		strings.Contains(strings.ToLower(errMsg), "tls") {
		return temporal.NewNonRetryableApplicationError(
			errMsg,
			"TLSError",
			nil, // no underlying cause
		)
	}

	// Network errors, timeout, DNS - temporary, can retry
	return temporal.NewApplicationError(
		errMsg,
		"NetworkError",
		// Retryable by default
	)
}

// Register is the exported function for Go plugin system
func Register() node.Node {
	return &HTTPRequestNode{}
}
