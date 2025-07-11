package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/csv"
	log "github.com/kream404/spoof/services/logger"

	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:  "extract",
	Long: `extract a new config file`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := ExtractConfigFile(args[0])
		if err != nil {
			log.Error("Failed to extract config	", "error", err.Error())
			os.Exit(1)
		}

		directory, _ := os.Getwd()
		path := filepath.Join(directory, strings.TrimSuffix(config.Files[0].Config.FileName, ".csv")+".json")
		err = WriteConfigToFile(config, path)
		if err != nil {
			log.Error("Failed to write config file	", "error", err.Error())
		}
	},
}

func ExtractConfigFile(path string) (*models.FileConfig, error) {
	records, filename, delimiter, header, footer, err := csv.ReadCSV(path)
	if err != nil {
		log.Error("Failed to read csv	", "path", path)
		return nil, err
	}

	fields, types, _ := csv.MapFields(records)
	config := models.Config{
		FileName:       filepath.Base(filename),
		Delimiter:      string(delimiter),
		RowCount:       len(records) - 1,
		IncludeHeaders: true,
		Header:         header,
		Footer:         footer,
	}

	entity := models.Entity{
		Config:      config,
		CacheConfig: nil,
		Fields:      fields,
	}

	fileConfig := &models.FileConfig{
		Files: []models.Entity{entity},
	}

	log.Info("Extracted config")
	log.Debug("Summary	", "field_types", fmt.Sprint(types), "count", fmt.Sprint(len(entity.Fields)))

	return fileConfig, nil
}

func WriteConfigToFile(config *models.FileConfig, path string) error {

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Error("Error writing config file", "error", err.Error())
		return err
	}
	log.Info("Created new config file	", "path", path)
	return os.WriteFile(path, data, 0644)
}
