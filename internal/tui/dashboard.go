package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/craigderington/prox/internal/process"
)

// renderDashboard renders the main dashboard view
func renderDashboard(m Model) string {
	var b strings.Builder

	// Header (boxed)
	b.WriteString(renderHeader(m))
	b.WriteString("\n")

	// Start process input area (between header and process list)
	b.WriteString(renderStartInput(m))
	b.WriteString("\n")

	// Process table (bordered)
	b.WriteString(renderProcessTable(m))
	b.WriteString("\n")

	// Footer/help (boxed)
	b.WriteString(renderFooter(m))

	return b.String()
}

// renderHeader renders the header with title and stats
func renderHeader(m Model) string {
	title := titleStyle.Render("⚡ prox")

	// Count processes by status
	online := 0
	stopped := 0
	errored := 0

	for _, proc := range m.processes {
		switch proc.Status {
		case process.StatusOnline:
			online++
		case process.StatusStopped:
			stopped++
		case process.StatusErrored:
			errored++
		}
	}

	// Individual stat boxes with color backgrounds
	totalBox := lipgloss.NewStyle().
		Foreground(colorText).
		Background(lipgloss.Color("#313244")).
		Padding(0, 2).
		Render(fmt.Sprintf("Total: %d", len(m.processes)))

	onlineBox := lipgloss.NewStyle().
		Foreground(colorSuccess).
		Background(lipgloss.Color("#0a3a0a")).
		Bold(true).
		Padding(0, 2).
		Render(fmt.Sprintf("● %d Online", online))

	stoppedBox := lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(lipgloss.Color("#2a2a2a")).
		Padding(0, 2).
		Render(fmt.Sprintf("○ %d Stopped", stopped))

	erroredBox := lipgloss.NewStyle().
		Foreground(colorDanger).
		Background(lipgloss.Color("#3a0a0a")).
		Bold(true).
		Padding(0, 2).
		Render(fmt.Sprintf("✗ %d Errored", errored))

	stats := lipgloss.JoinHorizontal(
		lipgloss.Left,
		totalBox,
		"  ",
		onlineBox,
		"  ",
		stoppedBox,
		"  ",
		erroredBox,
	)

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		stats,
	)

	// Box the header with minimal margins
	headerBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Width(m.width - 2).
		Render(header)

	return headerBox
}

// renderStartInput renders the start process input area
func renderStartInput(m Model) string {
	label := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("Start Process:")

	inputBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(func() lipgloss.Color {
			if m.inputMode {
				return colorPrimary
			}
			return colorBorder
		}()).
		Padding(0, 1).
		Width(m.width - 2).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			label,
			m.startInput.View(),
		))

	hint := lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true).
		Render("Press 'n' to start a new process • ESC to cancel • ENTER to submit")

	if m.inputMode {
		hint = lipgloss.NewStyle().
			Foreground(colorWarning).
			Italic(true).
			Render("Type your command • ESC to cancel • ENTER to submit")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		inputBox,
		hint,
	)
}

// renderProcessTable renders the process table
func renderProcessTable(m Model) string {
	if len(m.processes) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Padding(1, 2).
			Render("No processes running. Press 'n' to start a new process.")

		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			Width(m.width - 2).
			Render(emptyMsg)
	}

	// Build table rows
	rows := [][]string{}
	for i, proc := range m.processes {
		metrics := m.metrics[proc.ID]
		rows = append(rows, buildProcessRow(proc, metrics, i == m.selected))
	}

	// Create table with lipgloss table package with rounded borders
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorBorder)).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Header row
			if row == 0 {
				return tableHeaderStyle
			}

			// Data rows
			dataRow := row - 1
			if dataRow < 0 || dataRow >= len(rows) {
				return tableCellStyle
			}

			// Status column - always use status-specific colors
			if col == 1 {
				status := rows[dataRow][col]
				return GetStatusStyle(status)
			}

			// All other columns - use normal text color
			return tableCellStyle
		}).
		Headers("NAME", "STATUS", "CPU", "MEMORY", "UPTIME", "RESTARTS", "PID").
		Rows(rows...)

	return t.Render()
}

// buildProcessRow creates a row of data for the table
func buildProcessRow(proc *process.Process, metrics *process.ProcessMetrics, selected bool) []string {
	// Name - add selection indicator
	name := truncate(proc.Name, 18)
	if selected {
		name = "▶ " + name
	}

	// Status
	status := string(proc.Status)

	// CPU
	cpu := "-"
	if metrics != nil && proc.Status == process.StatusOnline {
		cpu = fmt.Sprintf("%.1f%%", metrics.CPU)
	}

	// Memory
	mem := "-"
	if metrics != nil && proc.Status == process.StatusOnline {
		mem = process.FormatBytes(metrics.Memory)
	}

	// Uptime
	uptime := "-"
	if metrics != nil && proc.Status == process.StatusOnline {
		uptime = process.FormatDuration(metrics.Uptime)
	}

	// Restarts
	restarts := fmt.Sprintf("%d", proc.Restarts)

	// PID
	pid := "-"
	if proc.PID > 0 {
		pid = fmt.Sprintf("%d", proc.PID)
	}

	return []string{name, status, cpu, mem, uptime, restarts, pid}
}

// renderFooter renders the footer with keyboard shortcuts
func renderFooter(m Model) string {
	help := []string{
		"n new",
		"↑/k up",
		"↓/j down",
		"enter monitor",
		"l logs",
		"r restart",
		"s stop",
		"d delete",
		"R refresh",
		"q quit",
	}

	helpText := strings.Join(help, " • ")

	// Box the footer with minimal margins
	footerBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderTop(true).
		BorderForeground(colorBorder).
		Foreground(colorMuted).
		Padding(0, 1).
		Width(m.width - 2).
		Render(helpText)

	return footerBox
}

// truncate truncates a string to the given length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
