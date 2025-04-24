package csv

import (
	"encoding/csv"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/kream404/spoof/fakers"
	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/database"

	"time"

	"github.com/briandowns/spinner"
)

func GenerateCSV(config models.FileConfig, outputPath string) error {
	var cache []map[string]any
	var cacheIndex, rowIndex = 0, 1 //cacheindex tracks row in cache, rowindex tracks row in file..

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
			cache, err = database.NewDBConnector().LoadCache(file.CacheConfig)
			if err != nil {
				println(fmt.Sprint(err))
				os.Exit(1) //if we provide a cache, and cant populate it throw an error
			}
		}

		// start spinner
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Prefix = "spoofing	"
		s.Start()

		// rng := rand.New(rand.NewSource(42))
		rng := CreateRNGSeed(file.CacheConfig);
		for range file.Config.RowCount {
			row, err := GenerateValues(file, cache, rowIndex, cacheIndex, rng)
			if err != nil {
				fmt.Printf("Row Error: %v\n", err)
				os.Exit(1)
			}
			if err := writer.Write(row); err != nil {
				fmt.Printf("CSV Write Error: %v\n", err)
			}

			cacheIndex++ //increment pointers
			rowIndex++

			if len(cache) > 0 && cacheIndex >= len(cache) {
				cacheIndex = 0
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

func GenerateValues(file models.Entity, seed []map[string]any, rowIndex int, seedIndex int, rng *rand.Rand) ([]string, error) {
	var record []string
	generatedFields := make(map[string]string)

	for _, field := range file.Fields {
		var value any
		var key string

		switch {
			case field.Type == "" && field.Value != "":
				value = field.Value

			case field.SeedType == "db":
				if field.Alias != "" {
					key = field.Alias
				} else {
					key = field.Name
				}
				value = seed[seedIndex][key]

			case field.Type == "reflection": //TODO: I dont like this, this could get out of hand quickly. not sure where this should live
				targetValue, ok := generatedFields[field.Target]
				if !ok {
					return nil, fmt.Errorf("reflection target '%s' not found in previous fields", field.Target)
				}

				value = targetValue
				if field.Modifier != nil {
					if parsed, err := strconv.ParseFloat(targetValue, 64); err == nil {
						value = parsed * *field.Modifier
					} else {
						fmt.Printf("modifier ignored: '%s' is not a numeric string\n", targetValue)
					}
				}

			case field.Type == "iterator":
				value = rowIndex

			default:
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

		valueStr := fmt.Sprint(value)
		generatedFields[field.Name] = valueStr
		record = append(record, valueStr)
	}
	return record, nil
}


func stringToSeed(s string) int64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return int64(h.Sum64())
}

func CreateRNGSeed(config models.CacheConfig) *rand.Rand {
	var seed int64
	if config.HasSeed() {
		println("seed provided: ", config.Seed);
		seed = stringToSeed(config.Seed)
	}else{
		s := uuid.NewString()
		println("seed generated: ", s);
		seed = stringToSeed(s)
	}
	return rand.New(rand.NewSource(seed))
}
