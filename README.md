# ⚡ prox

A modern process manager with a beautiful Terminal User Interface (TUI). Inspired by pm2, built with Go and Bubbletea.

## Features

✅ **Universal Process Management** - Run applications in any language (Node.js, Python, Go, Rust, Bash, etc.)
✅ **Beautiful TUI** - Real-time dashboard with live metrics
✅ **Smart Process Control** - Graceful shutdown, auto-restart, status tracking
✅ **Metrics Collection** - CPU, memory, and uptime monitoring
✅ **State Persistence** - Processes survive prox restarts
✅ **Keyboard Navigation** - Vim-like controls (hjkl, arrows)

## Installation

```bash
go build -o prox .
```

Or install globally:

```bash
go install
```

## Quick Start

### Launch TUI (Interactive Mode)

```bash
./prox
```

The TUI provides a real-time dashboard showing all managed processes with live metrics.

**Keyboard Shortcuts:**
- `↑/k` - Move selection up
- `↓/j` - Move selection down
- `r` - Restart selected process
- `s` - Stop selected process
- `d` - Delete selected process
- `R` - Refresh process list
- `q` - Quit

### CLI Mode

```bash
# Start a process
./prox start app.js --name my-app

# List all processes
./prox list

# Stop a process
./prox stop my-app

# Restart a process
./prox restart my-app
```

## Auto-detected Interpreters

prox automatically detects the interpreter based on file extension:

- `.js` → `node`
- `.py` → `python`
- `.rb` → `ruby`
- `.sh` → `bash`

Or specify manually:

```bash
./prox start script.js --interpreter node
```

## Process Management

All process state is stored in `~/.prox/`:

```
~/.prox/
├── state.json      # Process configurations and status
├── logs/           # Process logs (stdout/stderr)
├── pids/           # PID files
└── processes/      # Process definitions
```

## Architecture

Built with:
- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** - TUI framework (Elm Architecture)
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Terminal styling
- **[gopsutil](https://github.com/shirou/gopsutil)** - System metrics
- **[Cobra](https://github.com/spf13/cobra)** - CLI framework

## Development

See [CLAUDE.md](./CLAUDE.md) for detailed development documentation.

## License

MIT
