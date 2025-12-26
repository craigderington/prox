package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/craigderington/prox/internal/config"
	"github.com/craigderington/prox/internal/process"
	"github.com/spf13/cobra"
)

var startAllCmd = &cobra.Command{
	Use:     "start-all",
	Aliases: []string{"up"},
	Short:   "Start all services from prox.yml",
	Long:    `Start all services defined in prox.yml configuration file`,
	Run: func(cmd *cobra.Command, args []string) {
		// Find config file
		configPath, err := config.FindConfigFile()
		if err != nil {
			PrintError("No prox.yml found. Run 'prox init' first.")
			os.Exit(1)
		}

		// Load config
		PrintInfo("📖 Loading config from %s", configPath)
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			PrintError("Failed to load config: %v", err)
			os.Exit(1)
		}

		if len(cfg.Services) == 0 {
			PrintWarning("No services defined in config")
			os.Exit(0)
		}

		// Get manager
		mgr, _, err := getManager()
		if err != nil {
			PrintError("Failed to initialize manager: %v", err)
			os.Exit(1)
		}

		// Start services in dependency order
		PrintInfo("🚀 Starting %d service(s)...\n", len(cfg.Services))

		started := make(map[string]bool)
		errors := []string{}

		// Helper to start a service and its dependencies
		var startService func(name string, svc config.ServiceConfig) error
		startService = func(name string, svc config.ServiceConfig) error {
			if started[name] {
				return nil
			}

			// Start dependencies first
			for _, dep := range svc.DependsOn {
				if depSvc, ok := cfg.Services[dep]; ok {
					if err := startService(dep, depSvc); err != nil {
						return err
					}
					// Wait a bit for dependency to start
					time.Sleep(500 * time.Millisecond)
				}
			}

			// Parse command
			parts := strings.Fields(svc.Command)
			if len(parts) == 0 {
				return fmt.Errorf("empty command")
			}

			// Check if first part is a known interpreter
			var script string
			var args []string
			var interp string

			if svc.Interpreter != "" {
				interp = svc.Interpreter
			}

			// Check if command starts with an interpreter
			knownInterpreters := map[string]bool{
				"node": true, "python": true, "python3": true,
				"ruby": true, "bash": true, "sh": true,
				"go": true, "npm": true, "npx": true,
			}

			if len(parts) > 1 && knownInterpreters[parts[0]] {
				// First part is interpreter, second is script
				if interp == "" {
					interp = parts[0]
				}
				script = parts[1]
				if len(parts) > 2 {
					args = parts[2:]
				}
			} else {
				// Direct execution
				script = parts[0]
				if len(parts) > 1 {
					args = parts[1:]
				}
				if interp == "" {
					interp = detectInterpreter(script)
				}
			}

			// Set working directory
			cwd := svc.Cwd
			if cwd == "" {
				cwd, _ = os.Getwd()
			}

			// Convert restart policy
			restartPolicy := process.RestartOnFailure
			switch svc.Restart {
			case "always":
				restartPolicy = process.RestartAlways
			case "never":
				restartPolicy = process.RestartNever
			}

			// Create process config
			procConfig := process.ProcessConfig{
				Name:        name,
				Script:      script,
				Interpreter: interp,
				Args:        args,
				Cwd:         cwd,
				Env:         svc.Env,
				Restart:     restartPolicy,
				DependsOn:   svc.DependsOn,
			}

			// Start process
			proc, err := mgr.Start(procConfig)
			if err != nil {
				return err
			}

			started[name] = true
			PrintSuccess("  ✓ %s (PID %d)", name, proc.PID)
			return nil
		}

		// Start all services
		for name, svc := range cfg.Services {
			if err := startService(name, svc); err != nil {
				errors = append(errors, fmt.Sprintf("  ✗ %s: %v", name, err))
			}
		}

		// Save state
		if err := saveState(); err != nil {
			PrintWarning("Failed to save state: %v", err)
		}

		// Print summary
		fmt.Println()
		if len(errors) > 0 {
			PrintError("Failed to start some services:")
			for _, e := range errors {
				fmt.Println(e)
			}
			fmt.Println()
		}

		PrintSuccess("✓ Started %d/%d services", len(started), len(cfg.Services))
		fmt.Println()
		PrintMuted("  • Run 'prox' to open TUI dashboard")
		PrintMuted("  • Run 'prox list' to see all processes")
		PrintMuted("  • Run 'prox logs <name>' to view logs")
	},
}

func init() {
	rootCmd.AddCommand(startAllCmd)
}
