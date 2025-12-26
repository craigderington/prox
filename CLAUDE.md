# Process Manager TUI (pm2-tui)

## Project Overview

A Terminal User Interface (TUI) application that replicates and extends the functionality of pm2, providing process management, monitoring, and control for applications written in any language (Node.js, Python, Go, Rust, etc.) - all from a beautiful, interactive terminal interface.

## Core Objectives

1. **Universal Process Management**: Run and manage applications in any language/runtime
2. **Live Monitoring**: Real-time process statistics, logs, and resource usage
3. **Auto-restart**: Intelligent restart policies with backoff strategies
4. **Full pm2 Feature Parity**: All major pm2 features accessible via TUI
5. **Enhanced UX**: Intuitive keyboard navigation and visual feedback

## Technical Stack

### Recommended Technologies
- **Language**: Go (excellent for system-level operations, cross-platform, single binary)
- **TUI Framework**: Bubbletea + Bubbles + Lipgloss (modern, composable, well-maintained)
- **Alternative**: Rust with ratatui (if you prefer Rust's safety guarantees)

### Why Go + Bubbletea?
- Single binary distribution
- Excellent concurrency for monitoring multiple processes
- Cross-platform process management
- Your existing experience with lazystack
- Rich ecosystem for system operations

## Core Features

### 1. Process Lifecycle Management
```
Commands:
- start <app> [--name <name>] [--instances <n>]
- stop <name|id>
- restart <name|id>
- delete <name|id>
- reload <name|id> (zero-downtime reload)
```

**Implementation Notes:**
- Use `os/exec` package for process spawning
- Track PIDs, restart counts, uptime
- Support for process clusters/instances
- Graceful shutdown with SIGTERM → SIGKILL fallback

### 2. Configuration Management

**Process Configuration Schema:**
```json
{
  "name": "my-app",
  "script": "app.js",
  "interpreter": "node",
  "args": ["--port", "3000"],
  "cwd": "/path/to/app",
  "instances": 1,
  "exec_mode": "fork", // or "cluster"
  "watch": false,
  "ignore_watch": ["node_modules", "*.log"],
  "max_memory_restart": "500M",
  "env": {
    "NODE_ENV": "production"
  },
  "restart_policy": {
    "max_restarts": 10,
    "min_uptime": "10s",
    "backoff": {
      "type": "exponential",
      "delay": 1000,
      "max_delay": 60000
    }
  },
  "log": {
    "out_file": "/var/log/app-out.log",
    "error_file": "/var/log/app-err.log",
    "max_size": "10M",
    "max_files": 5
  }
}
```

**Config File Support:**
- JSON/YAML process definitions
- Ecosystem files (multiple apps)
- Environment-specific configs

### 3. Monitoring & Metrics

**Real-time Metrics to Track:**
- CPU usage (%)
- Memory usage (MB/GB + %)
- Uptime
- Restart count
- PID
- Status (online, stopped, errored, restarting)
- Request rate (if applicable)
- Event loop lag (for Node.js)

**Implementation:**
```go
type ProcessMetrics struct {
    PID           int
    CPU           float64
    Memory        uint64
    MemoryPercent float64
    Uptime        time.Duration
    Restarts      int
    Status        ProcessStatus
    LastRestart   time.Time
}
```

Use libraries:
- `github.com/shirou/gopsutil` for system metrics
- Custom parsers for application-specific metrics

### 4. Log Management

**Features:**
- Live log streaming in TUI
- Log rotation
- Multiple log views (stdout, stderr, combined)
- Log search/filtering
- Follow mode (tail -f)
- Export logs

**Log Viewer UI:**
```
┌─ Logs: my-app ───────────────────────────────────┐
│ [STDOUT] 2024-12-22 10:30:15 Server started      │
│ [STDOUT] 2024-12-22 10:30:16 Listening on :3000  │
│ [STDERR] 2024-12-22 10:30:20 Warning: deprecated │
│ [STDOUT] 2024-12-22 10:30:25 Request: GET /api   │
│                                                   │
│ [f] Follow | [/] Search | [c] Clear | [q] Quit   │
└───────────────────────────────────────────────────┘
```

### 5. TUI Interface Design

#### Main Dashboard View
```
┌─ Process Manager ─────────────────────────────────────────────────────────┐
│ 5 processes | 3 online | 1 stopped | 1 errored                            │
├───────────────────────────────────────────────────────────────────────────┤
│ Name        │ Status  │ CPU │ Mem   │ Uptime │ Restarts │ PID   │ Actions│
├─────────────┼─────────┼─────┼───────┼────────┼──────────┼───────┼────────┤
│ ► api       │ online  │ 2%  │ 125MB │ 2h 15m │ 0        │ 12345 │ [r][s] │
│   webapp    │ online  │ 1%  │ 89MB  │ 1h 45m │ 1        │ 12346 │ [r][s] │
│   worker    │ online  │ 5%  │ 256MB │ 3h 02m │ 0        │ 12347 │ [r][s] │
│   backup    │ stopped │ -   │ -     │ -      │ 3        │ -     │ [r][d] │
│   notif     │ errored │ -   │ -     │ 10s    │ 5        │ -     │ [r][d] │
└───────────────────────────────────────────────────────────────────────────┘
 [a] Add  [r] Restart  [s] Stop  [d] Delete  [l] Logs  [m] Metrics  [q] Quit
```

#### Process Detail View
```
┌─ Process: api (PID 12345) ────────────────────────────────────────────────┐
│ Status: ● online                                 Uptime: 2h 15m 32s        │
│ Script: /app/server.js                          Interpreter: node          │
│ CWD: /home/user/projects/api                    Instances: 1               │
├─ Metrics ─────────────────────────────────────────────────────────────────┤
│ CPU:    ████░░░░░░ 2.3%        Memory: ████████░░ 125MB / 512MB (24%)    │
│ Restarts: 0                    Last Restart: Never                        │
├─ Resource History (5m) ───────────────────────────────────────────────────┤
│  CPU %                                Memory MB                           │
│   5 │                                   200 │                    ╭─      │
│   4 │                                   150 │                ╭───╯       │
│   3 │      ╭──╮                         100 │         ╭──────╯           │
│   2 │ ╭────╯  ╰────╮                     50 │  ╭──────╯                  │
│   1 │─╯           ╰──                     0 │──╯                         │
│     └────────────────                      └────────────────             │
├─ Environment Variables ───────────────────────────────────────────────────┤
│ NODE_ENV=production    PORT=3000    DB_HOST=localhost                     │
└───────────────────────────────────────────────────────────────────────────┘
```

#### Live Monitor View (pm2 monit)
```
┌─ Live Monitor ────────────────────────────────────────────────────────────┐
│                                                                            │
│  ┌─ api ─────────────┐  ┌─ webapp ──────────┐  ┌─ worker ──────────┐   │
│  │ ● online          │  │ ● online          │  │ ● online          │   │
│  │ CPU:  ███░ 2.3%   │  │ CPU:  █░░░ 1.1%   │  │ CPU:  ████░ 4.8%   │   │
│  │ Mem:  ████ 125MB  │  │ Mem:  ███░ 89MB   │  │ Mem:  ██████ 256MB│   │
│  │ PID:  12345       │  │ PID:  12346       │  │ PID:  12347       │   │
│  │ Up:   2h 15m      │  │ Up:   1h 45m      │  │ Up:   3h 02m      │   │
│  └───────────────────┘  └───────────────────┘  └───────────────────┘   │
│                                                                            │
│  ┌─ Recent Logs ─────────────────────────────────────────────────────────┤
│  │ [api]    Server listening on port 3000                                │
│  │ [webapp] GET /api/users 200 45ms                                      │
│  │ [worker] Processing job #12345                                        │
│  │ [api]    POST /api/login 200 123ms                                    │
│  └───────────────────────────────────────────────────────────────────────┘
└───────────────────────────────────────────────────────────────────────────┘
```

### 6. Restart Policies & Strategies

**Restart Modes:**
1. **No Restart**: Process stays down on crash
2. **Always**: Restart on any exit
3. **On Failure**: Restart only on non-zero exit
4. **Exponential Backoff**: Increasing delays between restarts

**Crash Detection:**
```go
type RestartPolicy struct {
    Mode          RestartMode
    MaxRestarts   int           // Max restarts in time window
    TimeWindow    time.Duration // Rolling window
    MinUptime     time.Duration // Min runtime to consider "healthy"
    BackoffDelay  time.Duration
    MaxBackoff    time.Duration
    BackoffFactor float64       // Exponential factor
}
```

**Crash Loop Prevention:**
- Track restart frequency
- Exponential backoff
- Max restart limit
- Mark as "errored" after threshold

### 7. Advanced Features

#### File Watching & Auto-reload
```go
// Use fsnotify for file system monitoring
type WatchConfig struct {
    Enabled     bool
    Paths       []string
    IgnoreGlobs []string
    Debounce    time.Duration
}
```

#### Clustering (Multiple Instances)
- Load balancing across multiple instances
- Port assignment strategies
- Shared state coordination

#### Health Checks
```go
type HealthCheck struct {
    Enabled   bool
    Type      string // "http", "tcp", "command"
    URL       string
    Interval  time.Duration
    Timeout   time.Duration
    Retries   int
}
```

#### Scheduled Tasks
- Cron-like scheduling
- Run commands at specific times
- Recurring task management

### 8. Storage & Persistence

**State Management:**
```
~/.pmtui/
├── config.json          # Global configuration
├── processes/           # Process definitions
│   ├── api.json
│   ├── webapp.json
│   └── worker.json
├── logs/               # Process logs
│   ├── api-out.log
│   ├── api-err.log
│   └── ...
├── pids/               # PID files
└── metrics/            # Historical metrics (optional)
```

**Persistence:**
- Save process configurations
- Track process state across restarts
- Export/import configurations

## Implementation Roadmap

### Phase 1: Core Foundation (Week 1-2)
- [ ] Project setup with Go + Bubbletea
- [ ] Basic TUI layout and navigation
- [ ] Process spawning and management
- [ ] Simple process list view
- [ ] Start/stop/restart commands

### Phase 2: Monitoring (Week 3)
- [ ] Real-time metrics collection (CPU, memory)
- [ ] Process status tracking
- [ ] Dashboard view with metrics
- [ ] Live updates

### Phase 3: Logging (Week 4)
- [ ] Log capture (stdout/stderr)
- [ ] Log viewer component
- [ ] Log rotation
- [ ] Log filtering and search

### Phase 4: Advanced Features (Week 5-6)
- [ ] Restart policies and crash detection
- [ ] File watching and auto-reload
- [ ] Configuration file support
- [ ] Process clustering

### Phase 5: Polish (Week 7)
- [ ] Detailed process view
- [ ] Live monitor view (monit)
- [ ] Help system
- [ ] Error handling and recovery
- [ ] Documentation

### Phase 6: Extended Features (Week 8+)
- [ ] Health checks
- [ ] Scheduled tasks
- [ ] Metrics history and graphs
- [ ] Export/import functionality
- [ ] Plugin system

## Key Technical Challenges

### 1. Cross-platform Process Management
```go
// Use syscall for Unix, windows specific APIs for Windows
// Handle process groups properly
// Implement proper signal handling
```

### 2. Efficient Metrics Collection
```go
// Poll system metrics without impacting performance
// Use goroutines for concurrent monitoring
// Cache metrics with appropriate refresh rates
```

### 3. Log Management at Scale
```go
// Handle high-volume log streams
// Implement efficient ring buffers
// Support log rotation without losing data
```

### 4. TUI Responsiveness
```go
// Use Bubbletea's Cmd pattern for async operations
// Separate concerns: UI, business logic, I/O
// Debounce updates to avoid screen flicker
```

## Code Structure

```
pmtui/
├── main.go                 # Entry point
├── cmd/                    # CLI commands
│   ├── start.go
│   ├── stop.go
│   └── tui.go             # TUI command
├── internal/
│   ├── config/            # Configuration management
│   ├── process/           # Process management
│   │   ├── manager.go     # Process lifecycle
│   │   ├── metrics.go     # Metrics collection
│   │   ├── restart.go     # Restart policies
│   │   └── watch.go       # File watching
│   ├── logs/              # Log management
│   │   ├── collector.go
│   │   └── rotation.go
│   ├── storage/           # Persistence layer
│   └── tui/               # TUI components
│       ├── app.go         # Main TUI app
│       ├── dashboard.go   # Dashboard view
│       ├── detail.go      # Process detail
│       ├── logs.go        # Log viewer
│       ├── monitor.go     # Live monitor
│       └── styles.go      # Lipgloss styles
├── pkg/                   # Public packages
└── go.mod
```

## Example Usage

### Starting the TUI
```bash
# Launch interactive TUI
pmtui

# Start TUI with specific config
pmtui --config /path/to/ecosystem.json
```

### CLI Mode (pm2-style)
```bash
# Start a process
pmtui start app.js --name api

# Start with ecosystem file
pmtui start ecosystem.json

# Show status
pmtui status

# View logs
pmtui logs api --lines 100 --follow

# Restart all
pmtui restart all
```

## Testing Strategy

1. **Unit Tests**: Process management, metrics, restart logic
2. **Integration Tests**: Full process lifecycle scenarios
3. **TUI Tests**: Bubbletea testing utilities
4. **Performance Tests**: Handle 50+ processes simultaneously
5. **Cross-platform Tests**: Linux, macOS, Windows

## Documentation Plan

1. **README**: Quick start, installation, basic usage
2. **User Guide**: Comprehensive feature documentation
3. **API Documentation**: Go package documentation
4. **Process Configuration**: JSON schema and examples
5. **Troubleshooting**: Common issues and solutions

## Success Criteria

- ✅ Manage 50+ processes simultaneously
- ✅ Sub-second UI updates
- ✅ Crash recovery within 1 second
- ✅ Support all major languages/runtimes
- ✅ Memory usage < 50MB for manager process
- ✅ Zero data loss during restarts
- ✅ Intuitive keyboard navigation
- ✅ Cross-platform compatibility

## Next Steps

1. **Initial Setup**: Create Go project with Bubbletea
2. **Spike**: Prototype basic process spawning and monitoring
3. **Design Review**: Validate TUI layout and navigation flow
4. **Iterative Development**: Build feature by feature
5. **User Testing**: Get feedback early and often

## References & Resources

- pm2 documentation: https://pm2.keymetrics.io/docs/usage/quick-start/
- Bubbletea examples: https://github.com/charmbracelet/bubbletea/tree/master/examples
- gopsutil: https://github.com/shirou/gopsutil
- Process management in Go: https://pkg.go.dev/os/exec

---

**Notes**: This is an ambitious project but highly achievable given your experience with lazystack. The key is to start simple (basic process start/stop with simple TUI) and iterate. Focus on getting the core loop right: spawn → monitor → display → control.
