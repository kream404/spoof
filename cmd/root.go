package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-ini/ini"
	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/csv"
	"github.com/kream404/spoof/services/json"

	"github.com/spf13/cobra"
)

var (
	config_path    string
	config         models.FileConfig
	scaffold       bool
	scaffold_name  string
	profile        string
	versionFlag    bool
	verbose        bool
)

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
		//TODO: parse to map and fetch profile, inject into cache config - could also be provided in config
		// could set profile in env config and fetch from there
		if profile != "" {
			println("profile provided. loading connection profile: ", profile)
			home, _ := os.UserHomeDir()
			path := filepath.Join(home, "/.config/spoof/profiles.ini")
			cfg, _ := ini.Load(path)
			if cfg.HasSection(profile) {
				println("setting profile: ", profile)
			    section := cfg.Section(profile)
			    dbProfile := models.Profile{
			        Hostname: section.Key("db_hostname").String(),
			        Port:     section.Key("db_port").String(),
			        Username: section.Key("db_username").String(),
			        Password: section.Key("db_password").String(),
			    }
			    fmt.Println("DB Hostname:", dbProfile.Hostname)
			} else {
			    println("profile not found in config file: ", profile)
				println("set profile in /home/.config/spoof/profiles.ini")
			    os.Exit(1)
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
	// Use VarP to bind directly to variables
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "show cli version")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "V", false, "show additional logs")
	rootCmd.Flags().StringVarP(&config_path, "config", "c", "", "path to config file")
	rootCmd.Flags().StringVarP(&profile, "profile", "p", "", "db connection profile")
	rootCmd.Flags().BoolVarP(&scaffold, "scaffold", "s", false, "generate new faker scaffold")
	rootCmd.Flags().StringVarP(&scaffold_name, "scaffold_name", "n", "", "name of new faker")
}
// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
