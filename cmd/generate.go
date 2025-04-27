package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

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
	Use:   "generate",
	Long:  `generate a new config file`,
	Run: func(cmd *cobra.Command, args []string) {
		fileName := strings.ToLower(args[1])
		if !strings.HasSuffix(fileName, ".csv") {
			fileName += ".csv"
		}

		rowcount, err := strconv.Atoi(args[3])
		if err != nil {
			println("invalid rowcount: ", err)
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

		// Call to generate the config file
		GenerateConfigFile(config)
	},
}

const GenerateTemplate = `{
  "files": [{
    "config": {
      "file_name": "{{ .FileName }}",
      "delimiter": "{{ .Delimiter }}",
      "rowcount": {{ .Rowcount }},
      "include_headers": {{ .Headers }}
    },
    "cache": {
      "statement": ""
    },
    "fields": [
      { "name": "id", "type": "iterator" }
    ]
  }]
}`

func GenerateConfigFile(config GenerateConfig) error {
	fileName := strings.ToLower(config.ConfigFileName)
	if !strings.HasSuffix(fileName, ".json") {
		fileName += ".json"
	}

	// Create the file
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	// Parse the template
	tmpl, err := template.New("GenerateTemplate").Parse(GenerateTemplate)
	if err != nil {
		println("Error parsing template:", err)
		return err
	}

	// Execute the template
	err = tmpl.Execute(f, config)
	if err != nil {
		println("Error executing template:", err)
		return err
	}

	// Read the generated file content for debugging
	content, _ := os.ReadFile(fileName)
	fmt.Printf("Generated content:\n%s\n", string(content))

	return nil
}
