package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart <name|id>",
	Short: "Restart a process",
	Long:  `Restart a process by name or ID`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nameOrID := args[0]

		// Get shared manager
		mgr, _, err := getManager()
		if err != nil {
			PrintError("Failed to initialize manager: %v", err)
			os.Exit(1)
		}

		// Restart process
		if err := mgr.Restart(nameOrID); err != nil {
			PrintError("Failed to restart process: %v", err)
			os.Exit(1)
		}

		// Save state
		if err := saveState(); err != nil {
			PrintWarning("Failed to save state: %v", err)
		}

		PrintInfo("Restarted '%s'", nameOrID)
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
