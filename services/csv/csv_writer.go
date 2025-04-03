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

		seed := int64(42) // Change this to any seed value for deterministic results
		rng := rand.New(rand.NewSource(seed))

		for range file.Config.RowCount {
			row, err := GenerateValues(file, rng)
			if err != nil {
				log.Fatal(err)
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

func GenerateValues(file models.Entity, rng *rand.Rand) ([]string, error) {
	var record []string
	var value any
	var err error

	database, err := database.NewDBConnector().OpenConnection(file.CacheConfig);
	if err != nil{
		println("failed to connect db...", err)
	}
	database.FetchRows("SELECT * FROM account.customer;")
	database.CloseConnection();

	//conditionally pull from db here ?
	for _, field := range file.Fields {
		faker, _ := fakers.GetFakerByName(field.Type)
		switch f := faker.(type) {
		case *fakers.UUIDFaker:
			f = fakers.NewUUIDFaker(field.Format, rng)
			value, err = f.Generate()
		case *fakers.EmailFaker:
			fmt.Printf("seeded value: %d\n", rng.Int())
			f = fakers.NewEmailFaker(field.Format, rng)
			value, err = f.Generate()
		case *fakers.PhoneFaker:
			f = fakers.NewPhoneFaker(field.Format, rng)
			value, err = f.Generate()
		case *fakers.TimestampFaker:
			f = fakers.NewTimestampFaker(field.Format, rng)
			value, err = f.Generate()
		case *fakers.RangeFaker:
			f = fakers.NewRangeFaker(field.Format, field.Values, rng)
			value, err = f.Generate()
		case *fakers.NumberFaker:
			f = fakers.NewNumberFaker(field.Format, field.Min, field.Max, rng)
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
