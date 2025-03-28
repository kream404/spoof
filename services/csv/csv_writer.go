package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kream404/scratch/fakers"
	"github.com/kream404/scratch/models"
)

//helper method to generate file.
func GenerateCSV(config models.FileConfig, outputPath string) error {
	for _, file := range config.Files {
		outFile, err := MakeOutputDir(file.Config)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		writer := csv.NewWriter(outFile)
		writer.Comma = rune(file.Config.Delimiter[0])

		// Write headers if required
		if file.Config.IncludeHeaders {
			var headers []string
			for _, field := range file.Fields {
				headers = append(headers, field.Name)
			}
			if err := writer.Write(headers); err != nil {
				return fmt.Errorf("CSV Write Error (headers): %v", err)
			}
		}

		for range file.Config.RowCount {
			row, err := GenerateValues(file)
			if err != nil {
				fmt.Printf("Row generation error: %v\n", err)
				continue
			}
			if err := writer.Write(row); err != nil {
				fmt.Printf("CSV Write Error: %v\n", err)
			}
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			return fmt.Errorf("CSV writer error: %w", err)
		}

		outFile.Close()
	}
	return nil
}

//returns pointer to output file
func MakeOutputDir(config models.Config) (*os.File, error) {
	outputDir := "output"
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

//generate row todo: refactor entity to be fileschema or some better name
func GenerateValues(file models.Entity) ([]string, error) {
	var record []string
	var value any
	var err error

	//i wish there was a better way to do this...
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
		case *fakers.RangeFaker:
				f = fakers.NewRangeFaker(field.Format, field.Values)
				value, err = f.Generate()
		default:
			return nil, fmt.Errorf("unsupported faker type: %s", field.Type)
		}

		if err != nil {
			return nil, fmt.Errorf("Faker error: %w", err)
		}

		record = append(record, fmt.Sprint(value))
	}
	return record, nil
}
