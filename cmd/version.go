package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Give cli version",
	Long:  `All software has versions, this is spoof's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("0.0.5")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
