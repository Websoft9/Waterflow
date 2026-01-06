package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetWorkflowLogs tests workflow logs retrieval
func TestGetWorkflowLogs(t *testing.T) {
	tests := []struct {
		name          string
		workflowID    string
		query         LogsQuery
		serverStatus  int
		serverBody    string
		wantLogsCount int
		wantError     bool
	}{
		{
			name:         "successful logs query",
			workflowID:   "wf-123",
			query:        LogsQuery{Tail: 100},
			serverStatus: http.StatusOK,
			serverBody: `{"timestamp":"2026-01-05T10:30:45Z","level":"info","message":"Workflow started"}
{"timestamp":"2026-01-05T10:30:46Z","level":"info","job":"build","message":"Job started"}
{"timestamp":"2026-01-05T10:30:47Z","level":"error","message":"Workflow failed"}
`,
			wantLogsCount: 3,
			wantError:     false,
		},
		{
			name:          "empty logs",
			workflowID:    "wf-456",
			query:         LogsQuery{Tail: 50},
			serverStatus:  http.StatusOK,
			serverBody:    "",
			wantLogsCount: 0,
			wantError:     false,
		},
		{
			name:          "workflow not found",
			workflowID:    "nonexistent",
			query:         LogsQuery{},
			serverStatus:  http.StatusNotFound,
			serverBody:    `{"error":{"code":"not_found","message":"Workflow not found"}}`,
			wantLogsCount: 0,
			wantError:     true,
		},
		{
			name:         "with level filter",
			workflowID:   "wf-789",
			query:        LogsQuery{Tail: 100, Level: "error"},
			serverStatus: http.StatusOK,
			serverBody: `{"timestamp":"2026-01-05T10:30:47Z","level":"error","message":"Error occurred"}
`,
			wantLogsCount: 1,
			wantError:     false,
		},
		{
			name:         "with job and step filter",
			workflowID:   "wf-abc",
			query:        LogsQuery{Tail: 100, Job: "build", Step: "checkout"},
			serverStatus: http.StatusOK,
			serverBody: `{"timestamp":"2026-01-05T10:30:50Z","level":"info","job":"build","step":"checkout","message":"Cloning repo"}
`,
			wantLogsCount: 1,
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request path
				expectedPath := "/v1/workflows/" + tt.workflowID + "/logs"
				if r.URL.Path != expectedPath {
					t.Errorf("Request path = %v, want %v", r.URL.Path, expectedPath)
				}

				// Verify query parameters
				query := r.URL.Query()
				if tt.query.Tail > 0 {
					if query.Get("tail") == "" {
						t.Error("Expected 'tail' query parameter")
					}
				}
				if tt.query.Level != "" {
					if query.Get("level") != tt.query.Level {
						t.Errorf("Level parameter = %v, want %v", query.Get("level"), tt.query.Level)
					}
				}

				// Return response
				w.Header().Set("Content-Type", "application/x-ndjson")
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			// Create client
			client := New(server.URL, "", 0, false)

			// Execute
			logs, err := client.GetWorkflowLogs(tt.workflowID, tt.query)

			// Verify error
			if (err != nil) != tt.wantError {
				t.Errorf("GetWorkflowLogs() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// Verify logs count
			if !tt.wantError && len(logs) != tt.wantLogsCount {
				t.Errorf("GetWorkflowLogs() returned %v logs, want %v", len(logs), tt.wantLogsCount)
			}

			// Verify log structure
			if !tt.wantError && len(logs) > 0 {
				log := logs[0]
				if log.Timestamp == "" {
					t.Error("Log missing timestamp")
				}
				if log.Level == "" {
					t.Error("Log missing level")
				}
				if log.Message == "" {
					t.Error("Log missing message")
				}
			}
		})
	}
}

// TestLogEntryJSONMarshaling tests LogEntry JSON encoding/decoding
func TestLogEntryJSONMarshaling(t *testing.T) {
	original := LogEntry{
		Timestamp: "2026-01-05T10:30:45Z",
		Level:     "info",
		Job:       "build",
		Step:      "checkout",
		Message:   "test message",
		Error:     "test error",
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var decoded LogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp = %v, want %v", decoded.Timestamp, original.Timestamp)
	}
	if decoded.Level != original.Level {
		t.Errorf("Level = %v, want %v", decoded.Level, original.Level)
	}
	if decoded.Job != original.Job {
		t.Errorf("Job = %v, want %v", decoded.Job, original.Job)
	}
	if decoded.Step != original.Step {
		t.Errorf("Step = %v, want %v", decoded.Step, original.Step)
	}
	if decoded.Message != original.Message {
		t.Errorf("Message = %v, want %v", decoded.Message, original.Message)
	}
	if decoded.Error != original.Error {
		t.Errorf("Error = %v, want %v", decoded.Error, original.Error)
	}
}

// TestLogsQueryConstruction tests LogsQuery parameter building
func TestLogsQueryConstruction(t *testing.T) {
	tests := []struct {
		name  string
		query LogsQuery
	}{
		{"empty query", LogsQuery{}},
		{"with tail", LogsQuery{Tail: 100}},
		{"with level", LogsQuery{Level: "error"}},
		{"with job", LogsQuery{Job: "build"}},
		{"with step", LogsQuery{Step: "checkout"}},
		{"all parameters", LogsQuery{Tail: 50, Level: "info", Job: "deploy", Step: "push"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify structure compiles
			if tt.query.Tail < 0 && tt.query.Tail != -1 {
				t.Error("Invalid tail value")
			}
		})
	}
}
