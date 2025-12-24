package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/craigderington/prox/internal/config"
	"github.com/spf13/cobra"
)

var (
	initFile string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize prox configuration from existing project files",
	Long: `Auto-discover services from Procfile, package.json, or docker-compose.yml
and create a prox.yml configuration file.

Examples:
  prox init                    # Auto-discover from current directory
  prox init -f ../Procfile     # Use specific Procfile
  prox init -f config/prod.Procfile`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if prox.yml already exists
		if _, err := os.Stat("prox.yml"); err == nil {
			PrintError("prox.yml already exists. Delete it first or edit manually.")
			os.Exit(1)
		}

		// Auto-discover services
		PrintInfo("🔍 Discovering services...")

		var cfg *config.Config
		var err error
		var source string

		if initFile != "" {
			// Use specific file
			source = initFile
			PrintInfo("Using file: %s", initFile)
			cfg, err = config.AutoDiscoverFromFile(initFile)
		} else {
			// Auto-discover
			cfg, err = config.AutoDiscover()
			source = config.GetDiscoverySource()
		}

		if err != nil {
			PrintError("Failed to discover services: %v", err)
			if initFile == "" {
				PrintMuted("\nSupported files:")
				PrintMuted("  • Procfile")
				PrintMuted("  • package.json (npm scripts)")
				PrintMuted("  • docker-compose.yml")
				PrintMuted("\nOr use -f to specify a file")
			}
			os.Exit(1)
		}

		PrintSuccess("✓ Found %d service(s) in %s", len(cfg.Services), source)

		// Show discovered services
		fmt.Println()
		PrintInfo("Discovered services:")
		for name, svc := range cfg.Services {
			nameStyle := lipgloss.NewStyle().Foreground(colorSuccess).Render(name)
			cmdStyle := lipgloss.NewStyle().Foreground(colorMuted).Render(svc.Command)
			fmt.Printf("  • %s: %s\n", nameStyle, cmdStyle)
		}

		// Save to prox.yml
		fmt.Println()
		PrintInfo("📝 Writing prox.yml...")
		if err := config.SaveConfig("prox.yml", cfg); err != nil {
			PrintError("Failed to save config: %v", err)
			os.Exit(1)
		}

		PrintSuccess("✓ Created prox.yml")
		fmt.Println()
		PrintMuted("Next steps:")
		PrintMuted("  1. Review and edit prox.yml if needed")
		PrintMuted("  2. Run 'prox start' to start all services")
		PrintMuted("  3. Run 'prox' to open the TUI dashboard")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initFile, "file", "f", "", "Specific config file to use (Procfile, package.json, docker-compose.yml)")
}
