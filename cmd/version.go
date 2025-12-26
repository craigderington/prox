package cmd

import (
	"fmt"

	"github.com/craigderington/prox/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display detailed version information including build details",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("prox", version.GetVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
