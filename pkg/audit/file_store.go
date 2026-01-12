package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// FileAuditStore implements AuditLogger using append-only JSON Lines file storage.
//
// Features:
// - Append-only writes for immutability
// - Automatic log rotation with lumberjack
// - Thread-safe concurrent access
// - Date-based file naming
type FileAuditStore struct {
	config *FileAuditStoreConfig
	writer *lumberjack.Logger
	mu     sync.Mutex
	closed bool
}

// NewFileAuditStore creates a new file-based audit logger.
//
// The store uses JSON Lines format (one JSON object per line) for easy
// parsing and streaming. Files are rotated based on size, age, and count.
func NewFileAuditStore(config *FileAuditStoreConfig) (*FileAuditStore, error) {
	if config == nil {
		return nil, fmt.Errorf("audit store config is required")
	}

	if config.Path == "" {
		return nil, fmt.Errorf("audit log path is required")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(config.Path, 0750); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Set defaults
	maxSize := config.MaxSize
	if maxSize == 0 {
		maxSize = 100 // 100 MB default
	}

	maxAge := config.MaxAge
	if maxAge == 0 {
		maxAge = 90 // 90 days default
	}

	maxBackups := config.MaxBackups
	if maxBackups == 0 {
		maxBackups = 30 // 30 files default
	}

	compress := config.Compress

	// Create lumberjack logger for rotation
	filename := filepath.Join(config.Path, "audit.log")
	writer := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize,
		MaxAge:     maxAge,
		MaxBackups: maxBackups,
		Compress:   compress,
		LocalTime:  true,
	}

	return &FileAuditStore{
		config: config,
		writer: writer,
	}, nil
}

// Log writes an audit entry to the log file.
//
// Entries are written as JSON Lines (one JSON object per line).
// This method is thread-safe and ensures atomic writes.
func (s *FileAuditStore) Log(ctx context.Context, entry *AuditLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("audit store is closed")
	}

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	// Write JSON line
	if _, err := s.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	// Write newline
	if _, err := s.writer.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// Query retrieves audit logs matching the filter.
//
// Note: This is a basic implementation that reads from the current log file.
// For production, consider using a database for better query performance.
func (s *FileAuditStore) Query(ctx context.Context, filter AuditLogFilter) ([]*AuditLogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("audit store is closed")
	}

	// Open current log file for reading
	file, err := os.Open(s.writer.Filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []*AuditLogEntry{}, nil // No logs yet
		}
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer func() { _ = file.Close() }()

	var entries []*AuditLogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry AuditLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			// Skip malformed lines
			continue
		}

		// Apply filters
		if !matchesFilter(&entry, filter) {
			continue
		}

		entries = append(entries, &entry)

		// Apply limit
		if filter.Limit > 0 && len(entries) >= filter.Limit+filter.Offset {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan audit log: %w", err)
	}

	// Apply offset
	if filter.Offset > 0 {
		if filter.Offset >= len(entries) {
			return []*AuditLogEntry{}, nil
		}
		entries = entries[filter.Offset:]
	}

	// Apply limit after offset
	if filter.Limit > 0 && len(entries) > filter.Limit {
		entries = entries[:filter.Limit]
	}

	return entries, nil
}

// Close flushes and closes the audit store.
func (s *FileAuditStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	return s.writer.Close()
}

// matchesFilter checks if an entry matches the filter criteria.
func matchesFilter(entry *AuditLogEntry, filter AuditLogFilter) bool {
	return matchesTimeRange(entry, filter) &&
		matchesEventTypes(entry, filter) &&
		matchesEventCategory(entry, filter) &&
		matchesUserID(entry, filter) &&
		matchesResourceType(entry, filter) &&
		matchesResourceID(entry, filter) &&
		matchesResult(entry, filter)
}

func matchesTimeRange(entry *AuditLogEntry, filter AuditLogFilter) bool {
	if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
		return false
	}
	return true
}

func matchesEventTypes(entry *AuditLogEntry, filter AuditLogFilter) bool {
	if len(filter.EventTypes) == 0 {
		return true
	}
	for _, et := range filter.EventTypes {
		if entry.EventType == et {
			return true
		}
	}
	return false
}

func matchesEventCategory(entry *AuditLogEntry, filter AuditLogFilter) bool {
	if filter.EventCategory == nil {
		return true
	}
	return entry.EventCategory == *filter.EventCategory
}

func matchesUserID(entry *AuditLogEntry, filter AuditLogFilter) bool {
	if filter.UserID == "" {
		return true
	}
	return entry.User != nil && entry.User.ID == filter.UserID
}

func matchesResourceType(entry *AuditLogEntry, filter AuditLogFilter) bool {
	if filter.ResourceType == "" {
		return true
	}
	return entry.Resource != nil && entry.Resource.Type == filter.ResourceType
}

func matchesResourceID(entry *AuditLogEntry, filter AuditLogFilter) bool {
	if filter.ResourceID == "" {
		return true
	}
	return entry.Resource != nil && entry.Resource.ID == filter.ResourceID
}

func matchesResult(entry *AuditLogEntry, filter AuditLogFilter) bool {
	if filter.Result == nil {
		return true
	}
	return entry.Result == *filter.Result
}
