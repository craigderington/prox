package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/craigderington/prox/internal/process"
	"github.com/spf13/cobra"
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

		if len(processes) == 0 {
			PrintMuted("No processes running. Use 'prox start <script>' to add processes.")
			return
		}

		// Print header
		headerStyle := lipgloss.NewStyle().Foreground(colorInfo).Bold(true)
		fmt.Println(headerStyle.Render("⚡ prox - Process List"))
		fmt.Println()

		// Create table writer
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, headerStyle.Render("NAME")+"\t"+
			headerStyle.Render("STATUS")+"\t"+
			headerStyle.Render("PID")+"\t"+
			headerStyle.Render("RESTARTS")+"\t"+
			headerStyle.Render("UPTIME")+"\t"+
			headerStyle.Render("SCRIPT"))

		for _, proc := range processes {
			pid := "-"
			if proc.PID > 0 {
				pid = fmt.Sprintf("%d", proc.PID)
			}

			uptime := "-"
			if proc.Status == process.StatusOnline {
				uptime = formatDuration(proc.Uptime())
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

			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
				proc.Name,
				statusStyled,
				pid,
				proc.Restarts,
				uptime,
				mutedStyle.Render(proc.Script),
			)
		}

		w.Flush()
	},
}

func formatDuration(d interface{}) string {
	// This will be implemented with proper duration formatting
	return fmt.Sprintf("%v", d)
}

func init() {
	rootCmd.AddCommand(listCmd)
}
