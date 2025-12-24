package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Color palette
	colorPrimary = lipgloss.Color("#00D9FF") // Cyan/electric blue
	colorSuccess = lipgloss.Color("#00FF87") // Green
	colorWarning = lipgloss.Color("#FFD700") // Yellow
	colorDanger  = lipgloss.Color("#FF5F87") // Red
	colorMuted   = lipgloss.Color("#6C7086") // Gray
	colorBorder  = lipgloss.Color("#45475A") // Dark gray
	colorBg      = lipgloss.Color("#1E1E2E") // Dark background
	colorText    = lipgloss.Color("#CDD6F4") // Light text

	// Title style
	titleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)

	// Header style
	headerStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBorder).
			Bold(true).
			Padding(0, 1)

	// Status styles
	statusOnlineStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true).
				Padding(0, 1)

	statusStoppedStyle = lipgloss.NewStyle().
				Foreground(colorWarning).
				Padding(0, 1)

	statusErroredStyle = lipgloss.NewStyle().
				Foreground(colorDanger).
				Bold(true).
				Padding(0, 1)

	statusRestartingStyle = lipgloss.NewStyle().
				Foreground(colorWarning).
				Bold(true).
				Padding(0, 1)

	// Table styles
	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				Align(lipgloss.Left).
				Padding(0, 1)

	tableCellStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	tableSelectedStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Padding(0, 1)

	// Help/footer style
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(1, 2)

	// Border style
	borderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	// Stats style
	statsStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 2)

	statsLabelStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	statsValueStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)
)

// GetStatusStyle returns the style for a given status
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "online":
		return statusOnlineStyle
	case "stopped":
		return statusStoppedStyle
	case "errored":
		return statusErroredStyle
	case "restarting", "stopping":
		return statusRestartingStyle
	default:
		return tableCellStyle
	}
}

// StatusIndicator returns a styled status indicator
func StatusIndicator(status string) string {
	var symbol string
	switch status {
	case "online":
		symbol = "●"
	case "stopped":
		symbol = "○"
	case "errored":
		symbol = "✗"
	case "restarting":
		symbol = "↻"
	case "stopping":
		symbol = "⏹"
	default:
		symbol = "?"
	}
	return GetStatusStyle(status).Render(symbol + " " + status)
}
