package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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
var profile string

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

		if profile != "" {
			println("profile provided. loading connection profile: ", profile)
			home, _ := os.UserHomeDir()
			path := filepath.Join(home, "/.config/spoof/profiles.json")
			_, err := json.LoadProfiles(path)
			if err != nil {
				panic(err)
			}
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
	rootCmd.PersistentFlags().BoolP("version", "v", false, "show cli version")
	rootCmd.PersistentFlags().BoolP("verbose", "V", false, "show additional logs")
	rootCmd.PersistentFlags().StringVarP(&config_path, "config", "c", "", "path to config file")
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "db connection profile")

	rootCmd.PersistentFlags().BoolP("scaffold", "s", false, "generate new faker scaffold")
	rootCmd.PersistentFlags().StringVarP(&scaffold_name, "scaffold_name", "n", "", "name of new faker")


}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
