// Package apitest provides RFC 7807 (Problem Details) assertion helpers.
package apitest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// RFC7807Error represents an RFC 7807 Problem Details response.
type RFC7807Error struct {
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	Status   int                    `json:"status"`
	Detail   string                 `json:"detail,omitempty"`
	Instance string                 `json:"instance,omitempty"`
	Extra    map[string]interface{} `json:"-"`
}

// RFC7807Assertion provides fluent assertions for RFC 7807 responses.
type RFC7807Assertion struct {
	t        *testing.T
	response *Response
	error    RFC7807Error
	parsed   bool
}

// AssertRFC7807 starts RFC 7807 assertion chain.
func (r *Response) AssertRFC7807() *RFC7807Assertion {
	r.t.Helper()
	return &RFC7807Assertion{
		t:        r.t,
		response: r,
	}
}

// parse parses the response body as RFC 7807 if not already done.
func (a *RFC7807Assertion) parse() *RFC7807Assertion {
	if !a.parsed {
		a.response.JSON(&a.error)
		a.parsed = true
	}
	return a
}

// HasType asserts the error type matches.
func (a *RFC7807Assertion) HasType(expected string) *RFC7807Assertion {
	a.t.Helper()
	a.parse()
	assert.Equal(a.t, expected, a.error.Type, "RFC 7807 type mismatch")
	return a
}

// HasTitle asserts the error title matches.
func (a *RFC7807Assertion) HasTitle(expected string) *RFC7807Assertion {
	a.t.Helper()
	a.parse()
	assert.Equal(a.t, expected, a.error.Title, "RFC 7807 title mismatch")
	return a
}

// HasStatus asserts the error status matches.
func (a *RFC7807Assertion) HasStatus(expected int) *RFC7807Assertion {
	a.t.Helper()
	a.parse()
	assert.Equal(a.t, expected, a.error.Status, "RFC 7807 status mismatch")
	return a
}

// HasDetail asserts the error detail contains the substring.
func (a *RFC7807Assertion) HasDetail(substring string) *RFC7807Assertion {
	a.t.Helper()
	a.parse()
	assert.Contains(a.t, a.error.Detail, substring, "RFC 7807 detail mismatch")
	return a
}

// HasDetailExact asserts the error detail matches exactly.
func (a *RFC7807Assertion) HasDetailExact(expected string) *RFC7807Assertion {
	a.t.Helper()
	a.parse()
	assert.Equal(a.t, expected, a.error.Detail, "RFC 7807 detail mismatch")
	return a
}

// IsNotFound asserts a standard 404 error.
func (a *RFC7807Assertion) IsNotFound() *RFC7807Assertion {
	a.t.Helper()
	return a.HasType("not_found").HasStatus(404)
}

// IsBadRequest asserts a standard 400 error.
func (a *RFC7807Assertion) IsBadRequest() *RFC7807Assertion {
	a.t.Helper()
	return a.HasType("invalid_argument").HasStatus(400)
}

// IsInvalidArgument is an alias for IsBadRequest.
func (a *RFC7807Assertion) IsInvalidArgument() *RFC7807Assertion {
	return a.IsBadRequest()
}

// IsMethodNotAllowed asserts a standard 405 error.
func (a *RFC7807Assertion) IsMethodNotAllowed() *RFC7807Assertion {
	a.t.Helper()
	return a.HasType("method_not_allowed").HasStatus(405)
}

// IsInternalError asserts a standard 500 error.
func (a *RFC7807Assertion) IsInternalError() *RFC7807Assertion {
	a.t.Helper()
	return a.HasType("internal_error").HasStatus(500)
}

// IsServiceUnavailable asserts a standard 503 error.
func (a *RFC7807Assertion) IsServiceUnavailable() *RFC7807Assertion {
	a.t.Helper()
	return a.HasType("service_unavailable").HasStatus(503)
}
