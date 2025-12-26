package process

import (
	"os"
	"os/exec"
	"sync"
	"time"
)

// ProcessStatus represents the current state of a process
type ProcessStatus string

const (
	StatusOnline     ProcessStatus = "online"
	StatusStopped    ProcessStatus = "stopped"
	StatusErrored    ProcessStatus = "errored"
	StatusRestarting ProcessStatus = "restarting"
	StatusStopping   ProcessStatus = "stopping"
)

// Process represents a managed process with its configuration and runtime state.
type Process struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Script        string        `json:"script"`
	Interpreter   string        `json:"interpreter"` // e.g., "node", "python", "go", ""
	Args          []string      `json:"args"`
	Cwd           string        `json:"cwd"`
	Env           []string      `json:"env"`
	PID           int           `json:"pid"`
	Status        ProcessStatus `json:"status"`
	Restarts      int           `json:"restarts"`
	StartedAt     time.Time     `json:"started_at"`
	StoppedAt     *time.Time    `json:"stopped_at,omitempty"`
	RestartPolicy RestartPolicy `json:"restart_policy"`
	DependsOn     []string      `json:"depends_on,omitempty"`

	// Runtime state (not persisted)
	cmd       *exec.Cmd
	mu        sync.RWMutex
	stopCh    chan struct{}
	logStopCh chan struct{}
	logFiles  struct {
		stdout *os.File
		stderr *os.File
	}
	manager *Manager // Reference to manager for auto-restart
}

// ProcessMetrics holds real-time metrics for a process.
// Note: NetSent and NetRecv represent system-wide network metrics,
// not per-process metrics (which are difficult to obtain accurately).
type ProcessMetrics struct {
	PID           int
	CPU           float64
	Memory        uint64 // bytes
	MemoryPercent float64
	Uptime        time.Duration
	NetSent       uint64 // system-wide bytes sent (network accounting per-process is complex)
	NetRecv       uint64 // system-wide bytes received (network accounting per-process is complex)
}

// RestartPolicy defines how a process should be restarted
type RestartPolicy string

const (
	RestartAlways    RestartPolicy = "always"
	RestartOnFailure RestartPolicy = "on-failure"
	RestartNever     RestartPolicy = "never"
)

// ProcessConfig is the configuration for starting a process.
// It defines how a process should be executed and managed.
type ProcessConfig struct {
	Name        string            `json:"name"`
	Script      string            `json:"script"`
	Interpreter string            `json:"interpreter,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Cwd         string            `json:"cwd,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Restart     RestartPolicy     `json:"restart,omitempty"`
	DependsOn   []string          `json:"depends_on,omitempty"`
}

// Uptime returns how long the process has been running
func (p *Process) Uptime() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.Status != StatusOnline {
		return 0
	}
	return time.Since(p.StartedAt)
}
