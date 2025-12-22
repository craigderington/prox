package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <name|id>",
	Short: "Stop a process",
	Long:  `Stop a running process by name or ID`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nameOrID := args[0]

		// Get shared manager
		mgr, _, err := getManager()
		if err != nil {
			PrintError("Failed to initialize manager: %v", err)
			os.Exit(1)
		}

		// Stop process
		if err := mgr.Stop(nameOrID); err != nil {
			PrintError("Failed to stop process: %v", err)
			os.Exit(1)
		}

		// Save state
		if err := saveState(); err != nil {
			PrintWarning("Failed to save state: %v", err)
		}

		PrintSuccess("Stopped '%s'", nameOrID)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
