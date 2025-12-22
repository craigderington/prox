package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// CLI color palette
	colorSuccess = lipgloss.Color("#00FF87") // Green
	colorError   = lipgloss.Color("#FF5F87") // Red
	colorWarning = lipgloss.Color("#FFD700") // Yellow
	colorInfo    = lipgloss.Color("#00D9FF") // Cyan
	colorMuted   = lipgloss.Color("#6C7086") // Gray

	// CLI styles
	successStyle = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	warningStyle = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	infoStyle    = lipgloss.NewStyle().Foreground(colorInfo).Bold(true)
	mutedStyle   = lipgloss.NewStyle().Foreground(colorMuted)
	labelStyle   = lipgloss.NewStyle().Foreground(colorMuted)
	valueStyle   = lipgloss.NewStyle().Foreground(colorInfo)
)

// PrintSuccess prints a success message with green checkmark
func PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(successStyle.Render("✓") + " " + successStyle.Render(msg))
}

// PrintError prints an error message with red X
func PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(errorStyle.Render("✗") + " " + errorStyle.Render(msg))
}

// PrintWarning prints a warning message with yellow triangle
func PrintWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(warningStyle.Render("⚠") + " " + warningStyle.Render(msg))
}

// PrintInfo prints an info message with cyan bullet
func PrintInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(infoStyle.Render("●") + " " + msg)
}

// PrintDetail prints a detail line with label and value
func PrintDetail(label, value string) {
	fmt.Printf("  %s %s\n", labelStyle.Render(label+":"), valueStyle.Render(value))
}

// PrintMuted prints muted text
func PrintMuted(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(mutedStyle.Render(msg))
}
