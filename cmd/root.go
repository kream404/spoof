package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/csv"
	"github.com/kream404/spoof/services/json"

	"github.com/spf13/cobra"
)

var config_path string
var config models.FileConfig
var scaffold string
var scaffold_name string

var rootCmd = &cobra.Command{
	Use:   "spoof",
	Short: "A brief description of your CLI",
	Run: func(cmd *cobra.Command, args []string) {

		// multiple returns, _ indicates ignore the error. neato
		version, _ := cmd.Flags().GetBool("version")
		verbose, _ := cmd.Flags().GetBool("verbose")
		scaffold, _ := cmd.Flags().GetBool("scaffold")


		if version {
			versionCmd.Run(cmd, args)
			return
		}
		config, _ := json.LoadConfig(config_path)

		if(verbose && config != nil) {
			start := time.Now() // Start timer
			fmt.Println("config path: ", config_path)
			fmt.Println("=================================")
			csv.GenerateCSV(*config, "./output/output.csv")
			elapsed := time.Since(start)
			fmt.Printf("\n⏱️  Done in %s\n", elapsed)
		}

		if(scaffold && scaffold_name != ""){
			fmt.Println(scaffold)
			fmt.Println("generating faker..")
			fmt.Println("scaffold_name: ", scaffold_name)

			faker_config := FakerConfig{
				Name: scaffold_name,
				DataType: scaffold_name,
				Format: "",
			}
			GenerateFaker(faker_config)
		}

	},
}

func init() {
	//main flags
	rootCmd.PersistentFlags().Bool("version", false, "show cli version")
	rootCmd.PersistentFlags().Bool("verbose", false, "show additional logs")
	rootCmd.PersistentFlags().StringVar(&config_path, "config", "", "path to config file")

	//faker generation
	rootCmd.PersistentFlags().Bool("scaffold", false, "generate new faker scaffold")
	rootCmd.PersistentFlags().StringVar(&scaffold_name, "scaffold_name", "", "name of new faker")

}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
