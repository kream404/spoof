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
	"time"

	"github.com/briandowns/spinner"

	"github.com/google/uuid"
	"github.com/kream404/spoof/fakers"
	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/database"
	log "github.com/kream404/spoof/services/logger"
	s3c "github.com/kream404/spoof/services/s3"
	"github.com/shopspring/decimal"
)

func ProcessFiles(config models.FileConfig, force bool) error {
	ctx := context.Background()
	for _, file := range config.Files {
		if file.Config.FileCount <= 0 {
			file.Config.FileCount = 1
		}

		for i := 0; i < file.Config.FileCount; i++ {
			iterFile := file
			iterFile.Config.FileName = withIndexSuffix(file.Config.FileName, i, file.Config.FileCount)

			if err := processOneFile(ctx, iterFile, "output", force); err != nil {
				log.Error("file processing failed", "file", iterFile.Config.FileName, "err", err)
				return err
			}
		}
	}
	return nil
}

func processOneFile(ctx context.Context, file models.Entity, outDir string, force bool) error {
	if err := validateEntityConfig(file); err != nil {
		return fmt.Errorf("%w", err)
	}

	if strings.EqualFold(file.Postprocess.Operation, "delete") && strings.EqualFold(file.Postprocess.Location, "database") && file.Postprocess.Enabled {

		if !force {
			log.Warn("You are attempting to delete rows",
				"file", file.Config.FileName,
				"host", file.CacheConfig.Hostname,
				"table", file.Postprocess.Table,
				"schema", file.Postprocess.Schema,
			)
			log.Warn("To carry out this action pass `--force` flag")
			return nil
		}

		rows, err := Delete(ctx, file)
		if err != nil {
			return fmt.Errorf("failed to delete rows for file %q: %w", file.Config.FileName, err)
		}
		log.Info("Delete completed", "table", file.Postprocess.Table, "rows", rows)
		return nil
	}

	localPath, err := generateCSV(file, outDir)
	if err != nil {
		return fmt.Errorf("failed to generate CSV for %q: %w", file.Config.FileName, err)
	}

	if err := Insert(ctx, file, localPath); err != nil {
		return fmt.Errorf("database insert failed for %q: %w", file.Config.FileName, err)
	}

	if err := UploadToS3(ctx, file, localPath); err != nil {
		return fmt.Errorf("S3 upload failed for %q: %w", file.Config.FileName, err)
	}

	return nil
}

func Delete(ctx context.Context, file models.Entity) (int, error) {
	pp := file.Postprocess
	if !pp.Enabled || !strings.EqualFold(pp.Location, "database") || !strings.EqualFold(pp.Operation, "delete") {
		return 0, nil
	}

	csvPath := file.Config.FileName

	log.Info("delete from", "schema", pp.Schema, "table", pp.Table, "csv", csvPath)

	db, err := database.NewDBConnector().OpenConnection(*file.CacheConfig)
	if err != nil {
		return 0, fmt.Errorf("open db for delete: %w", err)
	}

	rows, err := db.DeleteRowsByKeyFromFile(csvPath, file)
	if err != nil {
		return 0, fmt.Errorf("failed to delete rows: %w", err)
	}
	return rows, nil
}

func generateCSV(file models.Entity, outDir string) (string, error) {
	var cache []map[string]any
	var cacheIndex, rowIndex = 0, 1

	if file.CacheConfig != nil && (file.CacheConfig.Source != "" || file.CacheConfig.Statement != "") {
		if strings.Contains(file.CacheConfig.Source, ".csv") {
			log.Debug("Loading CSV cache", "source", file.CacheConfig.Source)
			cache, _, _, _ = ReadCSVAsMap(file.CacheConfig.Source)
		} else {
			var err error
			cache, err = database.NewDBConnector().LoadCache(*file.CacheConfig)
			if err != nil {
				return "", fmt.Errorf("load cache: %w", err)
			}
		}
	}

	outFile, localPath, err := makeOutputFile(outDir, file.Config.FileName)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	tempWriter := &strings.Builder{}
	writer := csv.NewWriter(tempWriter)
	if d := file.Config.Delimiter; len(d) > 0 && d[0] != 0 {
		writer.Comma = rune(d[0])
	}

	// headers
	if file.Config.IncludeHeaders {
		var headers []string
		for _, field := range file.Fields {
			headers = append(headers, field.Name)
		}
		if err := writer.Write(headers); err != nil {
			return "", fmt.Errorf("CSV Write Error (headers): %v", err)
		}
	}

	fieldCaches := preloadFieldSources(file.Fields)
	rng, seed := CreateRNGSeed(file.Config.Seed)

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = fmt.Sprintf(" Generating %s (%d rows)...", file.Config.FileName, file.Config.RowCount)
	s.Start()

	for i := 0; i < file.Config.RowCount; i++ {
		row, err := GenerateValues(file, cache, fieldCaches, rowIndex, cacheIndex, rng)
		if err != nil {
			s.Stop()
			return "", fmt.Errorf("generate row: %w", err)
		}
		if err := writer.Write(row); err != nil {
			s.Stop()
			return "", fmt.Errorf("CSV write row: %w", err)
		}

		cacheIndex++
		rowIndex++
		if len(cache) > 0 && cacheIndex >= len(cache) {
			cacheIndex = 0
		}

		if i%500 == 0 {
			s.Suffix = fmt.Sprintf(" Generating %s... (%d/%d)", file.Config.FileName, i, file.Config.RowCount)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		s.Stop()
		return "", fmt.Errorf("CSV flush: %w", err)
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
		s.Stop()
		return "", fmt.Errorf("flush final writer: %w", err)
	}

	s.Stop()
	log.Info("CSV generated", "path", localPath, "seed", seed)
	return localPath, nil
}

func Insert(ctx context.Context, file models.Entity, localPath string) error {
	pp := file.Postprocess
	if !pp.Enabled {
		return nil
	}
	if !strings.EqualFold(pp.Location, "database") {
		return nil
	}
	if !strings.EqualFold(pp.Operation, "insert") {
		return nil
	}

	db, err := database.NewDBConnector().OpenConnection(*file.CacheConfig)
	if err != nil {
		return fmt.Errorf("open db for insert: %w", err)
	}

	rows, ierr := db.InsertRows(localPath, file)
	log.Debug("rows inserted", "value", rows)
	if ierr != nil {
		return fmt.Errorf("failed to insert rows: %w", ierr)
	}
	return nil
}

func UploadToS3(ctx context.Context, file models.Entity, localPath string) error {
	pp := file.Postprocess
	if !pp.Enabled {
		return nil
	}
	if !strings.HasPrefix(pp.Location, "s3://") {
		return nil
	}

	s3, err := s3c.NewS3Connector().OpenConnection(models.CacheConfig{}, pp.Region)
	if err != nil {
		return fmt.Errorf("init S3 connector: %w", err)
	}

	dest, err := joinS3URI(pp.Location, file.Config.FileName)
	if err != nil {
		return err
	}

	if _, err := s3.UploadFile(ctx, dest, localPath); err != nil {
		return fmt.Errorf("upload to S3: %w", err)
	}

	log.Info("Uploaded CSV to S3", "uri", dest)
	return nil
}

func makeOutputFile(baseDir, fileName string) (*os.File, string, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "output"
	}

	var outputFile string
	if strings.HasPrefix(fileName, "s3://") {
		base := filepath.Base(strings.TrimRight(fileName, "/"))
		if base == "" || base == "." || base == "/" {
			base = "output.csv"
		}
		outputFile = filepath.Join(baseDir, base)
	} else {
		outputFile = filepath.Join(baseDir, fileName)
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), os.ModePerm); err != nil {
		return nil, "", err
	}
	file, err := os.Create(outputFile)
	if err != nil {
		return nil, "", err
	}
	return file, outputFile, nil
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

func CreateRNGSeed(seed_in string) (*rand.Rand, string) {
	var s string
	var seed int64
	if seed_in != "" {
		s = seed_in
	} else {
		s = uuid.NewString()
	}

	seed = stringToSeed(s)
	return rand.New(rand.NewSource(seed)), s
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

func withIndexSuffix(name string, i, count int) string {
	if count <= 1 {
		return name
	}

	ext := filepath.Ext(name) // ".csv"
	base := strings.TrimSuffix(name, ext)

	if ext == "" {
		return fmt.Sprintf("%s_%d", base, i+1)
	}

	return fmt.Sprintf("%s_%d%s", base, i+1, ext)

}

func validateEntityConfig(file models.Entity) error {
	missing := []string{}

	if file.Config.FileName == "" {
		missing = append(missing, "config.fileName")
	}

	if file.Config.Delimiter == "" {
		missing = append(missing, "config.delimiter")
	}

	// Database checks for insert/delete
	if file.Postprocess.Enabled && strings.EqualFold(file.Postprocess.Location, "database") {
		if file.CacheConfig == nil {
			missing = append(missing, "cacheConfig")
		} else {
			if file.CacheConfig.Hostname == "" {
				missing = append(missing, "cacheConfig.hostname")
			}
			if file.CacheConfig.Name == "" {
				missing = append(missing, "cacheConfig.name")
			}
		}

		if file.Postprocess.Table == "" {
			missing = append(missing, "postprocess.table")
		}

		if file.Postprocess.Operation == "" {
			missing = append(missing, "postprocess.operation")
		}

		if file.Postprocess.Schema == "" {
			missing = append(missing, "postprocess.schema")
		}

		if file.Postprocess.Location == "" {
			missing = append(missing, "postprocess.location")
		}

		if strings.EqualFold(file.Postprocess.Operation, "delete") {
			if file.Postprocess.Key == "" {
				missing = append(missing, "postprocess.key")
			}

			if file.Postprocess.Type == "" {
				missing = append(missing, "postprocess.type")
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing config: %s", strings.Join(missing, ", "))
	}

	return nil
}
