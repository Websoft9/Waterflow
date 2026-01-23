// Package apitest provides HTTP API testing utilities for Waterflow.
package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Client wraps http.Handler for easier API testing.
type Client struct {
	t       *testing.T
	handler http.Handler
	headers map[string]string
}

// NewClient creates a new API test client.
func NewClient(t *testing.T, handler http.Handler) *Client {
	return &Client{
		t:       t,
		handler: handler,
		headers: make(map[string]string),
	}
}

// WithHeader adds a default header to all requests.
func (c *Client) WithHeader(key, value string) *Client {
	c.headers[key] = value
	return c
}

// WithJSON sets Content-Type to application/json.
func (c *Client) WithJSON() *Client {
	return c.WithHeader("Content-Type", "application/json")
}

// Request represents an HTTP request being built.
type Request struct {
	client  *Client
	method  string
	path    string
	body    io.Reader
	headers map[string]string
	query   map[string]string
}

// Get creates a GET request.
func (c *Client) Get(path string) *Request {
	return &Request{
		client:  c,
		method:  http.MethodGet,
		path:    path,
		headers: make(map[string]string),
		query:   make(map[string]string),
	}
}

// Post creates a POST request.
func (c *Client) Post(path string) *Request {
	return &Request{
		client:  c,
		method:  http.MethodPost,
		path:    path,
		headers: make(map[string]string),
		query:   make(map[string]string),
	}
}

// WithJSONBody sets the request body as JSON.
func (r *Request) WithJSONBody(v interface{}) *Request {
	data, err := json.Marshal(v)
	require.NoError(r.client.t, err, "Failed to marshal JSON body")
	r.body = bytes.NewReader(data)
	r.headers["Content-Type"] = "application/json"
	return r
}

// WithBody sets the request body as string.
func (r *Request) WithBody(body string) *Request {
	r.body = strings.NewReader(body)
	return r
}

// WithQuery adds a query parameter.
func (r *Request) WithQuery(key, value string) *Request {
	r.query[key] = value
	return r
}

// Do executes the request and returns the response.
func (r *Request) Do() *Response {
	r.client.t.Helper()

	url := r.path
	if len(r.query) > 0 {
		params := make([]string, 0, len(r.query))
		for k, v := range r.query {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		url = url + "?" + strings.Join(params, "&")
	}

	req := httptest.NewRequest(r.method, url, r.body)
	for k, v := range r.client.headers {
		req.Header.Set(k, v)
	}
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.client.handler.ServeHTTP(w, req)

	return &Response{
		t:        r.client.t,
		recorder: w,
	}
}

// Response wraps httptest.ResponseRecorder with assertion helpers.
type Response struct {
	t        *testing.T
	recorder *httptest.ResponseRecorder
}

// StatusCode returns the response status code.
func (r *Response) StatusCode() int {
	return r.recorder.Code
}

// Body returns the response body as string.
func (r *Response) Body() string {
	return r.recorder.Body.String()
}

// JSON unmarshals the response body into v.
func (r *Response) JSON(v interface{}) *Response {
	r.t.Helper()
	err := json.Unmarshal(r.recorder.Body.Bytes(), v)
	require.NoError(r.t, err, "Failed to unmarshal JSON response")
	return r
}

// AssertStatus asserts the response status code.
func (r *Response) AssertStatus(expected int) *Response {
	r.t.Helper()
	assert.Equal(r.t, expected, r.recorder.Code, "Body: %s", r.Body())
	return r
}

// AssertOK asserts status 200.
func (r *Response) AssertOK() *Response {
	return r.AssertStatus(http.StatusOK)
}

// AssertCreated asserts status 201.
func (r *Response) AssertCreated() *Response {
	return r.AssertStatus(http.StatusCreated)
}

// AssertBadRequest asserts status 400.
func (r *Response) AssertBadRequest() *Response {
	return r.AssertStatus(http.StatusBadRequest)
}

// AssertNotFound asserts status 404.
func (r *Response) AssertNotFound() *Response {
	return r.AssertStatus(http.StatusNotFound)
}

// AssertContentType asserts the Content-Type header.
func (r *Response) AssertContentType(expected string) *Response {
	r.t.Helper()
	actual := r.recorder.Header().Get("Content-Type")
	assert.Contains(r.t, actual, expected)
	return r
}

// AssertJSON asserts Content-Type is application/json.
func (r *Response) AssertJSON() *Response {
	return r.AssertContentType("application/json")
}

// AssertBodyContains asserts the body contains the substring.
func (r *Response) AssertBodyContains(substring string) *Response {
	r.t.Helper()
	assert.Contains(r.t, r.Body(), substring)
	return r
}
