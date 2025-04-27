package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	generate			 bool
)

var rootCmd = &cobra.Command{
	Use:   "spoof",
	Short: "A tool for generating csv's.",
	Run: func(cmd *cobra.Command, args []string) {

		// multiple returns, _ indicates ignore the error. neato
		version, _ := cmd.Flags().GetBool("version")
		verbose, _ := cmd.Flags().GetBool("verbose")
		scaffold, _ := cmd.Flags().GetBool("scaffold")
		generate, _ := cmd.Flags().GetBool("generate")

			if generate {
		    var genArgs []string
		    reader := bufio.NewReader(os.Stdin)

		    println("generating new config file.")

				print("name of config file: ")
		    output_name, _ := reader.ReadString('\n')
		    genArgs = append(genArgs, strings.TrimSpace(output_name))
		    println("config file in:", output_name)

		    print("filename: ")
		    file_name, _ := reader.ReadString('\n')
		    genArgs = append(genArgs, strings.TrimSpace(file_name))
		    println("filename in:", file_name)

		    print("delimiter: ")
		    delimiter, _ := reader.ReadString('\n')
		    genArgs = append(genArgs, strings.TrimSpace(delimiter))
		    println("delimiter in:", delimiter)

		    print("rowcount: ")
		    rowcount, _ := reader.ReadString('\n')
		    genArgs = append(genArgs, strings.TrimSpace(rowcount))
		    println("rowcount in:", rowcount)

		    print("include headers [Y/n]: ")
		    headers, _ := reader.ReadString('\n')
		    genArgs = append(genArgs, strings.TrimSpace(headers))
		    println("headers in:", headers)

		    fmt.Println("genArgs:", genArgs)
		    generateCmd.Run(cmd, genArgs)
		}


		if version {
			versionCmd.Run(cmd, args)
			return
		}

		//profile will always override config file - maybe this should be flipped
		if (profile != "" && config_path != "")  {
			fmt.Println("========================================")
			println("loading connection profile: ", profile)
			home, _ := os.UserHomeDir()
			cfg, _ := ini.Load(filepath.Join(home, "/.config/spoof/profiles.ini"))
			section := cfg.Section(profile)

			var err error
			config, err = json.LoadConfig(config_path)

			if err != nil {
				println("failed to load json config: ", fmt.Sprint(err))
				os.Exit(1)
			}
			profile := models.CacheConfig{
				Hostname: section.Key("hostname").String(),
				Port:     section.Key("port").String(),
				Username: section.Key("username").String(),
				Password: section.Key("password").String(),
				Name: section.Key("name").String(),
			}

			if profile.Password == "" {
				print("enter db password: ");
				input, _ := terminal.ReadPassword(0)
				profile.Password = string(input)
			}
			config.Files[0].CacheConfig = config.Files[0].CacheConfig.MergeConfig(profile)

		}else{
			config, _ = json.LoadConfig(config_path)
		}

		if(verbose && config != nil) {
			start := time.Now()
			fmt.Println("config path: ", config_path)
			fmt.Println("========================================")
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
	rootCmd.Flags().BoolVarP(&generate, "generate", "g", false, "generate a new config file")
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
