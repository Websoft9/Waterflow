// Package mocks provides mock implementations for testing.
package mocks

import (
	"bytes"
	"io"
	"net/http"
	"sync"
)

// HTTPClient is a mock HTTP client for testing.
type HTTPClient struct {
	mu           sync.Mutex
	responses    []MockResponse
	requestCount int
	requests     []*http.Request
	defaultResp  *MockResponse
}

// MockResponse represents a mock HTTP response.
type MockResponse struct {
	StatusCode int
	Body       string
	Headers    http.Header
	Error      error
}

// NewHTTPClient creates a new mock HTTP client.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		responses: make([]MockResponse, 0),
		requests:  make([]*http.Request, 0),
	}
}

// AddResponse queues a response to be returned on the next request.
func (c *HTTPClient) AddResponse(statusCode int, body string) *HTTPClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responses = append(c.responses, MockResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    make(http.Header),
	})
	return c
}

// AddResponseWithHeaders queues a response with custom headers.
func (c *HTTPClient) AddResponseWithHeaders(statusCode int, body string, headers http.Header) *HTTPClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responses = append(c.responses, MockResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    headers,
	})
	return c
}

// AddError queues an error to be returned on the next request.
func (c *HTTPClient) AddError(err error) *HTTPClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responses = append(c.responses, MockResponse{Error: err})
	return c
}

// SetDefaultResponse sets a default response when no queued responses remain.
func (c *HTTPClient) SetDefaultResponse(statusCode int, body string) *HTTPClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.defaultResp = &MockResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    make(http.Header),
	}
	return c
}

// Do implements the http.Client interface.
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store request for verification
	c.requests = append(c.requests, req)
	c.requestCount++

	// Get the next response
	var resp MockResponse
	if len(c.responses) > 0 {
		resp = c.responses[0]
		c.responses = c.responses[1:]
	} else if c.defaultResp != nil {
		resp = *c.defaultResp
	} else {
		// Default to 200 OK with empty body
		resp = MockResponse{StatusCode: http.StatusOK, Body: "", Headers: make(http.Header)}
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	return &http.Response{
		StatusCode: resp.StatusCode,
		Body:       io.NopCloser(bytes.NewBufferString(resp.Body)),
		Header:     resp.Headers,
		Request:    req,
	}, nil
}

// RequestCount returns the number of requests made.
func (c *HTTPClient) RequestCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requestCount
}

// Requests returns all captured requests.
func (c *HTTPClient) Requests() []*http.Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requests
}

// LastRequest returns the last request made, or nil if none.
func (c *HTTPClient) LastRequest() *http.Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.requests) == 0 {
		return nil
	}
	return c.requests[len(c.requests)-1]
}

// Reset clears all responses and requests.
func (c *HTTPClient) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responses = make([]MockResponse, 0)
	c.requests = make([]*http.Request, 0)
	c.requestCount = 0
}

// HTTPDoer interface for dependency injection.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Verify that HTTPClient implements HTTPDoer.
var _ HTTPDoer = (*HTTPClient)(nil)
