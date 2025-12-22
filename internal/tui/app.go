package tui

import (
	"fmt"
	"time"

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
	viewState    string // "dashboard" or "monitor"
	monitorModel *MonitorModel
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
	return Model{
		manager:      manager,
		collector:    process.NewMetricsCollector(manager),
		storage:      storage,
		processes:    []*process.Process{},
		metrics:      make(map[string]*process.ProcessMetrics),
		selected:     0,
		viewState:    "dashboard",
		monitorModel: nil,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		fetchProcesses(m.manager),
		tickCmd(),
	)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window size for both views
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height

		// Propagate to monitor view if active
		if m.viewState == "monitor" && m.monitorModel != nil {
			updatedMonitor, cmd := m.monitorModel.Update(msg)
			m.monitorModel = &updatedMonitor
			return m, cmd
		}
		return m, nil
	}

	// Route messages based on current view
	if m.viewState == "monitor" {
		return m.updateMonitorView(msg)
	}

	// Dashboard view handling
	return m.updateDashboard(msg)
}

// updateDashboard handles dashboard-specific messages
func (m Model) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

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
			tickCmd(),
			fetchProcesses(m.manager),
		)

	case errMsg:
		m.err = error(msg)
		return m, nil
	}

	return m, nil
}

// updateMonitorView handles monitor view messages
func (m Model) updateMonitorView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.monitorModel == nil {
		m.viewState = "dashboard"
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			// Exit monitor view
			m.viewState = "dashboard"
			m.monitorModel = nil
			return m, nil
		}
	}

	// Forward all messages to monitor model
	updatedMonitor, cmd := m.monitorModel.Update(msg)
	m.monitorModel = &updatedMonitor
	return m, cmd
}

// View renders the TUI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	// Route to appropriate view
	if m.viewState == "monitor" && m.monitorModel != nil {
		return m.monitorModel.View()
	}

	return renderDashboard(m)
}

// Commands for async operations

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
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
