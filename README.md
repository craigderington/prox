# ⚡ prox

A modern, powerful process manager with a beautiful Terminal User Interface (TUI). Inspired by pm2, built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

![prox](https://img.shields.io/badge/go-1.25+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

## ✨ Features

- 🚀 **Universal Process Management** - Run applications in any language (Node.js, Python, Go, Rust, Ruby, Bash, etc.)
- 🎨 **Beautiful TUI** - Three interactive views: Dashboard, Monitor, and Logs
- 📊 **Real-time Metrics** - CPU, memory, network, and uptime monitoring
- 🔄 **Smart Process Control** - Graceful shutdown with SIGTERM → SIGKILL fallback
- 💾 **State Persistence** - Processes survive prox restarts via `~/.prox/state.json`
- 📝 **Log Management** - Live log tailing with continuous file writing
- ⌨️  **Vim-like Navigation** - Keyboard-first interface (hjkl, arrows)
- 📦 **YAML Configuration** - Define all your services in `prox.yml`
- 🔧 **Auto-detection** - Automatically detects interpreters by file extension
- 🎯 **Process Monitoring** - 4-panel detailed view (pm2 monit style)
- 📜 **Log Viewer** - Real-time log streaming with export capabilities

## 📦 Installation

### Option 1: Install via go install (Recommended)

```bash
go install github.com/craigderington/prox@latest
```

This installs the `prox` binary to `$GOPATH/bin` (usually `~/go/bin`). Make sure this directory is in your `PATH`.

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/craigderington/prox.git
cd prox

# Build the binary
go build -o prox .

# Optional: Install globally
sudo mv prox /usr/local/bin/
```

## 🚀 Quick Start

### Launch Interactive TUI (Default Mode)

```bash
prox
```

The TUI provides three main views:

1. **Dashboard** - Overview of all processes with quick actions
2. **Monitor** - Detailed 4-panel view for selected process (like pm2 monit)
3. **Logs** - Real-time log viewer with continuous export capability

### CLI Commands

```bash
# Start a process
prox start app.py --name my-worker

# Start with custom interpreter and working directory
prox start server.js --name api --cwd /path/to/app --interpreter node

# Start all services from prox.yml
prox start-all

# List all processes
prox list

# View process logs
prox logs my-worker

# Stop a process
prox stop my-worker

# Restart a process
prox restart my-worker

# Delete a process
prox delete my-worker

# Initialize prox.yml from existing processes
prox init
```

## 📋 Configuration File (prox.yml)

Create a `prox.yml` file to define all your services:

```yaml
services:
  - name: web-server
    script: server.js
    interpreter: node
    cwd: /path/to/app
    args:
      - --port
      - "3000"
    env:
      NODE_ENV: production
      PORT: "3000"

  - name: worker
    script: worker.py
    interpreter: python3
    cwd: /path/to/worker
    env:
      PYTHONUNBUFFERED: "1"

  - name: api
    script: ./api
    cwd: /path/to/api
    env:
      GO_ENV: production
```

Then start all services at once:

```bash
prox start-all
```

## 🎮 Keyboard Shortcuts

### Dashboard View

| Key | Action |
|-----|--------|
| `↑/k` | Move selection up |
| `↓/j` | Move selection down |
| `n` | Start a new process (interactive input) |
| `Enter` | Open monitor view for selected process |
| `l` | Open logs view for selected process |
| `r` | Restart selected process |
| `s` | Stop selected process |
| `d` | Delete selected process |
| `R` | Refresh process list |
| `q` | Quit |

### Monitor View (4-Panel Detailed View)

| Key | Action |
|-----|--------|
| `↑/k` | Move selection up in process list |
| `↓/j` | Move selection down in process list |
| `tab` | Switch between panels (Processes → Metrics → Metadata → Logs) |
| `f` | Toggle follow mode in logs panel |
| `w` | Write logs to file (when logs panel focused) |
| `r` | Restart selected process |
| `s` | Stop selected process |
| `d` | Delete selected process |
| `Esc/q` | Return to dashboard |

### Logs View

| Key | Action |
|-----|--------|
| `↑/k` | Scroll up one line |
| `↓/j` | Scroll down one line |
| `u` | Scroll up half page |
| `d` | Scroll down half page |
| `g` | Go to top |
| `G` | Go to bottom |
| `f` | Toggle follow mode (auto-scroll) |
| `w` | **Toggle continuous writing** - Turns GOLD when actively writing logs to file |
| `r` | Refresh logs |
| `Esc/q` | Return to dashboard |

### ✨ Continuous Log Writing

The `w` key in the logs view now works as a **toggle**:

- **Press `w` once**: Starts continuous writing mode
  - Creates a timestamped file (e.g., `myapp_logs_2025-12-25_14-30-00.txt`)
  - Writes all current logs
  - Continuously appends new logs as they arrive
  - Indicator turns **GOLD** showing "w WRITING"

- **Press `w` again**: Stops writing and closes the file
  - Indicator returns to normal "w write"

## 🔧 Auto-detected Interpreters

prox automatically detects the interpreter based on file extension:

| Extension | Interpreter |
|-----------|-------------|
| `.js`, `.mjs`, `.cjs` | `node` |
| `.ts` | `ts-node` |
| `.py` | `python` |
| `.rb` | `ruby` |
| `.sh` | `bash` |
| `.pl` | `perl` |
| `.php` | `php` |

Or specify manually:

```bash
prox start script.py --interpreter python3
```

## 📁 Data Storage

All process state and data is stored in `~/.prox/`:

```
~/.prox/
├── state.json          # Process configurations and status
├── logs/               # Process logs (stdout/stderr)
│   ├── myapp-out.log
│   └── myapp-err.log
├── pids/               # PID files
└── processes/          # Process definitions
```

## 🏗️ Architecture

Built with modern Go libraries:

- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** - TUI framework (Elm Architecture)
- **[Bubbles](https://github.com/charmbracelet/bubbles)** - TUI components (viewports, text inputs)
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Terminal styling and layout
- **[gopsutil](https://github.com/shirou/gopsutil)** - Cross-platform system and process metrics
- **[Cobra](https://github.com/spf13/cobra)** - CLI framework

## 🎯 Use Cases

- **Development**: Manage microservices locally
- **Production**: Simple process orchestration on single servers
- **Testing**: Run and monitor test suites
- **Scripts**: Manage background tasks and cron alternatives

## 🔍 Example Workflow

```bash
# Start your services from YAML
prox start-all

# Launch TUI to monitor everything
prox

# In the TUI:
# - Press 'Enter' on a process to see detailed metrics
# - Press 'l' to view live logs
# - Press 'w' to start continuous log writing
# - Press 'r' to restart a service
# - Press 'q' to quit

# Or use CLI commands
prox logs api --follow
prox restart worker
prox list
```

## 🛠️ Development

See [CLAUDE.md](./CLAUDE.md) for detailed development documentation.

### Build & Test

```bash
# Build
go build -o prox .

# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Install locally
go install
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

## 🙏 Acknowledgments

- Inspired by [pm2](https://pm2.keymetrics.io/)
- Built with the amazing [Charm](https://charm.sh/) libraries
- Community feedback and contributions

---

**Made with ❤️ and Go**
