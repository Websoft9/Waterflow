package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// StdoutLogHandler writes logs to stdout in JSON format.
type StdoutLogHandler struct {
	mu     sync.Mutex
	buffer []*LogEntry
	config StdoutConfig
}

// StdoutConfig configures stdout log handler.
type StdoutConfig struct {
	BufferSize int
	Pretty     bool
}

// NewStdoutLogHandler creates a new stdout log handler.
func NewStdoutLogHandler(config StdoutConfig) *StdoutLogHandler {
	if config.BufferSize <= 0 {
		config.BufferSize = 100
	}

	return &StdoutLogHandler{
		buffer: make([]*LogEntry, 0, config.BufferSize),
		config: config,
	}
}

// OnLog processes a log entry.
func (h *StdoutLogHandler) OnLog(ctx context.Context, entry *LogEntry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buffer = append(h.buffer, entry)

	if len(h.buffer) >= h.config.BufferSize {
		return h.flushLocked()
	}

	return nil
}

// Close flushes remaining logs.
func (h *StdoutLogHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.flushLocked()
}

// flushLocked writes buffered logs to stdout (caller must hold lock).
func (h *StdoutLogHandler) flushLocked() error {
	if len(h.buffer) == 0 {
		return nil
	}

	for _, entry := range h.buffer {
		var data []byte
		var err error

		if h.config.Pretty {
			data, err = json.MarshalIndent(entry, "", "  ")
		} else {
			data, err = json.Marshal(entry)
		}

		if err != nil {
			return fmt.Errorf("failed to marshal log entry: %w", err)
		}

		if _, err := fmt.Fprintf(os.Stdout, "%s\n", data); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
	}

	h.buffer = h.buffer[:0]
	return nil
}
