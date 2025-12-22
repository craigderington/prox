package logs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogConfig represents log configuration for a process
type LogConfig struct {
	OutFile  string // stdout log file
	ErrFile  string // stderr log file
	MaxSize  int64  // max size in bytes before rotation
	MaxFiles int    // max number of rotated files to keep
}

// DefaultLogConfig returns default log configuration
func DefaultLogConfig(logDir, processName string) LogConfig {
	return LogConfig{
		OutFile:  filepath.Join(logDir, fmt.Sprintf("%s-out.log", processName)),
		ErrFile:  filepath.Join(logDir, fmt.Sprintf("%s-err.log", processName)),
		MaxSize:  10 * 1024 * 1024, // 10MB
		MaxFiles: 5,
	}
}

// Logger handles logging for a process
type Logger struct {
	config LogConfig
	outMu  sync.Mutex
	errMu  sync.Mutex
}

// NewLogger creates a new logger
func NewLogger(config LogConfig) *Logger {
	return &Logger{
		config: config,
	}
}

// StreamLogs streams stdout and stderr from readers to log files
// This function returns immediately and streams in the background
func (l *Logger) StreamLogs(stdout, stderr io.Reader, stopCh <-chan struct{}) {
	// Stream stdout
	go func() {
		if err := l.streamToFile(stdout, l.config.OutFile, "stdout", stopCh); err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "[prox] Error streaming stdout: %v\n", err)
		}
	}()

	// Stream stderr
	go func() {
		if err := l.streamToFile(stderr, l.config.ErrFile, "stderr", stopCh); err != nil && err != io.EOF {
			fmt.Fprintf(os.Stderr, "[prox] Error streaming stderr: %v\n", err)
		}
	}()
}

// streamToFile streams data from reader to file with rotation
func (l *Logger) streamToFile(reader io.Reader, filepath, stream string, stopCh <-chan struct{}) error {
	file, err := l.openLogFile(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 64KB buffer, 1MB max

	for scanner.Scan() {
		select {
		case <-stopCh:
			return nil
		default:
		}

		line := scanner.Text()
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logLine := fmt.Sprintf("[%s] %s\n", timestamp, line)

		// Write to file
		if stream == "stdout" {
			l.outMu.Lock()
			_, err := file.WriteString(logLine)
			file.Sync() // Flush to disk
			l.outMu.Unlock()
			if err != nil {
				return err
			}
		} else {
			l.errMu.Lock()
			_, err := file.WriteString(logLine)
			file.Sync() // Flush to disk
			l.errMu.Unlock()
			if err != nil {
				return err
			}
		}

		// Check if rotation is needed
		if err := l.rotateIfNeeded(file, filepath); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// openLogFile opens or creates a log file
func (l *Logger) openLogFile(filepath string) (*os.File, error) {
	// Open file in append mode (directory should already exist from storage initialization)
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// rotateIfNeeded checks file size and rotates if necessary
func (l *Logger) rotateIfNeeded(file *os.File, filepath string) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	if stat.Size() >= l.config.MaxSize {
		return l.rotateLog(filepath)
	}

	return nil
}

// rotateLog rotates the log file
func (l *Logger) rotateLog(filepath string) error {
	// Close current file (will be reopened by caller)

	// Rotate existing files
	for i := l.config.MaxFiles - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", filepath, i)
		newPath := fmt.Sprintf("%s.%d", filepath, i+1)

		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath)
		}
	}

	// Rotate current file to .1
	newPath := fmt.Sprintf("%s.1", filepath)
	if err := os.Rename(filepath, newPath); err != nil {
		return err
	}

	// Delete oldest file if exceeds max
	oldestPath := fmt.Sprintf("%s.%d", filepath, l.config.MaxFiles+1)
	os.Remove(oldestPath)

	return nil
}

// ReadTail reads the last n lines from a log file
func ReadTail(filepath string, n int) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	// Read all lines (TODO: optimize for large files)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last n lines
	if len(lines) > n {
		return lines[len(lines)-n:], nil
	}

	return lines, nil
}

// ReadAll reads all lines from a log file
func ReadAll(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
