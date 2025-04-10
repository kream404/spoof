package csv

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/kream404/spoof/fakers"
	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/database"
)

func GenerateCSV(config models.FileConfig, outputPath string) error {
	var seed []map[string]any
	var seedIndex = 0
	for _, file := range config.Files {
		outFile, err := MakeOutputDir(file.Config)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		writer := csv.NewWriter(outFile)
		writer.Comma = rune(file.Config.Delimiter[0])

		if file.Config.IncludeHeaders {
			var headers []string
			for _, field := range file.Fields {
				headers = append(headers, field.Name)
			}
			if err := writer.Write(headers); err != nil {
				return fmt.Errorf("CSV Write Error (headers): %v", err)
			}
		}

		if file.CacheConfig.HasCache(){
			seed, err = database.NewDBConnector().LoadCache(file.CacheConfig)
			if err != nil {
				println(fmt.Sprint(err))
			}
		}

		rng := rand.New(rand.NewSource(42))
		println("spoofing...")
		for range file.Config.RowCount {
			row, err := GenerateValues(file, seed, seedIndex, rng)
			if err != nil {
				log.Fatal(err)
			}
			if err := writer.Write(row); err != nil {
				fmt.Printf("CSV Write Error: %v\n", err)
			}
			seedIndex++
			if len(seed) > 0 && seedIndex >= len(seed) {
				seedIndex = 0
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

func GenerateValues(file models.Entity, seed []map[string]any, seedIndex int, rng *rand.Rand) ([]string, error) {
	var record []string
	for _, field := range file.Fields {
		var value any
		var key string
		if field.SeedType == "db" {
			if field.Alias != "" {
				key = field.Alias
			}else{
				key = field.Name
			}
			value = seed[seedIndex][key]
			// println("seeded value: ", fmt.Sprint(value))
		} else {
			factory, found := fakers.GetFakerByName(field.Type)
			if !found {
				return nil, fmt.Errorf("faker not found for type: %s", field.Type)
			}
			faker, err := factory(field, rng)
			if err != nil {
				return nil, fmt.Errorf("error creating faker for field %s: %w", field.Name, err)
			}
			value, err = faker.Generate()
			if err != nil {
				return nil, fmt.Errorf("error generating value for field %s: %w", field.Name, err)
			}
		}

		record = append(record, fmt.Sprint(value))
	}

	return record, nil
}
