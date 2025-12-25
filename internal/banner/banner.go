package banner

import (
	"github.com/charmbracelet/lipgloss"
)

const asciiArt = `
██████╗ ██████╗  ██████╗ ██╗  ██╗
██╔══██╗██╔══██╗██╔═══██╗╚██╗██╔╝
██████╔╝██████╔╝██║   ██║ ╚███╔╝
██╔═══╝ ██╔══██╗██║   ██║ ██╔██╗
██║     ██║  ██║╚██████╔╝██╔╝ ██╗
╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝
`

const tagline = "Process Manager for Modern Development"

const description = `
  prox is a lightweight TUI process manager with real-time metrics,
  auto-restart policies, and zero-config setup for any language.

  Quick Start:
  $ prox init              # Auto-discover from Procfile/package.json
  $ prox start             # Start all services
  $ prox                   # Open TUI dashboard with live metrics

  Features:
  • Beautiful TUI with CPU/Memory graphs
  • Auto-restart on crash (configurable policies)
  • Multi-language support (Node, Python, Go, Ruby, etc.)
  • Process dependencies and health checks
  • Zero-config with Procfile/package.json

  Learn more: https://github.com/yourusername/prox
`

// Render returns the formatted banner
func Render() string {
	colorPrimary := lipgloss.Color("#89b4fa")
	colorMuted := lipgloss.Color("#6c7086")
	colorAccent := lipgloss.Color("#f38ba8")

	logo := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(asciiArt)

	tag := lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		Render(tagline)

	desc := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(description)

	divider := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render("─────────────────────────────────────────────────────────────────")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		divider,
		"                   "+logo, // Center the logo
		"",
		"                    "+tag,
		desc,
		divider,
		"",
	)
}

// RenderCompact returns just the logo without description
func RenderCompact() string {
	colorPrimary := lipgloss.Color("#89b4fa")
	colorAccent := lipgloss.Color("#f38ba8")

	logo := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(asciiArt)

	tag := lipgloss.NewStyle().
		Foreground(colorAccent).
		Render("⚡ " + tagline)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		"                   "+logo, // Center the logo
		"",
		"                    "+tag,
		"",
	)
}
