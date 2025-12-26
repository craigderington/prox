package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	manager       *process.Manager
	storage       *storage.Storage
	processName   string
	viewport      viewport.Model
	entries       []logs.LogEntry
	totalLogsSeen int // Track total logs seen for incremental updates
	followMode    bool
	loading       bool
	err           error
	writingToFile bool     // Toggle state for continuous writing
	logFile       *os.File // Handle to open log file
	logFilePath   string   // Path to the log file being written
}

// NewLogsModel creates a new logs view model
func NewLogsModel(manager *process.Manager, storage *storage.Storage, processName string, width, height int) LogsModel {
	vp := viewport.New(width-4, height-10) // Leave more room for header
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	return LogsModel{
		manager:       manager,
		storage:       storage,
		processName:   processName,
		viewport:      vp,
		entries:       []logs.LogEntry{},
		totalLogsSeen: 0,
		followMode:    true,
		loading:       true,
		err:           nil,
		writingToFile: false,
		logFile:       nil,
		logFilePath:   "",
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

		case "r":
			// Refresh logs
			m.entries = []logs.LogEntry{}
			m.totalLogsSeen = 0
			m.loading = true
			return m, loadLogsCmd(m.storage, m.processName, 100)

		case "w":
			// Toggle continuous writing mode
			if m.writingToFile {
				// Turn OFF writing
				if m.logFile != nil {
					m.logFile.Close()
					m.logFile = nil
				}
				m.writingToFile = false
			} else {
				// Turn ON writing - create new file
				timestamp := time.Now().Format("2006-01-02_15-04-05")
				filename := fmt.Sprintf("%s_logs_%s.txt", m.processName, timestamp)
				filepath := filepath.Join(".", filename)

				file, err := os.Create(filepath)
				if err != nil {
					m.err = fmt.Errorf("failed to create log file: %w", err)
					return m, nil
				}

				// Write header
				file.WriteString("# Logs for process: " + m.processName + "\n")
				file.WriteString("# Started: " + time.Now().Format(time.RFC3339) + "\n")
				file.WriteString("# Continuous write mode - logs will be appended in real-time\n\n")

				// Write existing entries
				for _, entry := range m.entries {
					writeLogEntry(file, entry)
				}

				m.logFile = file
				m.logFilePath = filepath
				m.writingToFile = true
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10
		m.updateViewportContent()
		return m, nil

	case logEntriesMsg:
		if len(msg) > 0 {
			if len(m.entries) == 0 {
				// Initial load
				m.entries = msg
				m.totalLogsSeen = len(msg)
				m.loading = false
				m.updateViewportContent()
				if m.followMode {
					m.viewport.GotoBottom()
				}
				// Start continuous tailing after initial load
				return m, tailLogsCmd(m.storage, m.processName, m.totalLogsSeen)
			} else {
				// Incremental update
				m.entries = append(m.entries, msg...)
				m.totalLogsSeen += len(msg)

				// If writing to file, append new entries
				if m.writingToFile && m.logFile != nil {
					for _, entry := range msg {
						writeLogEntry(m.logFile, entry)
					}
				}

				// Cap at max lines
				if len(m.entries) > maxLogLines {
					m.entries = m.entries[len(m.entries)-maxLogLines:]
					// Don't change totalLogsSeen when capping
				}

				m.updateViewportContent()

				// Ensure we scroll to bottom when following
				if m.followMode {
					m.viewport.GotoBottom()
				}
			}
		}
		// Continue tailing
		return m, tailLogsCmd(m.storage, m.processName, m.totalLogsSeen)

	case logTickMsg:
		// Continue tailing on tick
		return m, tailLogsCmd(m.storage, m.processName, m.totalLogsSeen)

	case logErrorMsg:
		m.err = error(msg)
		m.loading = false
		return m, nil
	}

	return m, nil
}

// renderLogsHeader renders the header with title and log stats
func renderLogsHeader(m LogsModel) string {
	title := titleStyle.Render("⚡ prox - Logs")

	// Log stats - more compact
	totalLogs := m.totalLogsSeen
	displayedLogs := len(m.entries)
	followStatus := "OFF"
	if m.followMode {
		followStatus = "ON"
	}

	// Single line stats - truncate process name if too long
	processName := m.processName
	if len(processName) > 15 {
		processName = processName[:12] + "..."
	}
	statsText := fmt.Sprintf("Process: %s • Total: %d • Displayed: %d • Follow: %s",
		processName, totalLogs, displayedLogs, followStatus)

	stats := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(statsText)

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		stats,
	)

	return header
}

// View renders the logs view
func (m LogsModel) View() string {
	var b strings.Builder

	// Header with title and stats (similar to dashboard)
	header := renderLogsHeader(m)

	// Box the header
	headerBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 0). // Reduce padding to save space
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

	// Write status indicator
	writeIndicator := "w write"
	if m.writingToFile {
		writeIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")). // Gold color
			Bold(true).
			Render("w WRITING")
	}

	footer := fmt.Sprintf("↑/k up  ↓/j down  g top  G bottom  f follow[%s]  r refresh  %s  esc exit", followStatus, writeIndicator)

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

// writeLogEntry writes a single log entry to the file
func writeLogEntry(file *os.File, entry logs.LogEntry) {
	if file == nil {
		return
	}

	source := "OUT"
	if entry.Source == logs.LogSourceStderr {
		source = "ERR"
	}
	line := fmt.Sprintf("[%s] [%s] %s\n",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		source,
		entry.Content)
	file.WriteString(line)
}

// writeLogsToFile writes the current log entries to a file (one-time snapshot)
// Used by monitor view for quick export
func writeLogsToFile(processName string, entries []logs.LogEntry) tea.Cmd {
	return func() tea.Msg {
		// Create filename with timestamp
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("%s_logs_%s.txt", processName, timestamp)

		// Write to current directory
		filepath := filepath.Join(".", filename)

		file, err := os.Create(filepath)
		if err != nil {
			return logErrorMsg(fmt.Errorf("failed to create log file: %w", err))
		}
		defer file.Close()

		// Write header
		file.WriteString(fmt.Sprintf("# Logs for process: %s\n", processName))
		file.WriteString(fmt.Sprintf("# Exported: %s\n", time.Now().Format(time.RFC3339)))
		file.WriteString(fmt.Sprintf("# Total entries: %d\n\n", len(entries)))

		// Write each log entry
		for _, entry := range entries {
			writeLogEntry(file, entry)
		}

		// Return a success message (we'll handle this as a no-op for now)
		return nil
	}
}
