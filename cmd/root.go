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
	config_path   string
	config        *models.FileConfig
	scaffold      bool
	scaffold_name string
	profile       string
	version       bool
	verbose       bool
	generate      bool
	extract_path  string
)

var rootCmd = &cobra.Command{
	Use:   "spoof",
	Short: "A tool for generating csv's.",
	Run: func(cmd *cobra.Command, args []string) {
		if version {
			runVersion(cmd, args)
			return
		}

		if extract_path != "" {
			runExtract(cmd, extract_path)
			return
		}

		if generate {
			runGenerate(cmd)
			return
		}

		if err := loadConfig(); err != nil {
			fmt.Println("failed to load config:", err)
			os.Exit(1)
		}

		if config != nil {
			runVerboseCSV()
		}

		if scaffold && scaffold_name != "" {
			runScaffold()
		}
	},
}

func runVersion(cmd *cobra.Command, args []string) {
	versionCmd.Run(cmd, args)
}

func runExtract(cmd *cobra.Command, path string) {
	fmt.Println("extracting config from file: ", path)
	extractCmd.Run(cmd, []string{path})
}

func runGenerate(cmd *cobra.Command) {
	reader := bufio.NewReader(os.Stdin)
	var genArgs []string

	fmt.Println("generating new config file..")
	fmt.Println("========================================")

	fmt.Print("name of config file: ")
	outputName, _ := reader.ReadString('\n')
	genArgs = append(genArgs, strings.TrimSpace(outputName))

	fmt.Print("name of output file: ")
	fileName, _ := reader.ReadString('\n')
	genArgs = append(genArgs, strings.TrimSpace(fileName))

	fmt.Print("delimiter: ")
	delimiter, _ := reader.ReadString('\n')
	genArgs = append(genArgs, strings.TrimSpace(delimiter))

	fmt.Print("row count: ")
	rowCount, _ := reader.ReadString('\n')
	genArgs = append(genArgs, strings.TrimSpace(rowCount))

	fmt.Print("include headers [Y/n]: ")
	headers, _ := reader.ReadString('\n')
	genArgs = append(genArgs, strings.TrimSpace(headers))

	generateCmd.Run(cmd, genArgs)
}

func runVerboseCSV() {
	start := time.Now()
	fmt.Println("config path:", config_path)
	fmt.Println("========================================")
	csv.GenerateCSV(*config, "./output/output.csv")
	elapsed := time.Since(start)
	fmt.Printf("\n⏱️  Done in %s\n", elapsed)
}

func runScaffold() {
	fmt.Println("generating faker..")
	fmt.Println("scaffold_name:", scaffold_name)

	fakerConfig := FakerConfig{
		Name:     scaffold_name,
		DataType: scaffold_name,
		Format:   "",
	}
	GenerateFaker(fakerConfig)
}

func loadConfig() error {
	if config_path == "" {
		return fmt.Errorf("--config path is required")
	}

	var err error
	config, err = json.LoadConfig(config_path)
	if err != nil {
		return err
	}

	if profile != "" {
		fmt.Println("========================================")
		fmt.Println("loading connection profile:", profile)
		home, _ := os.UserHomeDir()
		profilePath := filepath.Join(home, "/.config/spoof/profiles.ini")
		cfg, err := ini.Load(profilePath)
		if err != nil {
			return fmt.Errorf("could not load profile file: %w", err)
		}

		section := cfg.Section(profile)
		cacheProfile := models.CacheConfig{
			Hostname: section.Key("hostname").String(),
			Port:     section.Key("port").String(),
			Username: section.Key("username").String(),
			Password: section.Key("password").String(),
			Name:     section.Key("name").String(),
		}

		if cacheProfile.Password == "" {
			fmt.Print("enter db password: ")
			input, _ := terminal.ReadPassword(0)
			cacheProfile.Password = string(input)
		}

		merged := config.Files[0].CacheConfig.MergeConfig(cacheProfile)
		config.Files[0].CacheConfig = &merged
	}

	return nil
}

func init() {
	rootCmd.Flags().BoolVarP(&version, "version", "v", false, "show cli version")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "V", false, "show additional logs")
	rootCmd.Flags().BoolVarP(&generate, "generate", "g", false, "generate a new config file")
	rootCmd.Flags().StringVarP(&extract_path, "extract", "e", "", "extract config file from csv")
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
