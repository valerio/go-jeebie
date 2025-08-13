package render

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// LogEntry represents a single log message with metadata
type LogEntry struct {
	Time    time.Time
	Level   slog.Level
	Message string
	Source  string
}

// LogBuffer is a thread-safe circular buffer for log entries
type LogBuffer struct {
	entries []LogEntry
	size    int
	index   int
	count   int
	mutex   sync.RWMutex
}

// NewLogBuffer creates a new log buffer with the specified capacity
func NewLogBuffer(size int) *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, size),
		size:    size,
	}
}

// Add inserts a new log entry into the buffer
func (lb *LogBuffer) Add(entry LogEntry) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.entries[lb.index] = entry
	lb.index = (lb.index + 1) % lb.size
	if lb.count < lb.size {
		lb.count++
	}
}

// GetRecent returns the most recent log entries, newest first
func (lb *LogBuffer) GetRecent(maxCount int) []LogEntry {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	if lb.count == 0 {
		return nil
	}

	count := lb.count
	if maxCount > 0 && maxCount < count {
		count = maxCount
	}

	result := make([]LogEntry, count)

	// Start from the most recent entry and work backwards
	for i := 0; i < count; i++ {
		// Calculate the index of the entry (count-1-i) entries ago
		entryIndex := (lb.index - 1 - i + lb.size) % lb.size
		result[i] = lb.entries[entryIndex]
	}

	return result
}

// Clear removes all entries from the buffer
func (lb *LogBuffer) Clear() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.count = 0
	lb.index = 0
}

// LogBufferHandler is a slog.Handler that captures logs to a LogBuffer
type LogBufferHandler struct {
	buffer *LogBuffer
	level  slog.Level
}

// NewLogBufferHandler creates a new handler that writes to the given buffer
func NewLogBufferHandler(buffer *LogBuffer, level slog.Level) *LogBufferHandler {
	return &LogBufferHandler{
		buffer: buffer,
		level:  level,
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *LogBufferHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes a log record
func (h *LogBufferHandler) Handle(_ context.Context, record slog.Record) error {
	// Extract source information if available
	source := ""
	if record.PC != 0 {
		source = "app"
	}

	// Build message with attributes
	message := record.Message

	// Add structured attributes to the message
	record.Attrs(func(a slog.Attr) bool {
		message += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	entry := LogEntry{
		Time:    record.Time,
		Level:   record.Level,
		Message: message,
		Source:  source,
	}

	h.buffer.Add(entry)
	return nil
}

// WithAttrs returns a new handler with additional attributes (not implemented for simplicity)
func (h *LogBufferHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, just return the same handler
	// In a full implementation, we'd store and format attributes
	return h
}

// WithGroup returns a new handler with a group (not implemented for simplicity)
func (h *LogBufferHandler) WithGroup(name string) slog.Handler {
	// For simplicity, just return the same handler
	return h
}

// FormatLogEntry formats a log entry for display
func FormatLogEntry(entry LogEntry) string {
	levelStr := ""
	switch entry.Level {
	case slog.LevelDebug:
		levelStr = "DBG"
	case slog.LevelInfo:
		levelStr = "INF"
	case slog.LevelWarn:
		levelStr = "WRN"
	case slog.LevelError:
		levelStr = "ERR"
	default:
		levelStr = "???"
	}

	timeStr := entry.Time.Format("15:04:05")
	return fmt.Sprintf("%s [%s] %s", timeStr, levelStr, entry.Message)
}
