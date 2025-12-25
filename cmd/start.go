package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/craigderington/prox/internal/process"
	"github.com/spf13/cobra"
)

var (
	processName string
	processArgs []string
	processCwd  string
	interpreter string
)

var startCmd = &cobra.Command{
	Use:   "start [script]",
	Short: "Start a process or all services from prox.yml",
	Long: `Start a new process or all services from prox.yml

Examples:
  prox start                           # Start all services from prox.yml
  prox start app.js                    # Start app.js with auto-detected name
  prox start app.js --name my-api      # Start with custom name "my-api"
  prox start server.py -n backend      # Start with short flag
  prox start app.js -i node            # Specify interpreter`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no args, try to start from prox.yml
		if len(args) == 0 {
			startAllCmd.Run(cmd, args)
			return
		}
		script := args[0]

		// Use script name as process name if not provided
		if processName == "" {
			processName = filepath.Base(script)
		}

		// Get remaining args
		if len(args) > 1 {
			processArgs = args[1:]
		}

		// Detect interpreter if not specified
		if interpreter == "" {
			interpreter = detectInterpreter(script)
		}

		// Get current directory if not specified
		if processCwd == "" {
			processCwd, _ = os.Getwd()
		}

		// Create config
		config := process.ProcessConfig{
			Name:        processName,
			Script:      script,
			Interpreter: interpreter,
			Args:        processArgs,
			Cwd:         processCwd,
		}

		// Get shared manager
		mgr, _, err := getManager()
		if err != nil {
			PrintError("Failed to initialize manager: %v", err)
			os.Exit(1)
		}

		// Start process
		proc, err := mgr.Start(config)
		if err != nil {
			PrintError("Failed to start process: %v", err)
			os.Exit(1)
		}

		// Save state
		if err := saveState(); err != nil {
			PrintWarning("Failed to save state: %v", err)
		}

		PrintSuccess("Started '%s' (PID %d)", proc.Name, proc.PID)
		PrintDetail("Script", proc.Script)
		if proc.Interpreter != "" {
			PrintDetail("Interpreter", proc.Interpreter)
		}
		PrintDetail("Status", string(proc.Status))

		fmt.Println()

		// Show all processes in a table
		processes := mgr.List()
		renderProcessTable(processes)

		fmt.Println()

		// Show helpful commands
		PrintInfo("Use 'prox' or 'prox monitor' to view in TUI")
		PrintMuted("     'prox list' to see all processes")
		PrintMuted("     'prox logs %s' to view logs", proc.Name)
		fmt.Println()
		PrintMuted("💡 Tip: Use --name to give your process a custom name")
		PrintMuted("     Example: prox start app.js --name my-api")
	},
}

func detectInterpreter(script string) string {
	ext := filepath.Ext(script)
	switch ext {
	case ".js":
		return "node"
	case ".py":
		return "python"
	case ".rb":
		return "ruby"
	case ".sh":
		return "bash"
	default:
		return "" // Direct execution
	}
}

// renderProcessTable renders a colored table of processes
func renderProcessTable(processes []*process.Process) {
	if len(processes) == 0 {
		PrintMuted("No processes running")
		return
	}

	// Build table rows
	rows := [][]string{}
	for i, proc := range processes {
		pid := "-"
		if proc.PID > 0 {
			pid = fmt.Sprintf("%d", proc.PID)
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
			pid,
			fmt.Sprintf("%d", proc.Restarts),
			proc.Script,
		})
	}

	// Create beautiful lipgloss table
	colorBorder := lipgloss.Color("#45475A")
	tableHeaderStyle := lipgloss.NewStyle().
		Foreground(colorInfo).
		Bold(true).
		Align(lipgloss.Left).
		Padding(0, 1)

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
				return lipgloss.NewStyle()
			}

			// ID column (col 0) - cyan
			if col == 0 {
				return lipgloss.NewStyle().Foreground(colorInfo).Padding(0, 1)
			}

			// Status column (col 2) - apply color coding
			if col == 2 {
				statusText := rows[dataRow][col]
				if strings.HasPrefix(statusText, "●") {
					return successStyle
				} else if strings.HasPrefix(statusText, "○") {
					return warningStyle // Orange for stopped
				} else if strings.HasPrefix(statusText, "✗") {
					return errorStyle
				} else if strings.HasPrefix(statusText, "↻") {
					return warningStyle
				}
				return lipgloss.NewStyle()
			}

			// Script column (last column) - muted
			if col == 5 {
				return mutedStyle
			}

			// Default style
			return lipgloss.NewStyle().Padding(0, 1)
		}).
		Headers("ID", "NAME", "STATUS", "PID", "RESTARTS", "SCRIPT").
		Rows(rows...)

	// Add some padding around the table
	tableOutput := t.Render()
	paddedOutput := lipgloss.NewStyle().
		Padding(0, 1).
		Render(tableOutput)

	fmt.Println(paddedOutput)
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVarP(&processName, "name", "n", "", "Process name")
	startCmd.Flags().StringVarP(&processCwd, "cwd", "c", "", "Working directory")
	startCmd.Flags().StringVarP(&interpreter, "interpreter", "i", "", "Interpreter (node, python, etc.)")
}
