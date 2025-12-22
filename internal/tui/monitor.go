package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/craigderington/prox/internal/logs"
	"github.com/craigderington/prox/internal/process"
	"github.com/craigderington/prox/internal/storage"
)

// PanelFocus represents which panel is currently focused
type PanelFocus int

const (
	FocusProcessList PanelFocus = iota
	FocusLogs
	FocusMetrics
	FocusMetadata
)

// MonitorModel represents the 4-panel monitor view (pm2-monit style)
type MonitorModel struct {
	manager       *process.Manager
	storage       *storage.Storage
	collector     *process.MetricsCollector
	processes     []*process.Process
	metrics       map[string]*process.ProcessMetrics
	selected      int
	processName   string
	logViewport   viewport.Model
	logEntries    []logs.LogEntry
	logLineCount  int
	followMode    bool
	focusedPanel  PanelFocus
	width         int
	height        int
}

// NewMonitorModel creates a new monitor view model
func NewMonitorModel(manager *process.Manager, storage *storage.Storage, collector *process.MetricsCollector, processes []*process.Process, metrics map[string]*process.ProcessMetrics, selected int, width, height int) MonitorModel {
	processName := ""
	if selected < len(processes) {
		processName = processes[selected].Name
	}

	// Calculate viewport dimensions (right column, top section)
	logWidth := (width * 2 / 3) - 6   // Right column width
	logHeight := (height * 2 / 3) - 8 // Top section height

	vp := viewport.New(logWidth, logHeight)
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	return MonitorModel{
		manager:      manager,
		storage:      storage,
		collector:    collector,
		processes:    processes,
		metrics:      metrics,
		selected:     selected,
		processName:  processName,
		logViewport:  vp,
		logEntries:   []logs.LogEntry{},
		logLineCount: 0,
		followMode:   true,
		focusedPanel: FocusLogs, // Start with logs focused
		width:        width,
		height:       height,
	}
}

// Init initializes the monitor view
func (m MonitorModel) Init() tea.Cmd {
	return tea.Batch(
		loadLogsCmd(m.storage, m.processName, 100),
		collectMetrics(m.collector),
	)
}

// Update handles messages for the monitor view
func (m MonitorModel) Update(msg tea.Msg) (MonitorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			// Cycle through panels
			m.focusedPanel = (m.focusedPanel + 1) % 4
			return m, nil

		case "j", "down":
			if m.focusedPanel == FocusProcessList {
				// Navigate process list
				if m.selected < len(m.processes)-1 {
					m.selected++
					// Update to new process
					if m.selected < len(m.processes) {
						m.processName = m.processes[m.selected].Name
						// Reload logs for new process
						m.logEntries = []logs.LogEntry{}
						m.logLineCount = 0
						return m, loadLogsCmd(m.storage, m.processName, 100)
					}
				}
			} else if m.focusedPanel == FocusLogs {
				// Scroll logs down
				m.logViewport.LineDown(1)
				m.followMode = false
			}
			return m, nil

		case "k", "up":
			if m.focusedPanel == FocusProcessList {
				// Navigate process list
				if m.selected > 0 {
					m.selected--
					// Update to new process
					if m.selected < len(m.processes) {
						m.processName = m.processes[m.selected].Name
						// Reload logs for new process
						m.logEntries = []logs.LogEntry{}
						m.logLineCount = 0
						return m, loadLogsCmd(m.storage, m.processName, 100)
					}
				}
			} else if m.focusedPanel == FocusLogs {
				// Scroll logs up
				m.logViewport.LineUp(1)
				m.followMode = false
			}
			return m, nil

		case "g":
			if m.focusedPanel == FocusLogs {
				m.logViewport.GotoTop()
				m.followMode = false
			}
			return m, nil

		case "G":
			if m.focusedPanel == FocusLogs {
				m.logViewport.GotoBottom()
				m.followMode = true
			}
			return m, nil

		case "f":
			if m.focusedPanel == FocusLogs {
				m.followMode = !m.followMode
				if m.followMode {
					m.logViewport.GotoBottom()
				}
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Recalculate viewport size
		logWidth := (m.width * 2 / 3) - 6
		logHeight := (m.height * 2 / 3) - 8
		m.logViewport.Width = logWidth
		m.logViewport.Height = logHeight
		m.updateLogViewport()
		return m, nil

	case logEntriesMsg:
		if len(m.logEntries) == 0 {
			// Initial load
			m.logEntries = msg
			m.logLineCount = len(msg)
			m.updateLogViewport()
			return m, tailLogsCmd(m.storage, m.processName, m.logLineCount)
		} else {
			// Incremental update
			m.logEntries = append(m.logEntries, msg...)
			m.logLineCount = len(m.logEntries)

			// Cap at max lines
			if len(m.logEntries) > maxLogLines {
				m.logEntries = m.logEntries[len(m.logEntries)-maxLogLines:]
			}

			m.updateLogViewport()
			return m, tailLogsCmd(m.storage, m.processName, m.logLineCount)
		}

	case logTickMsg:
		return m, tailLogsCmd(m.storage, m.processName, m.logLineCount)

	case metricsMsg:
		m.metrics = msg
		return m, collectMetrics(m.collector)

	case processesMsg:
		m.processes = msg
		return m, nil
	}

	return m, nil
}

// View renders the 4-panel monitor view
func (m MonitorModel) View() string {
	// Calculate panel dimensions
	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth - 2
	topHeight := (m.height * 2 / 3)
	bottomHeight := m.height - topHeight - 2

	// Render panels
	processListPanel := m.renderProcessListPanel(leftWidth, topHeight)
	logsPanel := m.renderLogsPanel(rightWidth, topHeight)
	metricsPanel := m.renderMetricsPanel(leftWidth, bottomHeight)
	metadataPanel := m.renderMetadataPanel(rightWidth, bottomHeight)

	// Top row
	topRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		processListPanel,
		logsPanel,
	)

	// Bottom row
	bottomRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		metricsPanel,
		metadataPanel,
	)

	// Join rows
	return lipgloss.JoinVertical(
		lipgloss.Left,
		topRow,
		bottomRow,
	)
}

// renderProcessListPanel renders the process list (top-left)
func (m MonitorModel) renderProcessListPanel(width, height int) string {
	var b strings.Builder

	title := "Process List"
	if m.focusedPanel == FocusProcessList {
		title = "▶ Process List"
	}

	b.WriteString(lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(title))
	b.WriteString("\n")

	for i, proc := range m.processes {
		metrics := m.metrics[proc.ID]

		// Selection indicator
		indicator := " "
		if i == m.selected {
			indicator = "▶"
		}

		// Status symbol
		statusSymbol := "○"
		statusColor := colorMuted
		switch proc.Status {
		case process.StatusOnline:
			statusSymbol = "●"
			statusColor = colorSuccess
		case process.StatusErrored:
			statusSymbol = "✗"
			statusColor = colorDanger
		case process.StatusRestarting:
			statusSymbol = "↻"
			statusColor = colorWarning
		}

		// Format line
		name := truncate(proc.Name, 12)
		mem := "-"
		cpu := "-"
		if metrics != nil && proc.Status == process.StatusOnline {
			mem = truncate(process.FormatBytes(metrics.Memory), 7)
			cpu = fmt.Sprintf("%.0f%%", metrics.CPU)
		}

		line := fmt.Sprintf("%s %d) %s  Mem: %s  CPU: %s  %s",
			indicator,
			i+1,
			name,
			mem,
			cpu,
			lipgloss.NewStyle().Foreground(statusColor).Render(statusSymbol+" "+string(proc.Status)),
		)

		b.WriteString(line)
		b.WriteString("\n")
	}

	content := b.String()

	// Highlight border if focused
	borderColor := colorBorder
	if m.focusedPanel == FocusProcessList {
		borderColor = colorPrimary
	}

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(height - 2).
		Padding(1).
		Render(content)
}

// renderLogsPanel renders the logs viewer (top-right)
func (m MonitorModel) renderLogsPanel(width, height int) string {
	titleText := fmt.Sprintf("Logs: %s", m.processName)
	if m.focusedPanel == FocusLogs {
		titleText = "▶ " + titleText
	}

	followStatus := ""
	if m.followMode {
		followStatus = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Render(" [FOLLOW ON]")
	}

	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(titleText + followStatus)

	var content string
	if len(m.logEntries) == 0 {
		content = lipgloss.NewStyle().
			Foreground(colorMuted).
			Render("No logs available")
	} else {
		content = m.logViewport.View()
	}

	innerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		content,
	)

	// Highlight border if focused
	borderColor := colorBorder
	if m.focusedPanel == FocusLogs {
		borderColor = colorPrimary
	}

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(height - 2).
		Padding(1).
		Render(innerContent)
}

// renderMetricsPanel renders custom metrics (bottom-left)
func (m MonitorModel) renderMetricsPanel(width, height int) string {
	titleText := "Custom Metrics"
	if m.focusedPanel == FocusMetrics {
		titleText = "▶ " + titleText
	}

	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(titleText)

	var proc *process.Process
	if m.selected < len(m.processes) {
		proc = m.processes[m.selected]
	}

	var content string
	if proc != nil {
		metrics := m.metrics[proc.ID]
		if metrics != nil && proc.Status == process.StatusOnline {
			// Calculate percentages
			cpuPercent := metrics.CPU
			memPercent := metrics.MemoryPercent

			// Create progress bars (25 chars wide)
			cpuBar := renderProgressBar(cpuPercent, 25)
			memBar := renderProgressBar(memPercent, 25)

			content = fmt.Sprintf("CPU Usage:  %5.1f%% %s\nMemory:     %5.1f%% %s\nUptime:            %s\nRestarts:          %d",
				cpuPercent,
				cpuBar,
				memPercent,
				memBar,
				process.FormatDuration(metrics.Uptime),
				proc.Restarts,
			)
		} else {
			content = "Process not running"
		}
	} else {
		content = "No process selected"
	}

	innerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		lipgloss.NewStyle().Foreground(colorText).Render(content),
	)

	// Highlight border if focused
	borderColor := colorBorder
	if m.focusedPanel == FocusMetrics {
		borderColor = colorPrimary
	}

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(height - 2).
		Padding(1).
		Render(innerContent)
}

// renderProgressBar creates a colored progress bar like htop
func renderProgressBar(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	// Calculate filled portion
	filled := int(percent / 100.0 * float64(width))
	empty := width - filled

	// Choose color based on percentage
	var barColor lipgloss.Color
	if percent < 50 {
		barColor = colorSuccess // Green
	} else if percent < 80 {
		barColor = colorWarning // Yellow
	} else {
		barColor = colorDanger // Red
	}

	// Build bar with filled and empty sections
	filledBar := lipgloss.NewStyle().
		Foreground(barColor).
		Render(strings.Repeat("█", filled))

	emptyBar := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(strings.Repeat("░", empty))

	return filledBar + emptyBar
}

// renderMetadataPanel renders process metadata (bottom-right)
func (m MonitorModel) renderMetadataPanel(width, height int) string {
	titleText := "Metadata"
	if m.focusedPanel == FocusMetadata {
		titleText = "▶ " + titleText
	}

	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(titleText)

	var proc *process.Process
	if m.selected < len(m.processes) {
		proc = m.processes[m.selected]
	}

	var content string
	if proc != nil {
		content = fmt.Sprintf(
			"App Name:      %s\nRestarts:      %d\nStatus:        %s\nScript path:   %s\nInterpreter:   %s\nPID:           %d\nWorking dir:   %s",
			proc.Name,
			proc.Restarts,
			proc.Status,
			proc.Script,
			func() string {
				if proc.Interpreter != "" {
					return proc.Interpreter
				}
				return "N/A"
			}(),
			proc.PID,
			proc.Cwd,
		)
	} else {
		content = "No process selected"
	}

	innerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		lipgloss.NewStyle().Foreground(colorText).Render(content),
	)

	// Highlight border if focused
	borderColor := colorBorder
	if m.focusedPanel == FocusMetadata {
		borderColor = colorPrimary
	}

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(height - 2).
		Padding(1).
		Render(innerContent)
}

// updateLogViewport updates the log viewport content
func (m *MonitorModel) updateLogViewport() {
	lines := make([]string, len(m.logEntries))

	for i, entry := range m.logEntries {
		lines[i] = formatLogEntry(entry)
	}

	content := strings.Join(lines, "\n")
	m.logViewport.SetContent(content)

	// Auto-scroll if in follow mode
	if m.followMode {
		m.logViewport.GotoBottom()
	}
}
