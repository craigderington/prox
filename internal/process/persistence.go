package process

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/craigderington/prox/internal/storage"
)

// ProcessState represents the persistent state of a process
type ProcessState struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Script      string        `json:"script"`
	Interpreter string        `json:"interpreter"`
	Args        []string      `json:"args"`
	Cwd         string        `json:"cwd"`
	Env         []string      `json:"env"`
	PID         int           `json:"pid"`
	Status      ProcessStatus `json:"status"`
	Restarts    int           `json:"restarts"`
	StartedAt   time.Time     `json:"started_at"`
	StoppedAt   *time.Time    `json:"stopped_at,omitempty"`
}

// ManagerState represents the persistent state of the manager
type ManagerState struct {
	Processes []ProcessState `json:"processes"`
}

// SaveState saves the current manager state to disk
func (m *Manager) SaveState(storage *storage.Storage) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := ManagerState{
		Processes: make([]ProcessState, 0, len(m.processes)),
	}

	for _, proc := range m.processes {
		proc.mu.RLock()
		state.Processes = append(state.Processes, ProcessState{
			ID:          proc.ID,
			Name:        proc.Name,
			Script:      proc.Script,
			Interpreter: proc.Interpreter,
			Args:        proc.Args,
			Cwd:         proc.Cwd,
			Env:         proc.Env,
			PID:         proc.PID,
			Status:      proc.Status,
			Restarts:    proc.Restarts,
			StartedAt:   proc.StartedAt,
			StoppedAt:   proc.StoppedAt,
		})
		proc.mu.RUnlock()
	}

	return storage.SaveState(state)
}

// LoadState loads the manager state from disk
func (m *Manager) LoadState(storage *storage.Storage) error {
	var rawState map[string]interface{}
	if err := storage.LoadState(&rawState); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to parse as new format first
	var state ManagerState
	if processes, ok := rawState["processes"].([]interface{}); ok && len(processes) > 0 {
		// Check if it's an array of objects (new format) or strings (old format)
		if _, ok := processes[0].(map[string]interface{}); ok {
			// New format - try to decode as ManagerState
			if err := storage.LoadState(&state); err != nil {
				return fmt.Errorf("failed to decode state as new format: %w", err)
			}
		} else if _, ok := processes[0].(string); ok {
			// Old format - array of process names
			fmt.Printf("[prox] Detected old state format, starting fresh\n")
			return nil // Start with empty state
		} else {
			return fmt.Errorf("unrecognized processes format in state file")
		}
	} else {
		// No processes in state, that's fine
		return nil
	}

	// Process the loaded state
	for _, procState := range state.Processes {
		proc := &Process{
			ID:          procState.ID,
			Name:        procState.Name,
			Script:      procState.Script,
			Interpreter: procState.Interpreter,
			Args:        procState.Args,
			Cwd:         procState.Cwd,
			Env:         procState.Env,
			PID:         procState.PID,
			Status:      procState.Status,
			Restarts:    procState.Restarts,
			StartedAt:   procState.StartedAt,
			StoppedAt:   procState.StoppedAt,
			stopCh:      make(chan struct{}),
			logStopCh:   make(chan struct{}),
		}

		// If process was online when saved, try to verify it's still running
		if proc.Status == StatusOnline {
			// Check if process still exists
			if !processExists(proc.PID) {
				proc.Status = StatusErrored
				now := time.Now()
				proc.StoppedAt = &now
			}
		}

		m.processes[proc.ID] = proc
	}

	return nil
}

// processExists checks if a process with the given PID is running
func processExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, sending signal 0 checks if process exists without actually sending a signal
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// SaveState saves manager state (convenience wrapper)
func SaveState(m *Manager, storage *storage.Storage) error {
	return m.SaveState(storage)
}

// LoadState loads manager state (convenience wrapper)
func LoadState(m *Manager, storage *storage.Storage) error {
	return m.LoadState(storage)
}

// RestoreRunningProcesses attempts to restore monitoring for running processes
func (m *Manager) RestoreRunningProcesses() error {
	m.mu.RLock()
	processes := make([]*Process, 0)
	for _, proc := range m.processes {
		if proc.Status == StatusOnline {
			processes = append(processes, proc)
		}
	}
	m.mu.RUnlock()

	for _, proc := range processes {
		// Try to reconnect to running process
		if processExists(proc.PID) {
			fmt.Printf("[prox] Reconnected to process '%s' (PID %d)\n", proc.Name, proc.PID)
			// TODO: Restore monitoring goroutine if needed
		} else {
			proc.mu.Lock()
			proc.Status = StatusErrored
			now := time.Now()
			proc.StoppedAt = &now
			proc.mu.Unlock()
			fmt.Printf("[prox] Process '%s' is no longer running\n", proc.Name)
		}
	}

	return nil
}
