package tui

import (
	"fmt"
	"strings"
	"time"

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
	cpuHistory       []float64
	memHistory       []float64
	netHistory       []float64
	maxHistory       int
	prevNetSent      uint64
	prevNetRecv      uint64
	lastMetricsTime  time.Time
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
	// Top row: Process List (left) | Logs (right)
	processVP := viewport.New(leftWidth, topHeight)
	processVP.MouseWheelEnabled = true
	processVP.MouseWheelDelta = 1

	logVP := viewport.New(rightWidth, topHeight)
	logVP.MouseWheelEnabled = true
	logVP.MouseWheelDelta = 3

	// Bottom row: Metadata (left) | Metrics (right) - aligned with top
	metadataVP := viewport.New(leftWidth, bottomHeight)
	metadataVP.MouseWheelEnabled = true
	metadataVP.MouseWheelDelta = 1

	metricsVP := viewport.New(rightWidth, bottomHeight)
	metricsVP.MouseWheelEnabled = true
	metricsVP.MouseWheelDelta = 1

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
		cpuHistory:       make([]float64, 0, 100),
		memHistory:       make([]float64, 0, 100),
		netHistory:       make([]float64, 0, 100),
		maxHistory:       100,
		prevNetSent:      0,
		prevNetRecv:      0,
		lastMetricsTime:  time.Now(),
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

		case "h", "left":
			// Move focus to left panel
			switch m.focusedPanel {
			case FocusLogs:
				m.focusedPanel = FocusProcessList
			case FocusMetrics:
				m.focusedPanel = FocusMetadata
			}
			return m, nil

		case "l", "right":
			// Move focus to right panel
			switch m.focusedPanel {
			case FocusProcessList:
				m.focusedPanel = FocusLogs
			case FocusMetadata:
				m.focusedPanel = FocusMetrics
			}
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

		case "w":
			// Write logs to disk (only when logs panel is focused)
			if m.focusedPanel == FocusLogs {
				return m, writeLogsToFile(m.processName, m.logEntries)
			}
			return m, nil

		case "r":
			// Restart selected process
			if m.selected < len(m.processes) {
				proc := m.processes[m.selected]
				if err := m.manager.Restart(proc.ID); err == nil {
					m.storage.SaveState(m.manager.List())
				}
				return m, tea.Batch(
					fetchProcesses(m.manager),
					collectMetrics(m.collector),
				)
			}
			return m, nil

		case "s":
			// Stop selected process
			if m.selected < len(m.processes) {
				proc := m.processes[m.selected]
				if err := m.manager.Stop(proc.ID); err == nil {
					m.storage.SaveState(m.manager.List())
				}
				return m, tea.Batch(
					fetchProcesses(m.manager),
					collectMetrics(m.collector),
				)
			}
			return m, nil

		case "d":
			// Delete selected process
			if m.selected < len(m.processes) {
				proc := m.processes[m.selected]
				if err := m.manager.Delete(proc.ID); err == nil {
					m.storage.SaveState(m.manager.List())
					// Adjust selection if needed
					if m.selected >= len(m.processes)-1 && m.selected > 0 {
						m.selected--
					}
					// Update process name
					if m.selected < len(m.processes) {
						m.processName = m.processes[m.selected].Name
					}
				}
				return m, tea.Batch(
					fetchProcesses(m.manager),
					collectMetrics(m.collector),
				)
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Recalculate viewport sizes for all panels
		availableHeight := m.height - 4
		leftWidth := (m.width / 3) - 6
		rightWidth := (m.width * 2 / 3) - 6
		topHeight := (availableHeight * 2 / 3) - 6
		bottomHeight := (availableHeight / 3) - 4

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
		// Top row: Process List (left) | Logs (right)
		m.processViewport.Width = leftWidth
		m.processViewport.Height = topHeight
		m.logViewport.Width = rightWidth
		m.logViewport.Height = topHeight
		// Bottom row: Metadata (left) | Metrics (right) - aligned with top
		m.metadataViewport.Width = leftWidth
		m.metadataViewport.Height = bottomHeight
		m.metricsViewport.Width = rightWidth
		m.metricsViewport.Height = bottomHeight

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
		// Add to history for sparklines
		if m.selected < len(m.processes) {
			proc := m.processes[m.selected]
			if metrics := msg[proc.ID]; metrics != nil {
				m.cpuHistory = append(m.cpuHistory, metrics.CPU)
				m.memHistory = append(m.memHistory, metrics.MemoryPercent)
				if len(m.cpuHistory) > m.maxHistory {
					m.cpuHistory = m.cpuHistory[1:]
				}
				if len(m.memHistory) > m.maxHistory {
					m.memHistory = m.memHistory[1:]
				}

				// Calculate network rate (bytes/sec)
				now := time.Now()
				timeDiff := now.Sub(m.lastMetricsTime).Seconds()
				if timeDiff > 0 {
					netRate := float64(metrics.NetSent+metrics.NetRecv-m.prevNetSent-m.prevNetRecv) / timeDiff
					m.netHistory = append(m.netHistory, netRate)
					if len(m.netHistory) > m.maxHistory {
						m.netHistory = m.netHistory[1:]
					}
					m.prevNetSent = metrics.NetSent
					m.prevNetRecv = metrics.NetRecv
					m.lastMetricsTime = now
				}
			}
		}
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
	// Simple text header without borders
	header := m.renderSimpleHeader()

	// Calculate viewport dimensions (content area inside borders)
	// Account for header (1 line), help (1 line) and margins
	availableHeight := m.height - 4
	// Account for spacing between panels
	leftWidth := (m.width / 3) - 6
	rightWidth := (m.width * 2 / 3) - 6
	topHeight := (availableHeight * 2 / 3) - 6
	bottomHeight := (availableHeight / 3) - 4

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
	// Top row: Process List (left 1/3) | Logs (right 2/3)
	processListPanel := m.renderProcessListPanel(leftWidth, topHeight)
	logsPanel := m.renderLogsPanel(rightWidth, topHeight)
	// Bottom row: Metadata (left 1/3) | Metrics (right 2/3) - aligned with top row
	metadataPanel := m.renderMetadataPanel(leftWidth, bottomHeight)
	metricsPanel := m.renderMetricsPanel(rightWidth, bottomHeight)

	// Top row with spacing
	topRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		processListPanel,
		" ", // Spacing between panels
		logsPanel,
	)

	// Bottom row with spacing
	bottomRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		metadataPanel,
		" ", // Spacing between panels
		metricsPanel,
	)

	// Help text - show focused panel controls
	var helpText string
	switch m.focusedPanel {
	case FocusProcessList:
		helpText = "↑/↓/j/k: select • h/l: panels • r: restart • s: stop • d: delete • esc: back"
	case FocusLogs:
		helpText = "↑/↓/j/k: scroll • g/G: top/bottom • f: follow • w: write • h/l: panels • r/s/d: controls • esc: back"
	default:
		helpText = "↑/↓/j/k: scroll • h/l: panels • r: restart • s: stop • d: delete • esc: back"
	}

	help := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(helpText)

	// Build the view properly using Lipgloss
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		topRow,
		bottomRow,
		help,
	)

	// Apply proper margins/padding
	return lipgloss.NewStyle().
		MarginTop(1).
		MarginBottom(1).
		Render(content)
}

// renderSimpleHeader renders a simple text header without borders
func (m MonitorModel) renderSimpleHeader() string {
	title := titleStyle.Render("⚡ prox monitor")

	// Show selected process name
	processInfo := ""
	if m.selected < len(m.processes) {
		processInfo = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Render(fmt.Sprintf("Monitoring: %s", m.processName))
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		title,
		"  ",
		processInfo,
	)
}

// renderMonitorHeader renders the header for monitor view
func (m MonitorModel) renderMonitorHeader() string {
	title := titleStyle.Render("⚡ prox monitor")

	// Show selected process name
	processInfo := ""
	if m.selected < len(m.processes) {
		processInfo = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Render(fmt.Sprintf("Monitoring: %s", m.processName))
	}

	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		title,
		"  ",
		processInfo,
	)

	// Wrap in a border spanning the full terminal width
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Width(m.width - 2).
		Render(header)
}

// renderProcessListPanel renders the process list (top-left)
func (m MonitorModel) renderProcessListPanel(width, height int) string {
	title := "📋 Process List"
	if m.focusedPanel == FocusProcessList {
		title = "▶ 📋 Process List"
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
	titleText := fmt.Sprintf("📜 Logs: %s", truncate(m.processName, 20))
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

// renderMetricsPanel renders custom metrics (bottom-right)
func (m MonitorModel) renderMetricsPanel(width, height int) string {
	titleText := "📈 Key CPU & Memory Metrics"
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

// renderWaveGraph creates a sparkline graph using block characters (like lazydocker)
// Newest data appears on the RIGHT, oldest on the LEFT (flows right to left)
func renderWaveGraph(data []float64, width int) string {
	if len(data) == 0 {
		return strings.Repeat("▁", width)
	}

	// Find min and max
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Avoid division by zero
	if max == min {
		max = min + 1
	}

	// Block characters for sparkline (8 levels)
	chars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	var wave strings.Builder

	// If we have less data than width, pad on the left with baseline
	if len(data) < width {
		padding := width - len(data)
		for i := 0; i < padding; i++ {
			wave.WriteString("▁")
		}
	}

	// Determine how many data points to show
	startIdx := 0
	if len(data) > width {
		// Show the most recent 'width' points
		startIdx = len(data) - width
	}

	// Render data points from oldest (left) to newest (right)
	for i := startIdx; i < len(data); i++ {
		// Normalize to 0-1
		normalized := (data[i] - min) / (max - min)
		// Map to character index (0-7 for 8 levels)
		charIndex := int(normalized * float64(len(chars)-1))
		if charIndex < 0 {
			charIndex = 0
		}
		if charIndex >= len(chars) {
			charIndex = len(chars) - 1
		}
		wave.WriteString(chars[charIndex])
	}

	return wave.String()
}

// renderMetadataPanel renders process metadata (bottom-left)
func (m MonitorModel) renderMetadataPanel(width, height int) string {
	titleText := "ℹ️  Metadata"
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
			name,
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

			// Calculate sparkline width - use full viewport width
			sparkWidth := m.metricsViewport.Width
			if sparkWidth < 20 {
				sparkWidth = 20
			}

			// Create wave graphs
			cpuSpark := renderWaveGraph(m.cpuHistory, sparkWidth)
			memSpark := renderWaveGraph(m.memHistory, sparkWidth)
			netSpark := renderWaveGraph(m.netHistory, sparkWidth)

			// Format labels
			cpuLabel := fmt.Sprintf("CPU: %.1f%%", cpuPercent)
			memLabel := fmt.Sprintf("Mem: %.1f%%", memPercent)
			netRate := float64(metrics.NetSent + metrics.NetRecv)
			if len(m.netHistory) > 0 {
				netRate = m.netHistory[len(m.netHistory)-1]
			}
			netLabel := fmt.Sprintf("Net: %s/s", process.FormatBytes(uint64(netRate)))

			// Build content with ONLY full-width sparklines
			content = fmt.Sprintf("%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s %s\n%s %d",
				lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(cpuLabel),
				lipgloss.NewStyle().Foreground(colorSuccess).Render(cpuSpark),
				lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render(memLabel),
				lipgloss.NewStyle().Foreground(colorWarning).Render(memSpark),
				lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(netLabel),
				lipgloss.NewStyle().Foreground(colorPrimary).Render(netSpark),
				lipgloss.NewStyle().Foreground(colorText).Render("Uptime:"),
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

		lines := []string{
			fmt.Sprintf("Name:     %s", truncate(proc.Name, maxFieldWidth)),
			fmt.Sprintf("Restarts: %d", proc.Restarts),
			fmt.Sprintf("Status:   %s", proc.Status),
			fmt.Sprintf("Script:   %s", truncate(proc.Script, maxFieldWidth)),
			fmt.Sprintf("Interp:   %s", func() string {
				if proc.Interpreter != "" {
					return proc.Interpreter
				}
				return "N/A"
			}()),
			fmt.Sprintf("PID:      %d", proc.PID),
			fmt.Sprintf("Cwd:      %s", truncate(proc.Cwd, maxFieldWidth)),
		}

		// Pad each line to the full viewport width
		paddedLines := make([]string, len(lines))
		for i, line := range lines {
			paddedLines[i] = lipgloss.NewStyle().Width(m.metadataViewport.Width).Render(line)
		}

		content = lipgloss.NewStyle().Foreground(colorText).Render(strings.Join(paddedLines, "\n"))
	} else {
		content = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(m.metadataViewport.Width).
			Render("No process selected")
	}

	m.metadataViewport.SetContent(content)
}
