package cmd

import (
	"fmt"
	"os"

	"github.com/kream404/scratch/models"
	"github.com/kream404/scratch/services/json"
	"github.com/spf13/cobra"
)

var config_path string
var config models.FileConfig

var rootCmd = &cobra.Command{
	Use:   "scratch", // This is the name of your CLI tool
	Short: "A brief description of your CLI",
	Run: func(cmd *cobra.Command, args []string) {

		// multiple returns, _ indicates ignore the error. neato
		version, _ := cmd.Flags().GetBool("version")
		verbose, _ := cmd.Flags().GetBool("verbose")

		if version {
			versionCmd.Run(cmd, args)
			return
		}

		config, _ := json.LoadConfig(config_path)

		if verbose && config_path != "" {
			fmt.Println("=================================")
			fmt.Println("config path: ", config_path)
			fmt.Println("=================================")
			print(json.ToJSONString(config))
		}

	},
}

func init() {
	// Register versionCmd (or any other commands) with rootCmd
	rootCmd.PersistentFlags().Bool("version", false, "show cli version")
	rootCmd.PersistentFlags().Bool("verbose", false, "show additional logs")
	rootCmd.PersistentFlags().StringVar(&config_path, "config", "", "path to config file")
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
