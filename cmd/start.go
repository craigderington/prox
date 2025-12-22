package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
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
	Use:   "start <script>",
	Short: "Start a process",
	Long: `Start a new process and add it to the process manager

Examples:
  prox start app.js                    # Start app.js with auto-detected name
  prox start app.js --name my-api      # Start with custom name "my-api"
  prox start server.py -n backend      # Start with short flag
  prox start app.js -i node            # Specify interpreter`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	headerStyle := lipgloss.NewStyle().Foreground(colorInfo).Bold(true)

	fmt.Fprintln(w, headerStyle.Render("ID")+"\t"+
		headerStyle.Render("NAME")+"\t"+
		headerStyle.Render("STATUS")+"\t"+
		headerStyle.Render("PID")+"\t"+
		headerStyle.Render("RESTARTS")+"\t"+
		headerStyle.Render("SCRIPT"))

	for i, proc := range processes {
		pid := "-"
		if proc.PID > 0 {
			pid = fmt.Sprintf("%d", proc.PID)
		}

		// Color-code status
		statusStr := string(proc.Status)
		var statusStyled string
		switch proc.Status {
		case process.StatusOnline:
			statusStyled = successStyle.Render("● " + statusStr)
		case process.StatusStopped:
			statusStyled = mutedStyle.Render("○ " + statusStr)
		case process.StatusErrored:
			statusStyled = errorStyle.Render("✗ " + statusStr)
		case process.StatusRestarting:
			statusStyled = warningStyle.Render("↻ " + statusStr)
		default:
			statusStyled = statusStr
		}

		// ID in cyan
		idStyled := lipgloss.NewStyle().Foreground(colorInfo).Render(fmt.Sprintf("%d", i))

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			idStyled,
			proc.Name,
			statusStyled,
			pid,
			proc.Restarts,
			mutedStyle.Render(proc.Script),
		)
	}

	w.Flush()
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVarP(&processName, "name", "n", "", "Process name")
	startCmd.Flags().StringVarP(&processCwd, "cwd", "c", "", "Working directory")
	startCmd.Flags().StringVarP(&interpreter, "interpreter", "i", "", "Interpreter (node, python, etc.)")
}
