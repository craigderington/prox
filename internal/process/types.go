package process

import (
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

// Process represents a managed process
type Process struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Script      string        `json:"script"`
	Interpreter string        `json:"interpreter"` // e.g., "node", "python", "go", ""
	Args        []string      `json:"args"`
	Cwd         string        `json:"cwd"`
	Env         []string      `json:"env"`
	PID         int           `json:"pid"`
	Status      ProcessStatus `json:"status"`
	Restarts    int           `json:"restarts"`
	StartedAt   time.Time     `json:"started_at"`
	StoppedAt   *time.Time    `json:"stopped_at,omitempty"`

	// Runtime state (not persisted)
	cmd       *exec.Cmd
	mu        sync.RWMutex
	stopCh    chan struct{}
	logStopCh chan struct{}
}

// ProcessMetrics holds real-time metrics for a process
type ProcessMetrics struct {
	PID           int
	CPU           float64
	Memory        uint64 // bytes
	MemoryPercent float64
	Uptime        time.Duration
}

// ProcessConfig is the configuration for starting a process
type ProcessConfig struct {
	Name        string            `json:"name"`
	Script      string            `json:"script"`
	Interpreter string            `json:"interpreter,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Cwd         string            `json:"cwd,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
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
