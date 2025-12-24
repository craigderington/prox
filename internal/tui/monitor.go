package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	manager          *process.Manager
	storage          *storage.Storage
	collector        *process.MetricsCollector
	processes        []*process.Process
	metrics          map[string]*process.ProcessMetrics
	selected         int
	processName      string
	processViewport  viewport.Model // Viewport for process list
	logViewport      viewport.Model // Viewport for logs
	metricsViewport  viewport.Model // Viewport for metrics
	metadataViewport viewport.Model // Viewport for metadata
	logEntries       []logs.LogEntry
	totalLogsSeen    int
	followMode       bool
	focusedPanel     PanelFocus
	width            int
	height           int
}

// NewMonitorModel creates a new monitor view model
func NewMonitorModel(manager *process.Manager, storage *storage.Storage, collector *process.MetricsCollector, processes []*process.Process, metrics map[string]*process.ProcessMetrics, selected int, width, height int) MonitorModel {
	processName := ""
	if selected < len(processes) {
		processName = processes[selected].Name
	}

	// Calculate panel dimensions
	leftWidth := (width / 3) - 4
	rightWidth := (width * 2 / 3) - 4
	topHeight := (height * 2 / 3) - 5 // Reserve space for title + borders
	bottomHeight := (height / 3) - 5

	// Ensure minimum sizes
	if leftWidth < 10 {
		leftWidth = 10
	}
	if rightWidth < 10 {
		rightWidth = 10
	}
	if topHeight < 3 {
		topHeight = 3
	}
	if bottomHeight < 3 {
		bottomHeight = 3
	}

	// Initialize viewports for each panel
	processVP := viewport.New(leftWidth, topHeight)
	processVP.MouseWheelEnabled = true
	processVP.MouseWheelDelta = 1

	logVP := viewport.New(rightWidth, topHeight)
	logVP.MouseWheelEnabled = true
	logVP.MouseWheelDelta = 3

	metricsVP := viewport.New(leftWidth, bottomHeight)
	metricsVP.MouseWheelEnabled = true
	metricsVP.MouseWheelDelta = 1

	metadataVP := viewport.New(rightWidth, bottomHeight)
	metadataVP.MouseWheelEnabled = true
	metadataVP.MouseWheelDelta = 1

	return MonitorModel{
		manager:          manager,
		storage:          storage,
		collector:        collector,
		processes:        processes,
		metrics:          metrics,
		selected:         selected,
		processName:      processName,
		processViewport:  processVP,
		logViewport:      logVP,
		metricsViewport:  metricsVP,
		metadataViewport: metadataVP,
		logEntries:       []logs.LogEntry{},
		totalLogsSeen:    0,
		followMode:       true,
		focusedPanel:     FocusProcessList,
		width:            width,
		height:           height,
	}
}

// Init initializes the monitor view
func (m MonitorModel) Init() tea.Cmd {
	return tea.Batch(
		fetchProcesses(m.manager),
		loadLogsCmd(m.storage, m.processName, 100),
		collectMetrics(m.collector),
		tickCmd(), // Start the ticker for live updates
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
			switch m.focusedPanel {
			case FocusProcessList:
				// Navigate process list and update selection
				if m.selected < len(m.processes)-1 {
					m.selected++
					m.processName = m.processes[m.selected].Name
					m.updateProcessViewport()
					m.updateMetricsViewport()
					m.updateMetadataViewport()
					// Reload logs for new process
					m.logEntries = []logs.LogEntry{}
					m.totalLogsSeen = 0
					return m, loadLogsCmd(m.storage, m.processName, 100)
				}
			case FocusLogs:
				// Scroll logs viewport
				m.logViewport.LineDown(1)
				m.followMode = false
			case FocusMetrics:
				// Scroll metrics viewport
				m.metricsViewport.LineDown(1)
			case FocusMetadata:
				// Scroll metadata viewport
				m.metadataViewport.LineDown(1)
			}
			return m, nil

		case "k", "up":
			switch m.focusedPanel {
			case FocusProcessList:
				// Navigate process list and update selection
				if m.selected > 0 {
					m.selected--
					m.processName = m.processes[m.selected].Name
					m.updateProcessViewport()
					m.updateMetricsViewport()
					m.updateMetadataViewport()
					// Reload logs for new process
					m.logEntries = []logs.LogEntry{}
					m.totalLogsSeen = 0
					return m, loadLogsCmd(m.storage, m.processName, 100)
				}
			case FocusLogs:
				// Scroll logs viewport
				m.logViewport.LineUp(1)
				m.followMode = false
			case FocusMetrics:
				// Scroll metrics viewport
				m.metricsViewport.LineUp(1)
			case FocusMetadata:
				// Scroll metadata viewport
				m.metadataViewport.LineUp(1)
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

		// Recalculate viewport sizes for all panels
		leftWidth := (m.width / 3) - 4
		rightWidth := (m.width * 2 / 3) - 4
		topHeight := (m.height * 2 / 3) - 5
		bottomHeight := (m.height / 3) - 5

		// Ensure minimum sizes
		if leftWidth < 10 {
			leftWidth = 10
		}
		if rightWidth < 10 {
			rightWidth = 10
		}
		if topHeight < 3 {
			topHeight = 3
		}
		if bottomHeight < 3 {
			bottomHeight = 3
		}

		// Update all viewports
		m.processViewport.Width = leftWidth
		m.processViewport.Height = topHeight
		m.logViewport.Width = rightWidth
		m.logViewport.Height = topHeight
		m.metricsViewport.Width = leftWidth
		m.metricsViewport.Height = bottomHeight
		m.metadataViewport.Width = rightWidth
		m.metadataViewport.Height = bottomHeight

		// Update viewport contents
		m.updateProcessViewport()
		m.updateLogViewport()
		m.updateMetricsViewport()
		m.updateMetadataViewport()
		return m, nil

	case logEntriesMsg:
		if len(msg) > 0 {
			if len(m.logEntries) == 0 {
				// Initial load
				m.logEntries = msg
				m.totalLogsSeen = len(msg)
				m.updateLogViewport()
				// Ensure we scroll to bottom on initial load
				if m.followMode {
					m.logViewport.GotoBottom()
				}
				// Start continuous tailing after initial load
				return m, tailLogsCmd(m.storage, m.processName, m.totalLogsSeen)
			} else {
				// Incremental update
				m.logEntries = append(m.logEntries, msg...)
				m.totalLogsSeen += len(msg)

				// Cap at max lines
				if len(m.logEntries) > maxLogLines {
					m.logEntries = m.logEntries[len(m.logEntries)-maxLogLines:]
					// Don't change totalLogsSeen when capping
				}

				m.updateLogViewport()
				// Ensure we scroll to bottom when following
				if m.followMode {
					m.logViewport.GotoBottom()
				}
			}
		}
		// Continue tailing
		return m, tailLogsCmd(m.storage, m.processName, m.totalLogsSeen)

	case logTickMsg:
		// Continue tailing on tick
		return m, tailLogsCmd(m.storage, m.processName, m.totalLogsSeen)

	case tickMsg:
		// Refresh processes and metrics on each tick
		return m, tea.Batch(
			tickCmd(),
			fetchProcesses(m.manager),
			collectMetrics(m.collector),
		)

	case metricsMsg:
		m.metrics = msg
		m.updateMetricsViewport()
		m.updateMetadataViewport()
		m.updateProcessViewport() // Update process list to show live CPU/Mem
		return m, nil

	case processesMsg:
		m.processes = msg
		m.updateProcessViewport()
		m.updateMetadataViewport()
		return m, nil
	}

	return m, nil
}

// View renders the 4-panel monitor view
func (m MonitorModel) View() string {
	// Calculate viewport dimensions (content area inside borders)
	leftWidth := (m.width / 3) - 4
	rightWidth := (m.width * 2 / 3) - 4
	topHeight := (m.height * 2 / 3) - 5
	bottomHeight := (m.height / 3) - 5

	// Ensure minimum sizes
	if leftWidth < 10 {
		leftWidth = 10
	}
	if rightWidth < 10 {
		rightWidth = 10
	}
	if topHeight < 3 {
		topHeight = 3
	}
	if bottomHeight < 3 {
		bottomHeight = 3
	}

	// Render panels (these will add borders and padding)
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

	// Help text - show focused panel controls
	var helpText string
	switch m.focusedPanel {
	case FocusProcessList:
		helpText = "↑/↓: select process • tab: next panel • esc: back to dashboard"
	case FocusLogs:
		helpText = "↑/↓: scroll • g/G: top/bottom • f: toggle follow • tab: next panel • esc: back"
	default:
		helpText = "↑/↓: scroll • tab: next panel • esc: back to dashboard"
	}

	help := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(helpText)

	// Join everything
	return lipgloss.JoinVertical(
		lipgloss.Left,
		topRow,
		bottomRow,
		help,
	)
}

// renderProcessListPanel renders the process list (top-left)
func (m MonitorModel) renderProcessListPanel(width, height int) string {
	title := "Process List"
	if m.focusedPanel == FocusProcessList {
		title = "▶ Process List"
	}

	titleBar := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(title)

	// Highlight border if focused
	borderColor := colorBorder
	if m.focusedPanel == FocusProcessList {
		borderColor = colorPrimary
	}

	// Wrap viewport in a bordered box with title
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleBar,
		"",
		m.processViewport.View(),
	)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(content)
}

// renderLogsPanel renders the logs viewer (top-right)
func (m MonitorModel) renderLogsPanel(width, height int) string {
	titleText := fmt.Sprintf("Logs: %s", truncate(m.processName, 20))
	if m.focusedPanel == FocusLogs {
		titleText = "▶ " + titleText
	}

	followStatus := ""
	if m.followMode {
		followStatus = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Render(" [FOLLOW]")
	}

	titleBar := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(titleText + followStatus)

	// Highlight border if focused
	borderColor := colorBorder
	if m.focusedPanel == FocusLogs {
		borderColor = colorPrimary
	}

	// Wrap viewport in a bordered box with title
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleBar,
		"",
		m.logViewport.View(),
	)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(content)
}

// renderMetricsPanel renders custom metrics (bottom-left)
func (m MonitorModel) renderMetricsPanel(width, height int) string {
	titleText := "Custom Metrics"
	if m.focusedPanel == FocusMetrics {
		titleText = "▶ " + titleText
	}

	titleBar := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(titleText)

	// Highlight border if focused
	borderColor := colorBorder
	if m.focusedPanel == FocusMetrics {
		borderColor = colorPrimary
	}

	// Wrap viewport in a bordered box with title
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleBar,
		"",
		m.metricsViewport.View(),
	)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(content)
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

	titleBar := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(titleText)

	// Highlight border if focused
	borderColor := colorBorder
	if m.focusedPanel == FocusMetadata {
		borderColor = colorPrimary
	}

	// Wrap viewport in a bordered box with title
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleBar,
		"",
		m.metadataViewport.View(),
	)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Render(content)
}

// updateProcessViewport updates the process list viewport content
func (m *MonitorModel) updateProcessViewport() {
	var lines []string

	for i, proc := range m.processes {
		metrics := m.metrics[proc.ID]

		// Status symbol and color
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

		// Format line components
		name := truncate(proc.Name, 12)
		mem := "   -   "
		cpu := "  -  "
		if metrics != nil && proc.Status == process.StatusOnline {
			mem = fmt.Sprintf("%7s", truncate(process.FormatBytes(metrics.Memory), 7))
			cpu = fmt.Sprintf("%5.1f%%", metrics.CPU)
		}

		// Build the line with selection highlighting
		line := fmt.Sprintf("%s %s  %s  %s",
			lipgloss.NewStyle().Foreground(statusColor).Render(statusSymbol),
			lipgloss.NewStyle().Width(12).Render(name),
			mem,
			cpu,
		)

		// Highlight selected process
		if i == m.selected {
			line = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				Render("▶ " + line)
		} else {
			line = "  " + line
		}

		lines = append(lines, line)
	}

	if len(lines) == 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(colorMuted).
			Render("No processes"))
	}

	content := strings.Join(lines, "\n")
	m.processViewport.SetContent(content)
}

// updateLogViewport updates the log viewport content
func (m *MonitorModel) updateLogViewport() {
	var lines []string

	if len(m.logEntries) == 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(colorMuted).
			Render("No logs available"))
	} else {
		for _, entry := range m.logEntries {
			lines = append(lines, formatLogEntry(entry))
		}
	}

	content := strings.Join(lines, "\n")
	m.logViewport.SetContent(content)

	// Auto-scroll if in follow mode
	if m.followMode {
		m.logViewport.GotoBottom()
	}
}

// updateMetricsViewport updates the metrics viewport content
func (m *MonitorModel) updateMetricsViewport() {
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

			// Calculate progress bar width based on viewport width
			barWidth := m.metricsViewport.Width - 20
			if barWidth < 10 {
				barWidth = 10
			}
			if barWidth > 30 {
				barWidth = 30
			}

			// Create progress bars
			cpuBar := renderProgressBar(cpuPercent, barWidth)
			memBar := renderProgressBar(memPercent, barWidth)

			// Build content with styled text and progress bars
			// Don't wrap in another style - it would override the progress bar colors
			content = fmt.Sprintf("%s %5.1f%% %s\n%s %5.1f%% %s\n%s %s\n%s %d",
				lipgloss.NewStyle().Foreground(colorText).Render("CPU:     "),
				cpuPercent,
				cpuBar,
				lipgloss.NewStyle().Foreground(colorText).Render("Memory:  "),
				memPercent,
				memBar,
				lipgloss.NewStyle().Foreground(colorText).Render("Uptime:  "),
				lipgloss.NewStyle().Foreground(colorText).Render(process.FormatDuration(metrics.Uptime)),
				lipgloss.NewStyle().Foreground(colorText).Render("Restarts:"),
				proc.Restarts,
			)
		} else {
			content = lipgloss.NewStyle().
				Foreground(colorMuted).
				Render("Process not running")
		}
	} else {
		content = lipgloss.NewStyle().
			Foreground(colorMuted).
			Render("No process selected")
	}

	m.metricsViewport.SetContent(content)
}

// updateMetadataViewport updates the metadata viewport content
func (m *MonitorModel) updateMetadataViewport() {
	var proc *process.Process
	if m.selected < len(m.processes) {
		proc = m.processes[m.selected]
	}

	var content string
	if proc != nil {
		// Truncate long paths to fit in the viewport
		maxFieldWidth := m.metadataViewport.Width - 12
		if maxFieldWidth < 10 {
			maxFieldWidth = 10
		}

		content = lipgloss.NewStyle().Foreground(colorText).Render(
			fmt.Sprintf(
				"Name:     %s\nRestarts: %d\nStatus:   %s\nScript:   %s\nInterp:   %s\nPID:      %d\nCwd:      %s",
				truncate(proc.Name, maxFieldWidth),
				proc.Restarts,
				proc.Status,
				truncate(proc.Script, maxFieldWidth),
				func() string {
					if proc.Interpreter != "" {
						return proc.Interpreter
					}
					return "N/A"
				}(),
				proc.PID,
				truncate(proc.Cwd, maxFieldWidth),
			),
		)
	} else {
		content = lipgloss.NewStyle().
			Foreground(colorMuted).
			Render("No process selected")
	}

	m.metadataViewport.SetContent(content)
}
