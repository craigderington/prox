package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/craigderington/prox/internal/process"
	"github.com/craigderington/prox/internal/storage"
)

// Model represents the TUI application state
type Model struct {
	manager      *process.Manager
	collector    *process.MetricsCollector
	storage      *storage.Storage
	processes    []*process.Process
	metrics      map[string]*process.ProcessMetrics
	selected     int
	width        int
	height       int
	err          error
	viewState    string // "dashboard", "monitor", or "logs"
	monitorModel *MonitorModel
	logsModel    *LogsModel
	startInput   textinput.Model
	inputMode    bool // true when user is typing in the start input
	pollInterval int  // metrics polling interval in seconds
}

// Message types for async operations
type (
	processesMsg []*process.Process
	metricsMsg   map[string]*process.ProcessMetrics
	tickMsg      time.Time
	errMsg       error
)

// NewModel creates a new TUI model
func NewModel(manager *process.Manager, storage *storage.Storage) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter command to start (e.g., python app.py --name my-worker, node server.js, ./myapp)"
	ti.CharLimit = 200
	ti.Width = 80

	// Load metrics configuration
	metricsConfig, _ := storage.LoadMetricsConfig()

	return Model{
		manager:      manager,
		collector:    process.NewMetricsCollector(manager),
		storage:      storage,
		processes:    []*process.Process{},
		metrics:      make(map[string]*process.ProcessMetrics),
		selected:     0,
		viewState:    "dashboard",
		monitorModel: nil,
		startInput:   ti,
		inputMode:    false,
		pollInterval: metricsConfig.PollIntervalSeconds,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchProcesses(m.manager),
		tickCmd(m.pollInterval),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size for both views
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height

		// Propagate to active view
		if m.viewState == "monitor" && m.monitorModel != nil {
			updatedMonitor, cmd := m.monitorModel.Update(msg)
			m.monitorModel = &updatedMonitor
			return m, cmd
		}
		if m.viewState == "logs" && m.logsModel != nil {
			updatedLogs, cmd := m.logsModel.Update(msg)
			m.logsModel = &updatedLogs
			return m, cmd
		}
		return m, nil
	}

	// Route messages based on current view
	if m.viewState != "dashboard" {
		return m.updateActiveView(msg)
	}

	// Dashboard view handling
	return m.updateDashboard(msg)
}

// updateDashboard handles dashboard-specific messages
func (m Model) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If in input mode, handle differently
		if m.inputMode {
			switch msg.String() {
			case "esc":
				// Exit input mode
				m.inputMode = false
				m.startInput.Blur()
				m.startInput.SetValue("")
				return m, nil

			case "enter":
				// Start process with the entered command
				command := strings.TrimSpace(m.startInput.Value())
				if command != "" {
					m.inputMode = false
					m.startInput.Blur()
					m.startInput.SetValue("")
					return m, startProcess(m.manager, command)
				}
				return m, nil

			default:
				// Forward key to textinput
				var cmd tea.Cmd
				m.startInput, cmd = m.startInput.Update(msg)
				return m, cmd
			}
		}

		// Normal mode (not in input mode)
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "n":
			// Enter input mode to start a new process
			m.inputMode = true
			m.startInput.Focus()
			return m, nil

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil

		case "down", "j":
			if m.selected < len(m.processes)-1 {
				m.selected++
			}
			return m, nil

		case "enter":
			// Open monitor view for selected process
			if m.selected < len(m.processes) {
				monitorModel := NewMonitorModel(m.manager, m.storage, m.collector, m.processes, m.metrics, m.selected, m.width, m.height)
				m.monitorModel = &monitorModel
				m.viewState = "monitor"
				return m, monitorModel.Init()
			}
			return m, nil

		case "r":
			// Restart selected process
			if m.selected < len(m.processes) {
				proc := m.processes[m.selected]
				return m, restartProcess(m.manager, proc.Name)
			}
			return m, nil

		case "s":
			// Stop selected process
			if m.selected < len(m.processes) {
				proc := m.processes[m.selected]
				return m, stopProcess(m.manager, proc.Name)
			}
			return m, nil

		case "d":
			// Delete selected process
			if m.selected < len(m.processes) {
				proc := m.processes[m.selected]
				return m, deleteProcess(m.manager, proc.Name)
			}
			return m, nil

		case "l":
			// Open logs view for selected process
			if m.selected < len(m.processes) {
				proc := m.processes[m.selected]
				logsModel := NewLogsModel(m.manager, m.storage, proc.Name, m.width, m.height)
				m.logsModel = &logsModel
				m.viewState = "logs"
				return m, logsModel.Init()
			}
			return m, nil

		case "R":
			// Refresh
			return m, fetchProcesses(m.manager)
		}

	case processesMsg:
		m.processes = msg
		// Adjust selection if needed
		if m.selected >= len(m.processes) {
			m.selected = len(m.processes) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		return m, collectMetrics(m.collector)

	case metricsMsg:
		m.metrics = msg
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			tickCmd(m.pollInterval),
			fetchProcesses(m.manager),
		)

	case errMsg:
		m.err = error(msg)
		return m, nil
	}

	return m, nil
}

// updateActiveView handles active view messages
func (m Model) updateActiveView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.viewState {
	case "monitor":
		if m.monitorModel == nil {
			m.viewState = "dashboard"
			return m, nil
		}
	case "logs":
		if m.logsModel == nil {
			m.viewState = "dashboard"
			return m, nil
		}
	default:
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			// Exit active view
			m.viewState = "dashboard"
			m.monitorModel = nil
			m.logsModel = nil
			return m, nil
		}
	}

	// Forward messages to active model
	switch m.viewState {
	case "monitor":
		updatedMonitor, cmd := m.monitorModel.Update(msg)
		m.monitorModel = &updatedMonitor
		return m, cmd
	case "logs":
		updatedLogs, cmd := m.logsModel.Update(msg)
		m.logsModel = &updatedLogs
		return m, cmd
	}

	return m, nil
}

// View renders the TUI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	// Route to appropriate view
	switch m.viewState {
	case "monitor":
		if m.monitorModel != nil {
			return m.monitorModel.View()
		}
	case "logs":
		if m.logsModel != nil {
			return m.logsModel.View()
		}
	}

	return renderDashboard(m)
}

// Commands for async operations

func tickCmd(intervalSeconds ...int) tea.Cmd {
	seconds := 2 // Default to 2 seconds
	if len(intervalSeconds) > 0 && intervalSeconds[0] > 0 {
		seconds = intervalSeconds[0]
	}
	return tea.Tick(time.Duration(seconds)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchProcesses(manager *process.Manager) tea.Cmd {
	return func() tea.Msg {
		return processesMsg(manager.List())
	}
}

func collectMetrics(collector *process.MetricsCollector) tea.Cmd {
	return func() tea.Msg {
		return metricsMsg(collector.CollectAllMetrics())
	}
}

func restartProcess(manager *process.Manager, name string) tea.Cmd {
	return func() tea.Msg {
		err := manager.Restart(name)
		if err != nil {
			return errMsg(err)
		}
		return fetchProcesses(manager)()
	}
}

func stopProcess(manager *process.Manager, name string) tea.Cmd {
	return func() tea.Msg {
		err := manager.Stop(name)
		if err != nil {
			return errMsg(err)
		}
		return fetchProcesses(manager)()
	}
}

func deleteProcess(manager *process.Manager, name string) tea.Cmd {
	return func() tea.Msg {
		err := manager.Delete(name)
		if err != nil {
			return errMsg(err)
		}
		return fetchProcesses(manager)()
	}
}

func startProcess(manager *process.Manager, command string) tea.Cmd {
	return func() tea.Msg {
		// Parse the command to extract script, name flag, and args
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return errMsg(fmt.Errorf("empty command"))
		}

		script := parts[0]
		var args []string
		var customName string

		// Parse for --name flag and remaining args
		i := 1
		for i < len(parts) {
			if parts[i] == "--name" || parts[i] == "-n" {
				// Next part is the name
				if i+1 < len(parts) {
					customName = parts[i+1]
					i += 2
					continue
				}
				i++
			} else {
				args = append(args, parts[i])
				i++
			}
		}

		// Generate a name from the script if not provided
		name := customName
		if name == "" {
			name = script
			if strings.Contains(name, "/") {
				nameParts := strings.Split(name, "/")
				name = nameParts[len(nameParts)-1]
			}
		}

		// Auto-detect interpreter based on file extension
		interpreter := detectInterpreter(script)

		// Create config and start the process with auto-detected interpreter
		config := process.ProcessConfig{
			Name:        name,
			Script:      script,
			Interpreter: interpreter,
			Args:        args,
		}

		_, err := manager.Start(config)
		if err != nil {
			return errMsg(err)
		}
		return fetchProcesses(manager)()
	}
}

// detectInterpreter detects the interpreter based on file extension
func detectInterpreter(script string) string {
	ext := filepath.Ext(script)
	switch ext {
	case ".js":
		return "node"
	case ".py":
		return "python"
	case ".rb":
		return "ruby"
	case ".sh":
		return "bash"
	default:
		return "" // Direct execution
	}
}
