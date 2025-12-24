package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/craigderington/prox/internal/daemon"
	"github.com/craigderington/prox/internal/process"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the prox daemon",
	Long:  `Start, stop, or check status of the background prox daemon for auto-restart`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the prox daemon",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if already running
		client, err := daemon.NewClient()
		if err != nil {
			PrintError("Failed to create client: %v", err)
			os.Exit(1)
		}

		if client.IsRunning() {
			PrintWarning("Daemon already running")
			return
		}

		// Get executable path
		exePath, err := os.Executable()
		if err != nil {
			PrintError("Failed to get executable path: %v", err)
			os.Exit(1)
		}

		// Start daemon in background
		daemonCmd := exec.Command(exePath, "daemon", "run")
		daemonCmd.Stdout = nil
		daemonCmd.Stderr = nil
		daemonCmd.Stdin = nil

		if err := daemonCmd.Start(); err != nil {
			PrintError("Failed to start daemon: %v", err)
			os.Exit(1)
		}

		// Detach from parent
		daemonCmd.Process.Release()

		// Wait for daemon to be ready
		PrintInfo("Starting daemon...")
		for i := 0; i < 10; i++ {
			time.Sleep(200 * time.Millisecond)
			if client.IsRunning() {
				PrintSuccess("✓ Daemon started")
				PrintMuted("  • Auto-restart is now enabled")
				PrintMuted("  • Processes will restart on crash")
				PrintMuted("  • Use 'prox daemon stop' to stop the daemon")
				return
			}
		}

		PrintWarning("Daemon started but not responding")
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the prox daemon",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := daemon.NewClient()
		if err != nil {
			PrintError("Failed to create client: %v", err)
			os.Exit(1)
		}

		if !client.IsRunning() {
			PrintMuted("Daemon not running")
			return
		}

		PrintInfo("Stopping daemon...")
		if err := client.Shutdown(); err != nil {
			PrintError("Failed to stop daemon: %v", err)
			os.Exit(1)
		}

		PrintSuccess("✓ Daemon stopped")
		PrintMuted("  • Auto-restart is now disabled")
		PrintMuted("  • Running processes will continue")
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := daemon.NewClient()
		if err != nil {
			PrintError("Failed to create client: %v", err)
			os.Exit(1)
		}

		if client.IsRunning() {
			PrintSuccess("✓ Daemon is running")

			// Get process list
			processes, err := client.List()
			if err != nil {
				PrintWarning("  Failed to get process list: %v", err)
				return
			}

			online := 0
			for _, p := range processes {
				if p.Status == process.StatusOnline {
					online++
				}
			}

			PrintMuted("  • Managing %d process(es)", len(processes))
			PrintMuted("  • %d online", online)

			// Show socket path
			configDir, _ := process.ConfigDir()
			sockPath := filepath.Join(configDir, "daemon.sock")
			PrintMuted("  • Socket: %s", sockPath)
		} else {
			PrintMuted("✗ Daemon is not running")
			PrintMuted("  • Start with: prox daemon start")
			PrintMuted("  • Auto-restart is disabled")
		}
	},
}

var daemonRunCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the daemon (internal use)",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		server, err := daemon.NewServer()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
			os.Exit(1)
		}

		// Redirect output to log file
		configDir, _ := process.ConfigDir()
		logPath := filepath.Join(configDir, "daemon.log")
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			os.Stdout = logFile
			os.Stderr = logFile
			defer logFile.Close()
		}

		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Daemon error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonRunCmd)
}
