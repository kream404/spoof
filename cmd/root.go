package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-ini/ini"
	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/csv"
	log "github.com/kream404/spoof/services/logger"
	"golang.org/x/term"

	"github.com/spf13/cobra"
)

var (
	configPath   string
	cfg          *models.FileConfig
	scaffold     bool
	scaffoldName string
	profile      string
	showVersion  bool
	force        bool
	verbose      bool
	generate     bool
	extractPath  string
	injectVars   []string
)

// Root command
var rootCmd = &cobra.Command{
	Use:   "spoof",
	Short: "A tool for generating CSV files.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			versionCmd.Run(cmd, args)
			return nil
		}

		if verbose {
			log.Init(slog.LevelDebug)
		} else {
			log.Init(slog.LevelInfo)
		}

		if scaffold && scaffoldName != "" {
			runScaffold()
		}

		if extractPath != "" {
			log.Info("Extracting config file", "path", extractPath)
			extractCmd.Run(cmd, []string{extractPath})
			return nil
		}

		if generate {
			return runGenerate(cmd)
		}

		if err := loadConfig(); err != nil {
			log.Error("Failed to load config file", "error", err.Error())
			return err
		}

		if cfg == nil {
			return errors.New("no configuration loaded")
		}

		ProcessFiles(force)
		return nil
	},
}

func runGenerate(cmd *cobra.Command) error {
	reader := bufio.NewReader(os.Stdin)
	var genArgs []string

	prompt := func(q string) (string, error) {
		fmt.Print(q)
		ans, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(ans), nil
	}

	name, err := prompt("name of config file: ")
	if err != nil {
		return err
	}
	genArgs = append(genArgs, name)

	out, err := prompt("name of output file: ")
	if err != nil {
		return err
	}
	genArgs = append(genArgs, out)

	delim, err := prompt("delimiter: ")
	if err != nil {
		return err
	}
	genArgs = append(genArgs, delim)

	rows, err := prompt("row count: ")
	if err != nil {
		return err
	}
	genArgs = append(genArgs, rows)

	headers, err := prompt("include headers [Y/n]: ")
	if err != nil {
		return err
	}
	genArgs = append(genArgs, headers)

	generateCmd.Run(cmd, genArgs)
	return nil
}

func ProcessFiles(force bool) {
	csv.ProcessFiles(*cfg, force)
}

func runScaffold() {
	fmt.Println("generating faker..")
	fmt.Println("scaffold_name:", scaffoldName)

	fakerConfig := FakerConfig{
		Name:     scaffoldName,
		DataType: scaffoldName,
		Format:   "",
	}
	GenerateFaker(fakerConfig)
}

func loadConfig() error {
	log.Debug("Loading config", "path", configPath)

	raw, err := os.ReadFile(configPath)
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
	isBundle := false
	if err := json.Unmarshal(raw, &b); err == nil && len(b.Files) > 0 {
		for _, f := range b.Files {
			if strings.TrimSpace(f.Source) != "" {
				isBundle = true
				break
			}
		}
	}

	varsMap := parseInjectedVars(injectVars)
	if len(varsMap) > 0 {
		log.Info("Injecting variables into config", "vars", injectVars)
	}

	var loaded models.FileConfig
	if isBundle {
		for _, f := range b.Files {
			src := strings.TrimSpace(f.Source)
			if src == "" {
				continue
			}
			log.Info("Loading referenced config", "source", src)

			childRaw, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("failed to read referenced config %q: %w", src, err)
			}
			if len(varsMap) > 0 {
				childRaw, err = injectJSON(childRaw, varsMap)
				if err != nil {
					return fmt.Errorf("variable injection failed for %q: %w", src, err)
				}
			}

			var child models.FileConfig
			if err := json.Unmarshal(childRaw, &child); err != nil {
				return fmt.Errorf("failed to unmarshal referenced config %q: %w", src, err)
			}
			loaded.Files = append(loaded.Files, child.Files...)
		}

		if len(loaded.Files) == 0 {
			if len(varsMap) > 0 {
				raw, err = injectJSON(raw, varsMap)
				if err != nil {
					return fmt.Errorf("variable injection failed: %w", err)
				}
			}
			if err := json.Unmarshal(raw, &loaded); err != nil {
				return fmt.Errorf("no files found from bundle sources and generic load failed: %w", err)
			}
		}
	} else {
		if len(varsMap) > 0 {
			raw, err = injectJSON(raw, varsMap)
			if err != nil {
				return fmt.Errorf("variable injection failed: %w", err)
			}
		}
		if err := json.Unmarshal(raw, &loaded); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	if len(loaded.Files) == 0 {
		return errors.New("no files found in configuration")
	}

	if profile != "" {
		if err := applyProfile(&loaded, profile); err != nil {
			return err
		}
	}

	cfg = &loaded
	return nil
}

func applyProfile(fc *models.FileConfig, name string) error {
	log.Info("Using connection profile", "profile", name)
	home, _ := os.UserHomeDir()
	profilePath := filepath.Join(home, "/.config/spoof/profiles.ini")
	cfgIni, err := ini.Load(profilePath)
	if err != nil {
		return fmt.Errorf("could not load profile file: %w", err)
	}
	section := cfgIni.Section(name)

	cacheProfile := models.CacheConfig{
		Hostname: section.Key("hostname").String(),
		Port:     section.Key("port").String(),
		Username: section.Key("username").String(),
		Password: section.Key("password").String(),
		Name:     section.Key("name").String(),
	}
	if cacheProfile.Hostname == "" {
		log.Error("Failed to load connection profile", "err", "profile not found")
		return errors.New("profile not found")
	}
	if cacheProfile.Password == "" {
		fmt.Print("enter db password: ")
		pw, _ := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		cacheProfile.Password = string(pw)
	}

	for i := range fc.Files {
		if fc.Files[i].CacheConfig == nil {
			fc.Files[i].CacheConfig = &models.CacheConfig{}
		}
		merged := fc.Files[i].CacheConfig.MergeConfig(cacheProfile)
		fc.Files[i].CacheConfig = &merged
	}
	log.Debug("Profile applied to all files", "count", len(fc.Files))
	return nil
}

func parseInjectedVars(pairs []string) map[string]string {
	varsMap := make(map[string]string)
	for _, pair := range pairs {
		// allow: --inject KEY=VAL, --inject "KEY=VAL,FOO=BAR"
		for _, p := range strings.Split(pair, ",") {
			kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
			if len(kv) == 2 && kv[0] != "" {
				varsMap[kv[0]] = kv[1]
			}
		}
	}
	return varsMap
}

func injectJSON(raw []byte, vars map[string]string) ([]byte, error) {
	s := string(raw)
	pairs := make([]string, 0, len(vars)*2)
	for k, v := range vars {
		pairs = append(pairs, "{{"+k+"}}", v)
		pairs = append(pairs, "{{ "+k+" }}", v)
	}
	r := strings.NewReplacer(pairs...)
	s = r.Replace(s)

	unfilled := regexp.MustCompile(`\{\{\s*[^}]+\s*\}}`)
	if unfilled.MatchString(s) {
		log.Debug("Unfilled template tokens remain after injection")
	}
	return []byte(s), nil
}

func init() {
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show CLI version")
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "allow destructive operation")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "V", false, "show additional logs")
	rootCmd.Flags().BoolVarP(&generate, "generate", "g", false, "generate a new config file")
	rootCmd.Flags().StringArrayVarP(&injectVars, "inject", "i", []string{}, "variables to inject (key=value)")
	rootCmd.Flags().StringVarP(&extractPath, "extract", "e", "", "extract config file from csv")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to config file")
	rootCmd.Flags().StringVarP(&profile, "profile", "p", "", "db connection profile")
	rootCmd.Flags().BoolVarP(&scaffold, "scaffold", "s", false, "generate new faker scaffold")
	rootCmd.Flags().StringVarP(&scaffoldName, "scaffold_name", "n", "", "name of new faker")
}

func Execute() {
	start := time.Now()
	if err := rootCmd.Execute(); err != nil {
		log.Error("Uncaught error", "error", err.Error())
		os.Exit(1)
	}
	elapsed := time.Since(start)
	log.Info("Done in", "time", elapsed)
}
