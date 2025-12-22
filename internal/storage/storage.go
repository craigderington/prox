package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Storage handles persistence of process manager state
type Storage struct {
	configDir string
}

// New creates a new storage instance
func New() (*Storage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".prox")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"processes", "logs", "pids"}
	for _, subdir := range subdirs {
		dir := filepath.Join(configDir, subdir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create %s directory: %w", subdir, err)
		}
	}

	return &Storage{
		configDir: configDir,
	}, nil
}

// ConfigDir returns the configuration directory path
func (s *Storage) ConfigDir() string {
	return s.configDir
}

// ProcessesDir returns the processes directory path
func (s *Storage) ProcessesDir() string {
	return filepath.Join(s.configDir, "processes")
}

// LogsDir returns the logs directory path
func (s *Storage) LogsDir() string {
	return filepath.Join(s.configDir, "logs")
}

// PidsDir returns the PIDs directory path
func (s *Storage) PidsDir() string {
	return filepath.Join(s.configDir, "pids")
}

// StateFile returns the path to the state file
func (s *Storage) StateFile() string {
	return filepath.Join(s.configDir, "state.json")
}

// SaveState saves data to the state file
func (s *Storage) SaveState(data interface{}) error {
	file, err := os.Create(s.StateFile())
	if err != nil {
		return fmt.Errorf("failed to create state file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode state: %w", err)
	}

	return nil
}

// LoadState loads data from the state file
func (s *Storage) LoadState(data interface{}) error {
	file, err := os.Open(s.StateFile())
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No state file yet, that's ok
		}
		return fmt.Errorf("failed to open state file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(data); err != nil {
		return fmt.Errorf("failed to decode state: %w", err)
	}

	return nil
}

// SavePID saves a process PID to a file
func (s *Storage) SavePID(name string, pid int) error {
	pidFile := filepath.Join(s.PidsDir(), fmt.Sprintf("%s.pid", name))
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// RemovePID removes a process PID file
func (s *Storage) RemovePID(name string) error {
	pidFile := filepath.Join(s.PidsDir(), fmt.Sprintf("%s.pid", name))
	err := os.Remove(pidFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetLogFile returns the path for a process log file
func (s *Storage) GetLogFile(name, stream string) string {
	return filepath.Join(s.LogsDir(), fmt.Sprintf("%s-%s.log", name, stream))
}
