package logs

import (
	"sort"
	"strings"
	"time"

	"github.com/craigderington/prox/internal/storage"
)

// LogSource represents the source of a log entry
type LogSource string

const (
	LogSourceStdout LogSource = "stdout"
	LogSourceStderr LogSource = "stderr"
)

// LogEntry represents a single log line with metadata
type LogEntry struct {
	Timestamp time.Time
	Source    LogSource
	Content   string
}

// MergeLogs reads and merges stdout and stderr logs chronologically
// Returns the last N lines merged by timestamp
func MergeLogs(storage *storage.Storage, processName string, tailLines int) ([]LogEntry, error) {
	outPath := storage.GetLogFile(processName, "out")
	errPath := storage.GetLogFile(processName, "err")

	// Read log files
	outLines, err := ReadTail(outPath, tailLines)
	if err != nil {
		// If error reading, continue with empty slice (file might not exist yet)
		outLines = []string{}
	}

	errLines, err := ReadTail(errPath, tailLines)
	if err != nil {
		errLines = []string{}
	}

	// Parse and merge
	entries := make([]LogEntry, 0, len(outLines)+len(errLines))

	for _, line := range outLines {
		ts, content := parseLogLine(line)
		entries = append(entries, LogEntry{
			Timestamp: ts,
			Source:    LogSourceStdout,
			Content:   content,
		})
	}

	for _, line := range errLines {
		ts, content := parseLogLine(line)
		entries = append(entries, LogEntry{
			Timestamp: ts,
			Source:    LogSourceStderr,
			Content:   content,
		})
	}

	// Sort by timestamp (stable sort preserves order for identical timestamps)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	// Return last N entries if we have more than requested
	if len(entries) > tailLines && tailLines > 0 {
		return entries[len(entries)-tailLines:], nil
	}

	return entries, nil
}

// parseLogLine extracts timestamp and content from a log line
// Format: [2006-01-02 15:04:05] content here
// Returns (timestamp, content) or (zero time, original line) if no timestamp
func parseLogLine(line string) (time.Time, string) {
	if len(line) < 21 {
		return time.Time{}, line
	}

	if line[0] != '[' {
		return time.Time{}, line
	}

	closeBracket := strings.Index(line, "]")
	if closeBracket == -1 || closeBracket < 19 {
		return time.Time{}, line
	}

	timestampStr := line[1:closeBracket]
	ts, err := time.Parse("2006-01-02 15:04:05", timestampStr)
	if err != nil {
		return time.Time{}, line
	}

	// Return parsed timestamp and content after "] "
	content := line
	if closeBracket+2 < len(line) {
		content = line[closeBracket+2:]
	} else if closeBracket+1 < len(line) {
		content = line[closeBracket+1:]
	}

	return ts, content
}
