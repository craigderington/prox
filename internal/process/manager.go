package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// Manager manages all processes
type Manager struct {
	processes map[string]*Process
	mu        sync.RWMutex
	storage   Storage
}

// Storage interface for log paths
type Storage interface {
	GetLogFile(name, stream string) string
	LogsDir() string
}

// NewManager creates a new process manager
func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*Process),
	}
}

// SetStorage sets the storage backend for the manager
func (m *Manager) SetStorage(storage Storage) {
	m.storage = storage
}

// Start starts a new process with the given configuration
func (m *Manager) Start(config ProcessConfig) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if a process with this name already exists
	for _, p := range m.processes {
		if p.Name == config.Name {
			return nil, fmt.Errorf("process '%s' already exists", config.Name)
		}
	}

	// Set default restart policy
	restartPolicy := config.Restart
	if restartPolicy == "" {
		restartPolicy = RestartOnFailure
	}

	// Create process
	proc := &Process{
		ID:            uuid.New().String(),
		Name:          config.Name,
		Script:        config.Script,
		Interpreter:   config.Interpreter,
		Args:          config.Args,
		Cwd:           config.Cwd,
		Status:        StatusStopped,
		RestartPolicy: restartPolicy,
		DependsOn:     config.DependsOn,
		stopCh:        make(chan struct{}),
		logStopCh:     make(chan struct{}),
		manager:       m,
	}

	// Prepare environment variables
	env := os.Environ()
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	proc.Env = env

	// Set working directory to current dir if not specified
	if proc.Cwd == "" {
		proc.Cwd, _ = os.Getwd()
	}

	// Start the process
	if err := m.startProcess(proc); err != nil {
		return nil, err
	}

	m.processes[proc.ID] = proc
	return proc, nil
}

// startProcess actually spawns the process
func (m *Manager) startProcess(proc *Process) error {
	// Build command
	var cmd *exec.Cmd
	if proc.Interpreter != "" {
		// Use interpreter (e.g., node app.js)
		args := append([]string{proc.Script}, proc.Args...)
		cmd = exec.Command(proc.Interpreter, args...)
	} else {
		// Direct execution
		cmd = exec.Command(proc.Script, proc.Args...)
	}

	cmd.Dir = proc.Cwd
	cmd.Env = proc.Env

	// Set up process group for proper signal propagation on Unix
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Set up log files if storage is available
	if m.storage != nil {
		outFile, err := os.OpenFile(
			m.storage.GetLogFile(proc.Name, "out"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644,
		)
		if err != nil {
			return fmt.Errorf("failed to open stdout log: %w", err)
		}

		errFile, err := os.OpenFile(
			m.storage.GetLogFile(proc.Name, "err"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644,
		)
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open stderr log: %w", err)
		}

		// Redirect stdout/stderr to log files
		cmd.Stdout = outFile
		cmd.Stderr = errFile

		// Store file handles for cleanup later
		proc.logFiles.stdout = outFile
		proc.logFiles.stderr = errFile
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		// Clean up log files if start failed
		if proc.logFiles.stdout != nil {
			proc.logFiles.stdout.Close()
			proc.logFiles.stdout = nil
		}
		if proc.logFiles.stderr != nil {
			proc.logFiles.stderr.Close()
			proc.logFiles.stderr = nil
		}
		proc.Status = StatusErrored
		return fmt.Errorf("failed to start process: %w", err)
	}

	proc.cmd = cmd
	proc.PID = cmd.Process.Pid
	proc.Status = StatusOnline
	proc.StartedAt = time.Now()
	proc.StoppedAt = nil

	// Monitor process in background
	go m.monitorProcess(proc)

	return nil
}

// monitorProcess watches a process and handles crashes
func (m *Manager) monitorProcess(proc *Process) {
	err := proc.cmd.Wait()

	proc.mu.Lock()
	defer proc.mu.Unlock()

	// Check if this was an intentional stop
	select {
	case <-proc.stopCh:
		// Intentional stop
		proc.Status = StatusStopped
		now := time.Now()
		proc.StoppedAt = &now
		return
	default:
	}

	// Unexpected exit
	proc.Restarts++
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	// Log the error
	if err != nil {
		fmt.Fprintf(os.Stderr, "[prox] Process '%s' (PID %d) exited with error (code %d): %v\n",
			proc.Name, proc.PID, exitCode, err)
	} else {
		fmt.Fprintf(os.Stderr, "[prox] Process '%s' (PID %d) exited unexpectedly\n",
			proc.Name, proc.PID)
	}

	// Determine if we should restart based on policy
	shouldRestart := false
	switch proc.RestartPolicy {
	case RestartAlways:
		shouldRestart = true
		fmt.Fprintf(os.Stderr, "[prox] Restarting '%s' (policy: always)...\n", proc.Name)
	case RestartOnFailure:
		if exitCode != 0 {
			shouldRestart = true
			fmt.Fprintf(os.Stderr, "[prox] Restarting '%s' (policy: on-failure, exit code: %d)...\n",
				proc.Name, exitCode)
		}
	case RestartNever:
		shouldRestart = false
	}

	if shouldRestart {
		// Close old log files
		if proc.logFiles.stdout != nil {
			proc.logFiles.stdout.Close()
			proc.logFiles.stdout = nil
		}
		if proc.logFiles.stderr != nil {
			proc.logFiles.stderr.Close()
			proc.logFiles.stderr = nil
		}

		// Recreate channels
		proc.stopCh = make(chan struct{})
		proc.logStopCh = make(chan struct{})

		// Brief delay before restart
		time.Sleep(1 * time.Second)

		// Attempt restart
		if err := m.startProcess(proc); err != nil {
			fmt.Fprintf(os.Stderr, "[prox] Failed to restart '%s': %v\n", proc.Name, err)
			proc.Status = StatusErrored
			now := time.Now()
			proc.StoppedAt = &now
		}
	} else {
		// Mark as errored without restarting
		proc.Status = StatusErrored
		now := time.Now()
		proc.StoppedAt = &now

		// Close log files
		if proc.logFiles.stdout != nil {
			proc.logFiles.stdout.Close()
			proc.logFiles.stdout = nil
		}
		if proc.logFiles.stderr != nil {
			proc.logFiles.stderr.Close()
			proc.logFiles.stderr = nil
		}
	}
}

// Stop stops a process by name or ID
func (m *Manager) Stop(nameOrID string) error {
	m.mu.RLock()
	proc := m.findProcess(nameOrID)
	m.mu.RUnlock()

	if proc == nil {
		return fmt.Errorf("process not found: %s", nameOrID)
	}

	return m.stopProcess(proc)
}

// stopProcess stops a specific process
func (m *Manager) stopProcess(proc *Process) error {
	proc.mu.Lock()
	defer proc.mu.Unlock()

	if proc.Status == StatusStopped {
		return fmt.Errorf("process '%s' is already stopped", proc.Name)
	}

	// Handle case where we reconnected to a process (cmd is nil but PID exists)
	if proc.cmd == nil || proc.cmd.Process == nil {
		if proc.PID > 0 && processExists(proc.PID) {
			// We have a PID but no cmd - this happens when reconnecting to a process
			// Kill the process group directly
			if err := syscall.Kill(-proc.PID, syscall.SIGTERM); err != nil {
				// Fallback to killing just the process
				if err := syscall.Kill(proc.PID, syscall.SIGTERM); err != nil {
					return fmt.Errorf("failed to send SIGTERM to PID %d: %w", proc.PID, err)
				}
			}

			// Wait for process to exit (up to 5 seconds)
			for i := 0; i < 50; i++ {
				if !processExists(proc.PID) {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}

			// Force kill if still running
			if processExists(proc.PID) {
				syscall.Kill(-proc.PID, syscall.SIGKILL)
				syscall.Kill(proc.PID, syscall.SIGKILL) // Fallback
				time.Sleep(500 * time.Millisecond)
			}
		}

		proc.Status = StatusStopped
		now := time.Now()
		proc.StoppedAt = &now
		return nil
	}

	proc.Status = StatusStopping

	// Signal that this is an intentional stop
	close(proc.stopCh)

	// Stop log streaming
	close(proc.logStopCh)

	// Try graceful shutdown with SIGTERM
	// Use negative PID to signal the entire process group (since we used Setpgid: true)
	// This ensures child processes also receive the signal and release resources like ports
	pgid := proc.PID
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		// Fallback to signaling just the parent process if process group signal fails
		if err := proc.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}
	}

	// Wait for graceful shutdown (5 seconds)
	done := make(chan error, 1)
	go func() {
		done <- proc.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill after timeout - signal entire process group
		if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
			// Fallback to killing just the parent if process group kill fails
			if err := proc.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %w", err)
			}
		}
		<-done // Wait for Wait() to finish
	case <-done:
		// Process exited gracefully
	}

	proc.Status = StatusStopped
	now := time.Now()
	proc.StoppedAt = &now

	// Close log files
	if proc.logFiles.stdout != nil {
		proc.logFiles.stdout.Close()
		proc.logFiles.stdout = nil
	}
	if proc.logFiles.stderr != nil {
		proc.logFiles.stderr.Close()
		proc.logFiles.stderr = nil
	}

	return nil
}

// Restart restarts a process
func (m *Manager) Restart(nameOrID string) error {
	m.mu.RLock()
	proc := m.findProcess(nameOrID)
	m.mu.RUnlock()

	if proc == nil {
		return fmt.Errorf("process not found: %s", nameOrID)
	}

	proc.mu.Lock()
	proc.Status = StatusRestarting
	proc.mu.Unlock()

	// Stop the process if it's running
	if proc.Status == StatusOnline {
		if err := m.stopProcess(proc); err != nil {
			return err
		}
	}

	// Recreate stop channels
	proc.stopCh = make(chan struct{})
	proc.logStopCh = make(chan struct{})

	// Start it again
	return m.startProcess(proc)
}

// Delete removes a process (stops it first if running)
func (m *Manager) Delete(nameOrID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proc := m.findProcess(nameOrID)
	if proc == nil {
		return fmt.Errorf("process not found: %s", nameOrID)
	}

	// Stop if running
	if proc.Status == StatusOnline {
		if err := m.stopProcess(proc); err != nil {
			return err
		}
	}

	// Remove from map
	delete(m.processes, proc.ID)
	return nil
}

// List returns all managed processes
func (m *Manager) List() []*Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	processes := make([]*Process, 0, len(m.processes))
	for _, proc := range m.processes {
		processes = append(processes, proc)
	}

	// Sort alphabetically by name for consistent ordering
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Name < processes[j].Name
	})

	return processes
}

// Get returns a process by name or ID
func (m *Manager) Get(nameOrID string) *Process {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.findProcess(nameOrID)
}

// findProcess finds a process by name or ID (must be called with lock held)
func (m *Manager) findProcess(nameOrID string) *Process {
	// Try by ID first
	if proc, ok := m.processes[nameOrID]; ok {
		return proc
	}

	// Try by name
	for _, proc := range m.processes {
		if proc.Name == nameOrID {
			return proc
		}
	}

	return nil
}

// StopAll stops all running processes
func (m *Manager) StopAll() error {
	m.mu.RLock()
	processes := make([]*Process, 0, len(m.processes))
	for _, proc := range m.processes {
		if proc.Status == StatusOnline {
			processes = append(processes, proc)
		}
	}
	m.mu.RUnlock()

	var lastErr error
	for _, proc := range processes {
		if err := m.stopProcess(proc); err != nil {
			lastErr = err
			fmt.Fprintf(os.Stderr, "[prox] Error stopping '%s': %v\n", proc.Name, err)
		}
	}

	return lastErr
}

// ConfigDir returns the prox configuration directory
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".prox")
	return configDir, nil
}
