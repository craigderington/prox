package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/craigderington/prox/internal/logs"
	"github.com/craigderington/prox/internal/process"
	"github.com/craigderington/prox/internal/storage"
)

const (
	maxLogLines = 1000 // Cap to prevent memory bloat
)

// LogsModel represents the logs view state
type LogsModel struct {
	manager     *process.Manager
	storage     *storage.Storage
	processName string
	viewport    viewport.Model
	entries     []logs.LogEntry
	lineCount   int // Track line count for incremental updates
	followMode  bool
	loading     bool
	err         error
}

// NewLogsModel creates a new logs view model
func NewLogsModel(manager *process.Manager, storage *storage.Storage, processName string, width, height int) LogsModel {
	vp := viewport.New(width-4, height-8) // Leave room for header/footer
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	return LogsModel{
		manager:     manager,
		storage:     storage,
		processName: processName,
		viewport:    vp,
		entries:     []logs.LogEntry{},
		lineCount:   0,
		followMode:  true,
		loading:     true,
		err:         nil,
	}
}

// Init initializes the logs view
func (m LogsModel) Init() tea.Cmd {
	return loadLogsCmd(m.storage, m.processName, 100)
}

// Update handles messages for the logs view
func (m LogsModel) Update(msg tea.Msg) (LogsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.viewport.LineDown(1)
			m.followMode = false
			return m, nil

		case "k", "up":
			m.viewport.LineUp(1)
			m.followMode = false
			return m, nil

		case "d":
			m.viewport.HalfViewDown()
			m.followMode = false
			return m, nil

		case "u":
			m.viewport.HalfViewUp()
			m.followMode = false
			return m, nil

		case "g":
			m.viewport.GotoTop()
			m.followMode = false
			return m, nil

		case "G":
			m.viewport.GotoBottom()
			m.followMode = true
			return m, nil

		case "f":
			m.followMode = !m.followMode
			if m.followMode {
				m.viewport.GotoBottom()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 8
		m.updateViewportContent()
		return m, nil
	}

	return m, nil
}

// View renders the logs view
func (m LogsModel) View() string {
	var b strings.Builder

	// Header with process name and status
	header := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("⚡ prox - Logs"),
		lipgloss.NewStyle().
			Foreground(colorMuted).
			Render(fmt.Sprintf("Process: %s", m.processName)),
	)

	// Box the header
	headerBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(1, 2).
		Width(m.viewport.Width + 4).
		Render(header)

	b.WriteString(headerBox)
	b.WriteString("\n")

	// Content - box the viewport/message
	var content string
	if m.loading {
		content = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(2, 4).
			Render("Loading logs...")
	} else if m.err != nil {
		content = lipgloss.NewStyle().
			Foreground(colorDanger).
			Padding(2, 4).
			Render(fmt.Sprintf("Error: %v", m.err))
	} else if len(m.entries) == 0 {
		emptyMsg := fmt.Sprintf(`No logs available for process '%s'

Process may still be starting or has no output yet.
Try running it for a moment, then press 'r' to refresh.

Press 'esc' or 'q' to go back.`, m.processName)

		content = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Padding(2, 4).
			Render(emptyMsg)
	} else {
		content = m.viewport.View()
	}

	// Box the content
	contentBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Width(m.viewport.Width + 4).
		Height(m.viewport.Height + 2).
		Render(content)

	b.WriteString(contentBox)
	b.WriteString("\n")

	// Footer with controls
	followStatus := "OFF"
	if m.followMode {
		followStatus = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true).
			Render("ON")
	} else {
		followStatus = lipgloss.NewStyle().
			Foreground(colorMuted).
			Render("OFF")
	}

	footer := fmt.Sprintf("↑/k up  ↓/j down  g top  G bottom  f follow[%s]  r refresh  esc exit", followStatus)

	// Box the footer
	footerBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderTop(true).
		BorderForeground(colorBorder).
		Foreground(colorMuted).
		Padding(0, 2).
		Width(m.viewport.Width + 4).
		Render(footer)

	b.WriteString(footerBox)

	return b.String()
}

// updateViewportContent rebuilds viewport with colorized lines
func (m *LogsModel) updateViewportContent() {
	lines := make([]string, len(m.entries))

	for i, entry := range m.entries {
		lines[i] = formatLogEntry(entry)
	}

	content := strings.Join(lines, "\n")
	m.viewport.SetContent(content)

	// Auto-scroll if in follow mode
	if m.followMode {
		m.viewport.GotoBottom()
	}
}

// formatLogEntry colorizes a log entry based on source
func formatLogEntry(entry logs.LogEntry) string {
	var indicator string
	if entry.Source == logs.LogSourceStderr {
		indicator = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true).
			Render("[ERR]")
	} else {
		indicator = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Render("[OUT]")
	}

	// Format timestamp
	timeStr := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(entry.Timestamp.Format("15:04:05"))

	return fmt.Sprintf("%s %s %s", timeStr, indicator, entry.Content)
}

// Messages for log view

type logEntriesMsg []logs.LogEntry
type logErrorMsg error
type logTickMsg time.Time

// Commands for log operations

func loadLogsCmd(storage *storage.Storage, processName string, tailLines int) tea.Cmd {
	return func() tea.Msg {
		entries, err := logs.MergeLogs(storage, processName, tailLines)
		if err != nil {
			return logErrorMsg(err)
		}
		return logEntriesMsg(entries)
	}
}

func tailLogsCmd(storage *storage.Storage, processName string, currentLineCount int) tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		// Read all logs and compare count
		entries, err := logs.MergeLogs(storage, processName, 0) // Read all
		if err != nil {
			return logTickMsg(t)
		}

		// If we have new entries, return them
		if len(entries) > currentLineCount {
			newEntries := entries[currentLineCount:]
			return logEntriesMsg(newEntries)
		}

		return logTickMsg(t)
	})
}
