package csv

import (
	"encoding/csv"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/kream404/spoof/fakers"
	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/database"
	log "github.com/kream404/spoof/services/logger"
	"github.com/shopspring/decimal"
)

func GenerateCSV(config models.FileConfig, outputPath string) error {
	var cache []map[string]any
	var cacheIndex, rowIndex = 0, 1 //cacheindex tracks row in cache, rowindex tracks row in file..

	for _, file := range config.Files {
		outFile, err := MakeOutputDir(file.Config)
		if err != nil {
			log.Error("failed to create output file", "error", err)
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

		if file.CacheConfig != nil {
			cache, err = database.NewDBConnector().LoadCache(*file.CacheConfig)
			if err != nil {
				log.Error("Failed to load cache ", "error", err)
				os.Exit(1) //if we provide a cache, and cant populate it throw an error
			}
		}

		//TODO: look into new spinner that works with slog

		// rng := rand.New(rand.NewSource(42))
		rng := CreateRNGSeed(file.Config.Seed)
		for range file.Config.RowCount {
			row, err := GenerateValues(file, cache, rowIndex, cacheIndex, rng)
			if err != nil {
				log.Error("Row Error ", "error", err)
				os.Exit(1)
			}
			if err := writer.Write(row); err != nil {
				log.Error("CSV write error ", "error", err)
				os.Exit(1)
			}

			cacheIndex++ //increment pointers
			rowIndex++

			if len(cache) > 0 && cacheIndex >= len(cache) {
				cacheIndex = 0
			}
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			log.Error("CSV write error ", "error", err)
			os.Exit(1)
		}

		outFile.Close()
	}
	return nil
}

func MakeOutputDir(config models.Config) (*os.File, error) {
	outputDir := "output"
	outputFile := filepath.Join(outputDir, config.FileName)

	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return nil, err
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return nil, err
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

		case field.Type == "reflection":
			targetValue, ok := generatedFields[field.Target]
			if !ok {
				log.Error("Reflection error", "field_name", field.Name)
				if field.Target == "" {
					return nil, fmt.Errorf("You must provide a 'target' to use reflection")
				}
				return nil, fmt.Errorf("reflection target '%s' not found in previous fields", field.Target)
			}

			value = targetValue
			if field.Modifier != nil {
				modifiedValue, err := modifier(targetValue, *field.Modifier)
				if err != nil {
					log.Error("modifier ignored: not a valid number", "target", targetValue)
					return nil, err
				}
				value = modifiedValue
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
				log.Error("Error creating faker", "field_name", field.Name, "type", field.Type)
				return nil, fmt.Errorf("%w", err)
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

func CreateRNGSeed(seed_in string) *rand.Rand {
	var s string
	var seed int64
	if seed_in != "" {
		s = seed_in
	} else {
		s = uuid.NewString()
	}
	log.Info("============================================")
	log.Info("", "seed", s)
	log.Info("============================================")

	seed = stringToSeed(s)
	return rand.New(rand.NewSource(seed))
}

func modifier(raw string, modifier float64) (string, error) {
	decimalValue, err := decimal.NewFromString(raw)
	if err != nil {
		return "", fmt.Errorf("invalid number: %v", err)
	}
	modifiedValue := decimalValue.Mul(decimal.NewFromFloat(modifier))

	decimals := 0
	if dot := strings.Index(raw, "."); dot != -1 {
		decimals = len(raw) - dot - 1
	}

	return modifiedValue.StringFixed(int32(decimals)), nil
}
