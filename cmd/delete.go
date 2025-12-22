package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <name|id>",
	Aliases: []string{"del", "rm"},
	Short:   "Delete a process",
	Long:    `Delete a process (stops it first if running)`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nameOrID := args[0]

		// Get shared manager
		mgr, _, err := getManager()
		if err != nil {
			PrintError("Failed to initialize manager: %v", err)
			os.Exit(1)
		}

		// Delete process
		if err := mgr.Delete(nameOrID); err != nil {
			PrintError("Failed to delete process: %v", err)
			os.Exit(1)
		}

		// Save state
		if err := saveState(); err != nil {
			PrintWarning("Failed to save state: %v", err)
		}

		PrintSuccess("Deleted '%s'", nameOrID)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
