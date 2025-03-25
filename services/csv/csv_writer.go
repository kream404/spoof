package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kream404/scratch/fakers"
	"github.com/kream404/scratch/services/json"
	"github.com/kream404/scratch/models"
)

func GenerateCSV(config models.FileConfig, outputPath string) error {
	// Get project root directory
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %v", err)
	}

	// Construct the path to the root-level output directory
	outputDir := filepath.Join(rootDir, "output")
	outputFile := filepath.Join(outputDir, "output.csv")

	// Ensure the directory exists
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = rune(config.Config.Delimiter[0])
	defer writer.Flush()

	schema := config.Entities
	for _, entity := range schema {
		json, _ := json.ToJSONString(entity)
		fmt.Printf("Starting CSV generation for : %v", json)
		fmt.Println("")
		for i := 0; i < config.Config.RowCount; i++ {
			var record []string
			for _, field := range entity.Fields {
				var value any
				fmt.Printf("type: %s", field.Type)
				fmt.Println("")
				faker, _ := fakers.GetFakerByName(field.Type)
				switch f := faker.(type) {
				case *fakers.UUIDFaker:
					value, err = f.Generate()
				case *fakers.EmailFaker:
					value, err = f.Generate()
				case *fakers.PhoneFaker:
					value, err = f.Generate()
				default:
					return fmt.Errorf("unsupported faker type: %s", field.Type)
				}
				if err != nil {
					    fmt.Printf("Faker error: %v\n", err)
					return fmt.Errorf("error: %s", err)
				}
				// fmt.Printf("value: %s", value)
				record = append(record, fmt.Sprint(value))
			}

			fmt.Printf("record: %v\n", record)
			os.Stdout.Sync()
			if err := writer.Write(record); err != nil {
    			fmt.Printf("CSV Write Error: %v\n", err)
			}
		}
	}

	return nil
}
