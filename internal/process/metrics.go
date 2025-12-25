package process

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	gopsutil_process "github.com/shirou/gopsutil/v3/process"
)

// MetricsCollector collects metrics for all managed processes
type MetricsCollector struct {
	manager *Manager
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(manager *Manager) *MetricsCollector {
	return &MetricsCollector{
		manager: manager,
	}
}

// CollectMetrics collects metrics for a specific process
func (mc *MetricsCollector) CollectMetrics(proc *Process) (*ProcessMetrics, error) {
	proc.mu.RLock()
	pid := proc.PID
	status := proc.Status
	startedAt := proc.StartedAt
	proc.mu.RUnlock()

	// If process is not online, return empty metrics
	if status != StatusOnline || pid <= 0 {
		return &ProcessMetrics{
			PID:    pid,
			Uptime: 0,
		}, nil
	}

	// Get process handle from gopsutil
	p, err := gopsutil_process.NewProcess(int32(pid))
	if err != nil {
		// Process might have just exited
		return &ProcessMetrics{
			PID:    pid,
			Uptime: time.Since(startedAt),
		}, nil
	}

	// Collect CPU percentage
	cpuPercent, err := p.CPUPercent()
	if err != nil {
		cpuPercent = 0
	}

	// Collect memory info
	memInfo, err := p.MemoryInfo()
	var memory uint64
	var memoryPercent float64
	if err == nil && memInfo != nil {
		memory = memInfo.RSS
		// Calculate memory percentage
		memPercent, err := p.MemoryPercent()
		if err == nil {
			memoryPercent = float64(memPercent)
		}
	}

	// Collect network IO from /proc/<pid>/net/dev
	var netSent, netRecv uint64
	netDevPath := fmt.Sprintf("/proc/%d/net/dev", pid)
	data, err := ioutil.ReadFile(netDevPath)
	if err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			if lineNum <= 2 {
				continue // Skip header
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			// Split by spaces/tabs, but handle multiple spaces
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				// fields[0] is interface: , fields[1] is receive bytes, fields[9] is transmit bytes
				if recvBytes, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					netRecv += recvBytes
				}
				if sentBytes, err := strconv.ParseUint(fields[9], 10, 64); err == nil {
					netSent += sentBytes
				}
			}
		}
	}

	return &ProcessMetrics{
		PID:           pid,
		CPU:           cpuPercent,
		Memory:        memory,
		MemoryPercent: memoryPercent,
		Uptime:        time.Since(startedAt),
		NetSent:       netSent,
		NetRecv:       netRecv,
	}, nil
}

// CollectAllMetrics collects metrics for all managed processes
func (mc *MetricsCollector) CollectAllMetrics() map[string]*ProcessMetrics {
	processes := mc.manager.List()
	metrics := make(map[string]*ProcessMetrics)

	for _, proc := range processes {
		m, err := mc.CollectMetrics(proc)
		if err != nil {
			// Log error but continue
			fmt.Printf("[prox] Error collecting metrics for %s: %v\n", proc.Name, err)
			continue
		}
		metrics[proc.ID] = m
	}

	return metrics
}

// GetSystemCPU returns overall system CPU usage
func GetSystemCPU() (float64, error) {
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}
	if len(percentages) > 0 {
		return percentages[0], nil
	}
	return 0, nil
}

// FormatBytes formats bytes into human-readable string
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats a duration into human-readable string
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dd%dh", int(d.Hours())/24, int(d.Hours())%24)
}
