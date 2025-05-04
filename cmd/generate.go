package cmd

import (
	"os"
	"strconv"
	"strings"
	"text/template"

	log "github.com/kream404/spoof/services/logger"
	"github.com/spf13/cobra"
)

type GenerateConfig struct {
	ConfigFileName string
	FileName       string
	Delimiter      string
	Rowcount       int
	Headers        bool
}

var generateCmd = &cobra.Command{
	Use:  "generate",
	Long: `generate a new config file`,
	Run: func(cmd *cobra.Command, args []string) {
		fileName := strings.ToLower(args[1])
		log.Debug("Generaating new config file ", "filename", fileName)
		if !strings.HasSuffix(fileName, ".csv") {
			fileName += ".csv"
		}

		rowcount, err := strconv.Atoi(args[3])
		if err != nil {
			log.Error("Invalid row count ", "error", err)
			os.Exit(1)
		}

		s := args[4]
		headers := false
		if strings.ToLower(s) == "y" {
			headers = true
		}

		config := GenerateConfig{
			ConfigFileName: args[0],
			FileName:       fileName,
			Delimiter:      args[2],
			Rowcount:       rowcount,
			Headers:        headers,
		}

		GenerateConfigFile(config)
	},
}

const GenerateTemplate = `{
  "files": [{
    "config": {
      "file_name": "{{ .FileName }}",
      "delimiter": "{{ .Delimiter }}",
      "rowcount": "{{ .Rowcount }}",
      "include_headers": {{ .Headers }}
    },
    "cache": {
    },
    "fields": [
      { "name": "placeholder", "type": "placeholder" }
    ]
  }]
}`

func GenerateConfigFile(config GenerateConfig) error {
	fileName := strings.ToLower(config.ConfigFileName)
	if !strings.HasSuffix(fileName, ".json") {
		fileName += ".json"
	}

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	tmpl, err := template.New("GenerateTemplate").Parse(GenerateTemplate)
	if err != nil {
		log.Error("Error parsing template:", "error", err.Error())
		return err
	}

	err = tmpl.Execute(f, config)
	if err != nil {
		log.Error("Error executing template:", "error", err.Error())
		return err
	}

	filepath, _ := os.Getwd()
	log.Info("========================================")
	log.Info("Config file generated ", "file", filepath+"/"+fileName)
	log.Info("========================================")
	return nil
}
