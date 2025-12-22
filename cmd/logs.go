package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/craigderington/prox/internal/logs"
	"github.com/spf13/cobra"
)

var (
	logsLines  int
	logsStream string
)

var logsCmd = &cobra.Command{
	Use:   "logs <name|id>",
	Short: "View process logs",
	Long:  `View stdout and stderr logs for a process`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nameOrID := args[0]

		// Get shared manager
		mgr, storage, err := getManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to initialize manager: %v\n", err)
			os.Exit(1)
		}

		// Find process
		proc := mgr.Get(nameOrID)
		if proc == nil {
			fmt.Fprintf(os.Stderr, "⚠️  Process not found: %s\n", nameOrID)
			os.Exit(1)
		}

		// Determine which log file(s) to read
		var logFiles []string
		var labels []string

		switch strings.ToLower(logsStream) {
		case "out", "stdout":
			logFiles = []string{storage.GetLogFile(proc.Name, "out")}
			labels = []string{"STDOUT"}
		case "err", "stderr":
			logFiles = []string{storage.GetLogFile(proc.Name, "err")}
			labels = []string{"STDERR"}
		default: // both
			logFiles = []string{
				storage.GetLogFile(proc.Name, "out"),
				storage.GetLogFile(proc.Name, "err"),
			}
			labels = []string{"STDOUT", "STDERR"}
		}

		// Read and display logs
		fmt.Printf("📋 Logs for '%s' (last %d lines)\n\n", proc.Name, logsLines)

		for i, logFile := range logFiles {
			lines, err := logs.ReadTail(logFile, logsLines)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Failed to read %s: %v\n", labels[i], err)
				continue
			}

			if len(lines) == 0 {
				fmt.Printf("  [%s] No logs\n", labels[i])
				continue
			}

			for _, line := range lines {
				fmt.Printf("  [%s] %s\n", labels[i], line)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "Number of lines to show")
	logsCmd.Flags().StringVarP(&logsStream, "stream", "t", "both", "Stream to show (stdout, stderr, both)")
}
