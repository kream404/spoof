package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kream404/scratch/fakers"
	"github.com/kream404/scratch/models"
)

//should probably refactor the file config, config.Config is silly
func GenerateCSV(config models.FileConfig, outputPath string) error {
	//setup
	for _, file := range config.Files {
		outFile, err := MakeOutputDir(file.Config)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outFile.Close() //defer close until wrtite is complete

		writer := csv.NewWriter(outFile)
		writer.Comma = rune(file.Config.Delimiter[0])

		//append headers first
		if file.Config.IncludeHeaders {

			var headers []string
			for _, field := range file.Fields {
				headers = append(headers, field.Name)
			}
			if err := writer.Write(headers); err != nil {
				return fmt.Errorf("CSV Write Error (headers): %v", err)
			}
		}

		var value any
		for range file.Config.RowCount {
			var record []string

			for _, field := range file.Fields {
				faker, _ := fakers.GetFakerByName(field.Type)
				switch f := faker.(type) {
				case *fakers.UUIDFaker:
					f = fakers.NewUUIDFaker(field.Format)
					value, err = f.Generate()
				case *fakers.EmailFaker:
					f = fakers.NewEmailFaker(field.Format)
					value, err = f.Generate()
				case *fakers.PhoneFaker:
					f = fakers.NewPhoneFaker(field.Format)
					value, err = f.Generate()
				case *fakers.TimestampFaker:
					f = fakers.NewTimestampFaker(field.Format)
					value, err = f.Generate()
				default:
					return fmt.Errorf("unsupported faker type: %s", field.Type)
				}
				if err != nil {
					fmt.Printf("Faker error: %v\n", err)
					return fmt.Errorf("error: %s", err)
				}
				record = append(record, fmt.Sprint(value))
			}

			os.Stdout.Sync()
			if err := writer.Write(record); err != nil {
   			fmt.Printf("CSV Write Error: %v\n", err)
			}
		}
	}
	return nil
}

func MakeOutputDir(config models.Config) (*os.File, error) {
	rootDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	outputDir := filepath.Join(rootDir, "output")
	outputFile := filepath.Join(outputDir, config.FileName)

	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return file, nil
}
