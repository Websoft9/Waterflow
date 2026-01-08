package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileLogHandler writes logs to a file with rotation support.
type FileLogHandler struct {
	mu           sync.Mutex
	file         *os.File
	buffer       []*LogEntry
	config       FileConfig
	bytesWritten int64
}

// FileConfig configures file log handler.
type FileConfig struct {
	Path       string
	BufferSize int
	MaxSizeMB  int64
}

// NewFileLogHandler creates a new file log handler.
func NewFileLogHandler(config FileConfig) (*FileLogHandler, error) {
	if config.BufferSize <= 0 {
		config.BufferSize = 100
	}
	if config.MaxSizeMB <= 0 {
		config.MaxSizeMB = 100
	}

	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(config.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}

	return &FileLogHandler{
		file:         file,
		buffer:       make([]*LogEntry, 0, config.BufferSize),
		config:       config,
		bytesWritten: stat.Size(),
	}, nil
}

// OnLog processes a log entry.
func (h *FileLogHandler) OnLog(ctx context.Context, entry *LogEntry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buffer = append(h.buffer, entry)

	if len(h.buffer) >= h.config.BufferSize {
		return h.flushLocked()
	}

	return nil
}

// Close flushes remaining logs and closes the file.
func (h *FileLogHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.flushLocked(); err != nil {
		h.file.Close()
		return err
	}

	return h.file.Close()
}

func (h *FileLogHandler) flushLocked() error {
	if len(h.buffer) == 0 {
		return nil
	}

	for _, entry := range h.buffer {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal log entry: %w", err)
		}

		n, err := fmt.Fprintf(h.file, "%s\n", data)
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}

		h.bytesWritten += int64(n)
	}

	h.buffer = h.buffer[:0]

	if h.bytesWritten > h.config.MaxSizeMB*1024*1024 {
		return h.rotateLocked()
	}

	return nil
}

func (h *FileLogHandler) rotateLocked() error {
	if err := h.file.Close(); err != nil {
		return fmt.Errorf("failed to close file for rotation: %w", err)
	}

	backupPath := fmt.Sprintf("%s.old", h.config.Path)
	if err := os.Rename(h.config.Path, backupPath); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	file, err := os.OpenFile(h.config.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open new log file: %w", err)
	}

	h.file = file
	h.bytesWritten = 0

	return nil
}
