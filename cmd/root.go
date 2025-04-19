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
	"golang.org/x/crypto/ssh/terminal"

	"github.com/spf13/cobra"
)

var (
	config_path    string
	config         *models.FileConfig
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

		//profile will always override config file - maybe this should be flipped
		if (profile != "" && config_path != "")  {
			fmt.Println("=================================")
			println("loading connection profile: ", profile)
			home, _ := os.UserHomeDir()
			cfg, _ := ini.Load(filepath.Join(home, "/.config/spoof/profiles.ini"))
			section := cfg.Section(profile)

			config, _ = json.LoadConfig(config_path)

			profileCache := models.CacheConfig{
				Hostname: section.Key("db_hostname").String(),
				Port:     section.Key("db_port").String(),
				Username: section.Key("db_username").String(),
				Password: section.Key("db_password").String(),
				Name: section.Key("db_name").String(),
			}

			if profileCache.Password == "" {
				print("enter db password: ");
				input, _ := terminal.ReadPassword(0)
				profileCache.Password = string(input)
				println("from input: ", profileCache.Password)
			}
			config.Files[0].CacheConfig = config.Files[0].CacheConfig.MergeConfig(profileCache)

		}else{
			config, _ = json.LoadConfig(config_path)
		}

		if(verbose && config != nil) {
			start := time.Now()
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
	rootCmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "show cli version")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "V", false, "show additional logs")
	rootCmd.Flags().StringVarP(&config_path, "config", "c", "", "path to config file")
	rootCmd.Flags().StringVarP(&profile, "profile", "p", "", "db connection profile")
	rootCmd.Flags().BoolVarP(&scaffold, "scaffold", "s", false, "generate new faker scaffold")
	rootCmd.Flags().StringVarP(&scaffold_name, "scaffold_name", "n", "", "name of new faker")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
