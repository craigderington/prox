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

	// Box the header
	headerBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(1, 2).
		Width(m.width - 4).
		Render(header)

	return headerBox
}

// renderProcessTable renders the process table
func renderProcessTable(m Model) string {
	if len(m.processes) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Padding(2, 4).
			Render("No processes running. Use 'prox start <script>' to add processes.")

		return lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2).
			Width(m.width - 4).
			Render(emptyMsg)
	}

	// Build table rows
	rows := [][]string{}
	for i, proc := range m.processes {
		metrics := m.metrics[proc.ID]
		rows = append(rows, buildProcessRow(proc, metrics, i == m.selected))
	}

	// Create table with lipgloss table package
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorBorder)).
		StyleFunc(func(row, col int) lipgloss.Style {
			// Header row (row 0) - explicitly return header style ONLY for row 0
			if row == 0 {
				return tableHeaderStyle
			}

			// All other rows are data rows
			// Calculate the actual process index (row 1 = process 0, row 2 = process 1, etc)
			dataRow := row - 1

			// Bounds check to prevent index out of range
			if dataRow < 0 || dataRow >= len(rows) {
				// Return normal cell style as fallback
				return lipgloss.NewStyle().Foreground(colorText).Padding(0, 1)
			}

			// Status column (col 1) - ALWAYS preserve status colors
			// This must come before any other styling to ensure status colors are preserved
			if col == 1 {
				status := rows[dataRow][col]
				return GetStatusStyle(status)
			}

			// For all other columns (not status), check if row is selected
			if dataRow == m.selected {
				// Selected row - normal text color with padding
				return lipgloss.NewStyle().Foreground(colorText).Padding(0, 1)
			}

			// Default style for non-selected, non-status cells
			// Explicitly set foreground to prevent any default coloring
			return lipgloss.NewStyle().Foreground(colorText).Padding(0, 1)
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
		"↑/k up",
		"↓/j down",
		"enter monitor",
		"r restart",
		"s stop",
		"d delete",
		"R refresh",
		"q quit",
	}

	helpText := strings.Join(help, " • ")

	// Box the footer
	footerBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderTop(true).
		BorderForeground(colorBorder).
		Foreground(colorMuted).
		Padding(0, 2).
		Width(m.width - 4).
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
