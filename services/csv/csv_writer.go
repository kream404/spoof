package csv

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/kream404/spoof/fakers"
	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/database"
	log "github.com/kream404/spoof/services/logger"
	s3c "github.com/kream404/spoof/services/s3"
	"github.com/shopspring/decimal"
)

func GenerateCSV(config models.FileConfig, outputPath string) error {
	var cache []map[string]any
	var cacheIndex, rowIndex = 0, 1

	for _, file := range config.Files {
		cacheIndex = 0
		cache = nil
		rowIndex = 1

		outFile, localPath, err := MakeOutputDir(file.Config)
		if err != nil {
			log.Error("failed to create output file", "error", err)
			return err
		}

		tempWriter := &strings.Builder{}
		writer := csv.NewWriter(tempWriter)
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
			if strings.Contains(file.CacheConfig.Source, ".csv") {
				log.Debug("Loading CSV cache", "source", file.CacheConfig.Source)
				cache, _, _, _ = ReadCSVAsMap(file.CacheConfig.Source)
			} else {
				cache, err = database.NewDBConnector().LoadCache(*file.CacheConfig)
				if err != nil {
					log.Error("Failed to load cache ", "error", err)
					os.Exit(1)
				}
			}
		}

		fieldCaches := preloadFieldSources(file.Fields)
		rng := CreateRNGSeed(file.Config.Seed)
		for i := 0; i < file.Config.RowCount; i++ {
			row, err := GenerateValues(file, cache, fieldCaches, rowIndex, cacheIndex, rng)
			if err != nil {
				log.Error("Row Error ", "error", err)
				os.Exit(1)
			}
			if err := writer.Write(row); err != nil {
				log.Error("CSV write error ", "error", err)
				os.Exit(1)
			}

			cacheIndex++
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

		finalWriter := bufio.NewWriter(outFile)

		if file.Config.Header != "" {
			_, _ = finalWriter.WriteString(file.Config.Header + "\n")
		}

		_, _ = finalWriter.WriteString(tempWriter.String())

		if file.Config.Footer != "" {
			_, _ = finalWriter.WriteString(file.Config.Footer + "\n")
		}

		if err := finalWriter.Flush(); err != nil {
			log.Error("Failed to flush final writer", "error", err)
			os.Exit(1)
		}

		outFile.Close()

		if file.Postprocess.Upload && strings.HasPrefix(file.Postprocess.Location, "s3://") {
			ctx := context.Background()

			s3, err := s3c.NewS3Connector().OpenConnection(models.CacheConfig{}, file.Postprocess.Region)
			if err != nil {
				return fmt.Errorf("failed to init S3 connector: %w", err)
			}

			dest, err := joinS3URI(file.Postprocess.Location, file.Config.FileName)
			if err != nil {
				return err
			}

			if _, err := s3.UploadFile(ctx, dest, localPath); err != nil {
				log.Error("Failed to upload file to S3", "dest", dest, "error", err)
				return err
			}

			log.Info("Uploaded CSV to S3", "uri", dest)
		}
	}
	return nil
}

func GenerateValues(file models.Entity, cache []map[string]any, fieldSources fieldCache, rowIndex int, seedIndex int, rng *rand.Rand) ([]string, error) {
	var record []string
	generatedFields := make(map[string]string)

	for _, field := range file.Fields {
		var value any
		var key string
		if field.Alias != "" {
			key = field.Alias
		} else {
			key = field.Name
		}

		injected := false
		if shouldInjectFromSource(field, rng) && field.Source != "" && strings.Contains(field.Source, ".csv") {
			if rows, ok := fieldSources[field.Source]; ok && len(rows) > 0 {
				idx := seedIndex % len(rows)
				if val := rows[idx][key]; val != nil {
					value = val
					injected = true
					log.Debug("Injecting value from preloaded source CSV",
						"source", field.Source, "field", field.Name, "value", value)
				} else {
					log.Warn("Key not found in preloaded source row; falling back",
						"key", key, "field", field.Name, "source", field.Source)
				}
			} else {
				log.Debug("No preloaded rows for source; skipping injection", "source", field.Source)
			}
		}

		if !injected {
			switch {
			case field.Seed:
				value = cache[seedIndex][key]

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

			case field.Type == "":
				value = field.Value

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
		}

		var valueStr string
		if value == nil {
			valueStr = ""
		} else {
			valueStr = fmt.Sprint(value)
		}

		generatedFields[field.Name] = valueStr
		record = append(record, valueStr)
	}
	return record, nil
}

type fieldCache map[string][]map[string]any

func preloadFieldSources(fields []models.Field) fieldCache {
	fieldCache := make(fieldCache)
	seen := make(map[string]struct{})
	for _, f := range fields {
		if f.Source == "" || !strings.Contains(f.Source, ".csv") {
			continue
		}
		if _, ok := seen[f.Source]; ok {
			continue
		}
		rows, _, _, err := ReadCSVAsMap(f.Source)
		if err != nil {
			log.Error("Failed to preload source CSV; will skip injection for this source",
				"source", f.Source, "err", err)
			continue
		}
		fieldCache[f.Source] = rows
		seen[f.Source] = struct{}{}
		log.Debug("Preloaded CSV source", "source", f.Source, "rows", len(rows))
	}
	return fieldCache
}

func shouldInjectFromSource(field models.Field, rng *rand.Rand) bool {
	if field.Source == "" {
		return false
	}
	// default to 100% if rate is omitted
	if field.Rate == nil {
		return true
	}
	r := *field.Rate
	if r <= 0 {
		return false
	}
	if r >= 100 {
		return true
	}
	return rng.Intn(100) < r // 0..99 < r
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

func MakeOutputDir(config models.Config) (*os.File, string, error) {
	outputDir := "output"
	var outputFile string

	if strings.HasPrefix(config.FileName, "s3://") {
		base := filepath.Base(strings.TrimRight(config.FileName, "/"))
		if base == "" || base == "." || base == "/" {
			base = "output.csv"
		}
		outputFile = filepath.Join(outputDir, base)
	} else {
		outputFile = filepath.Join(outputDir, config.FileName)
	}

	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return nil, "", err
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return nil, "", err
	}
	return file, outputFile, nil
}

func joinS3URI(base, key string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid S3 URI %q: %w", base, err)
	}

	u.Path = path.Join(u.Path, key)
	return u.String(), nil
}
