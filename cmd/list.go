package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/craigderington/prox/internal/banner"
	"github.com/craigderington/prox/internal/process"
	"github.com/spf13/cobra"
)

var (
	showBanner bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "status"},
	Short:   "List all processes",
	Long:    `Display a list of all managed processes with their status`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get shared manager
		mgr, _, err := getManager()
		if err != nil {
			PrintError("Failed to initialize manager: %v", err)
			os.Exit(1)
		}

		processes := mgr.List()

		// Show banner if requested or if no processes
		if showBanner || len(processes) == 0 {
			fmt.Println(banner.Render())
		}

		if len(processes) == 0 {
			PrintMuted("No processes running. Get started:")
			PrintMuted("  $ prox init              # Auto-discover services")
			PrintMuted("  $ prox start app.js      # Start a single process")
			return
		}

		// Print compact header if banner not shown
		if !showBanner {
			headerStyle := lipgloss.NewStyle().Foreground(colorInfo).Bold(true)
			fmt.Println(headerStyle.Render("⚡ prox - Process List"))
			fmt.Println()
		}

		// Collect metrics for all processes
		collector := process.NewMetricsCollector(mgr)
		metricsMap := collector.CollectAllMetrics()

		// Build table rows
		rows := [][]string{}
		for i, proc := range processes {
			metrics := metricsMap[proc.ID]

			uptime := "-"
			if proc.Status == process.StatusOnline {
				uptime = process.FormatDuration(proc.Uptime())
			}

			// CPU and Memory
			cpu := "-"
			mem := "-"
			if metrics != nil && proc.Status == process.StatusOnline {
				cpu = fmt.Sprintf("%.1f%%", metrics.CPU)
				mem = process.FormatBytes(metrics.Memory)
			}

			// Status with symbol
			statusStr := string(proc.Status)
			var statusWithSymbol string
			switch proc.Status {
			case process.StatusOnline:
				statusWithSymbol = "● " + statusStr
			case process.StatusStopped:
				statusWithSymbol = "○ " + statusStr
			case process.StatusErrored:
				statusWithSymbol = "✗ " + statusStr
			case process.StatusRestarting:
				statusWithSymbol = "↻ " + statusStr
			default:
				statusWithSymbol = statusStr
			}

			rows = append(rows, []string{
				fmt.Sprintf("%d", i),
				proc.Name,
				statusWithSymbol,
				fmt.Sprintf("%d", proc.Restarts),
				cpu,
				mem,
				uptime,
				proc.Script,
			})
		}

		// Create beautiful lipgloss table
		colorBorder := lipgloss.Color("#45475A")
		tableHeaderStyle := lipgloss.NewStyle().
			Foreground(colorInfo).
			Bold(true).
			Align(lipgloss.Left)

		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(colorBorder)).
			StyleFunc(func(row, col int) lipgloss.Style {
				// Header row
				if row == 0 {
					return tableHeaderStyle
				}

				// Bounds check for data rows
				dataRow := row - 1
				if dataRow < 0 || dataRow >= len(rows) {
					return mutedStyle
				}

				// Status column (col 2) - apply color coding
				if col == 2 {
					// Get the status text from the row
					statusText := rows[dataRow][col]
					// Check what status it contains
					if strings.HasPrefix(statusText, "●") { // online
						return successStyle
					} else if strings.HasPrefix(statusText, "○") { // stopped
						return warningStyle // Orange for stopped
					} else if strings.HasPrefix(statusText, "✗") { // errored
						return errorStyle
					} else if strings.HasPrefix(statusText, "↻") { // restarting
						return warningStyle
					}
					return mutedStyle
				}

				// CPU column - color by usage
				if col == 4 {
					return lipgloss.NewStyle().Foreground(colorInfo)
				}

				// Memory column - color by usage
				if col == 5 {
					return lipgloss.NewStyle().Foreground(colorWarning)
				}

				// Uptime column - muted
				if col == 6 {
					return mutedStyle
				}

				// Script column - muted
				if col == 7 {
					return mutedStyle
				}

				// Default
				return mutedStyle
			}).
			Headers("ID", "NAME", "STATUS", "↺", "CPU", "MEMORY", "UPTIME", "SCRIPT").
			Rows(rows...)

		// Add some padding around the table
		tableOutput := t.Render()
		paddedOutput := lipgloss.NewStyle().
			Padding(0, 1).
			Render(tableOutput)

		fmt.Println(paddedOutput)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&showBanner, "banner", "b", false, "Show prox banner and info")
}
