package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-ini/ini"
	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/csv"
	json_parser "github.com/kream404/spoof/services/json"
	log "github.com/kream404/spoof/services/logger"
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
	force         bool
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

		if verbose {
			log.Init(slog.LevelDebug)
		} else {
			log.Init(slog.LevelInfo)
		}

		if scaffold && scaffold_name != "" {
			runScaffold()
		}

		if extract_path != "" {
			log.Info("Extracting config file", "path", extract_path)
			runExtract(cmd, extract_path)
			return
		}

		if generate {
			runGenerate(cmd)
			return
		}

		if err := loadConfig(); err != nil {
			log.Error("Failed to load config file ", "error", err.Error())
			os.Exit(1)
		}

		if config != nil {
			ProcessFiles(force)
		}

	},
}

func runVersion(cmd *cobra.Command, args []string) {
	versionCmd.Run(cmd, args)
}

func runExtract(cmd *cobra.Command, path string) {
	extractCmd.Run(cmd, []string{path})
}

func runGenerate(cmd *cobra.Command) {
	reader := bufio.NewReader(os.Stdin)
	var genArgs []string
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

func ProcessFiles(force bool) {
	csv.ProcessFiles(*config, force)
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
	log.Debug("Loading config", "path", config_path)

	raw, err := os.ReadFile(config_path)
	if err != nil {
		return fmt.Errorf("could not read config: %w", err)
	}

	type bundleFile struct {
		Source string `json:"source"`
	}
	type bundle struct {
		Files []bundleFile `json:"files"`
	}

	var b bundle
	var loaded models.FileConfig

	parseErr := json.Unmarshal(raw, &b)
	isBundle := false
	if parseErr == nil && len(b.Files) > 0 {
		for _, f := range b.Files {
			if strings.TrimSpace(f.Source) != "" {
				isBundle = true
				break
			}
		}
	}

	if isBundle {
		for _, f := range b.Files {
			src := strings.TrimSpace(f.Source)
			if src == "" {
				continue
			}

			log.Info("Loading referenced config", "source", src)
			c, err := json_parser.LoadConfig(src)
			if err != nil {
				return fmt.Errorf("failed to load referenced config %q: %w", src, err)
			}
			loaded.Files = append(loaded.Files, c.Files...)
		}

		if len(loaded.Files) == 0 {
			cfg, err := json_parser.LoadConfig(config_path)
			if err != nil {
				return fmt.Errorf("no files found from bundle sources and generic load failed: %w", err)
			}
			loaded = *cfg
		}
	} else {
		cfg, err := json_parser.LoadConfig(config_path)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		loaded = *cfg
	}

	if len(loaded.Files) == 0 {
		return fmt.Errorf("no files found in configuration")
	}

	if profile != "" {
		log.Info("Using connection profile", "profile", profile)
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
		if cacheProfile.Hostname == "" {
			log.Error("Failed to load connection profile", "err", "profile not found")
			os.Exit(1)
		}
		if cacheProfile.Password == "" {
			fmt.Print("enter db password: ")
			input, _ := terminal.ReadPassword(0)
			fmt.Println()
			cacheProfile.Password = string(input)
		}

		for i := range loaded.Files {
			if loaded.Files[i].CacheConfig == nil {
				loaded.Files[i].CacheConfig = &models.CacheConfig{}
			}
			merged := loaded.Files[i].CacheConfig.MergeConfig(cacheProfile)
			loaded.Files[i].CacheConfig = &merged
		}
		log.Debug("Profile applied to all files", "count", len(loaded.Files))
	}

	config = &loaded
	return nil
}

func init() {
	rootCmd.Flags().BoolVarP(&version, "version", "v", false, "show cli version")
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "allow destructive operation")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "V", false, "show additional logs")
	rootCmd.Flags().BoolVarP(&generate, "generate", "g", false, "generate a new config file")
	rootCmd.Flags().StringVarP(&extract_path, "extract", "e", "", "extract config file from csv")
	rootCmd.Flags().StringVarP(&config_path, "config", "c", "", "path to config file")
	rootCmd.Flags().StringVarP(&profile, "profile", "p", "", "db connection profile")
	rootCmd.Flags().BoolVarP(&scaffold, "scaffold", "s", false, "generate new faker scaffold")
	rootCmd.Flags().StringVarP(&scaffold_name, "scaffold_name", "n", "", "name of new faker")
}

func Execute() {
	start := time.Now()
	if err := rootCmd.Execute(); err != nil {
		log.Error("Uncaught error ", "error", err.Error())
		os.Exit(1)
	}
	elapsed := time.Since(start)
	log.Info("Done in ", "time", elapsed)
}
