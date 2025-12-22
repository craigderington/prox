package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/craigderington/prox/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "prox",
	Short: "⚡ Process manager TUI",
	Long: `prox ⚡ - A modern process manager with a beautiful TUI

Universal process management for applications in any language.
Inspired by pm2, built with Go and Bubbletea.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand, launch TUI
		if err := launchTUI(); err != nil {
			fmt.Fprintf(os.Stderr, "Error launching TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can go here
}

func launchTUI() error {
	// Get shared manager and storage
	mgr, store, err := getManager()
	if err != nil {
		return fmt.Errorf("failed to initialize manager: %w", err)
	}

	// Create TUI model
	model := tui.NewModel(mgr, store)

	// Create Bubbletea program
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Run the program
	if _, err := p.Run(); err != nil {
		return err
	}

	// Save state on exit
	if err := saveState(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
	}

	return nil
}
