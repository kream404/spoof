package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/csv"

	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:  "extract",
	Long: `extract a new config file`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := ExtractConfigFile(args[0])
		if err != nil {
			println("failed to extract config: ", err.Error())
		}

		directory, _ := os.Getwd()
		path := filepath.Join(directory, strings.TrimSuffix(config.Files[0].Config.FileName, ".csv")+".json")
		err = WriteConfigToFile(config, path)
		if err != nil {
			println("failed to write file: ", err.Error())
		}
	},
}

func ExtractConfigFile(path string) (*models.FileConfig, error) {
	records, file, err := csv.ReadCSV(path)
	if err != nil {
		return nil, err
	}

	delimiter := csv.DetectDelimiter(records[0][0])
	fields, _ := csv.MapFields(records)

	config := models.Config{
		FileName:       filepath.Base(file.Name()),
		Delimiter:      string(delimiter),
		RowCount:       100,
		IncludeHeaders: false,
	}

	entity := models.Entity{
		Config:      config,
		CacheConfig: nil,
		Fields:      fields,
	}

	fileConfig := &models.FileConfig{
		Files: []models.Entity{entity},
	}

	return fileConfig, nil
}

func WriteConfigToFile(config *models.FileConfig, outputPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		println(err)
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}
