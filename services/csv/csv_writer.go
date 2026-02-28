package csv

import (
	"bufio"
	"context"
	"encoding/csv"
	jsonstd "encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/google/uuid"

	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/database"
	"github.com/kream404/spoof/services/evaluator" // ✅ new
	log "github.com/kream404/spoof/services/logger"
	s3c "github.com/kream404/spoof/services/s3"
)

func ProcessFiles(config models.FileConfig, force bool, dryRun bool) error {
	ctx := context.Background()
	acc := &OutputAccumulator{}

	for _, file := range config.Files {
		if file.Config.FileCount <= 0 {
			file.Config.FileCount = 1
		}

		for i := 0; i < file.Config.FileCount; i++ {
			iterFile := file
			iterFile.Config.FileName = withIndexSuffix(file.Config.FileName, i, file.Config.FileCount)

			if err := processOneFile(ctx, iterFile, "output", force, dryRun, acc); err != nil {
				log.Error("file processing failed", "file", iterFile.Config.FileName, "err", err)
				return err
			}
		}

		if err := acc.FlushToStdout(file.Config.FileName); err != nil {
			return err
		}
	}

	return nil
}

func processOneFile(ctx context.Context, file models.Entity, outDir string, force bool, dryRun bool, acc *OutputAccumulator) error {
	if err := validateEntityConfig(file); err != nil {
		return fmt.Errorf("%w", err)
	}

	if (strings.EqualFold(file.Postprocess.Operation, "delete") || strings.EqualFold(file.Postprocess.Operation, "insert")) &&
		strings.EqualFold(file.Postprocess.Location, "database") &&
		file.Postprocess.Enabled {

		if !force {
			log.Warn("You are attempting to perform a destructive database operation",
				"operation", file.Postprocess.Operation,
				"file", file.Config.FileName,
				"host", file.CacheConfig.Hostname,
				"table", file.Postprocess.Table,
				"schema", file.Postprocess.Schema,
			)
			log.Warn("To carry out this action pass the `--force` flag")
			return nil
		}
	}

	var (
		err       error
		localPath string
	)

	if file.Fields != nil {
		localPath, err = generateCSV(file, outDir, acc)
	}

	if err != nil {
		return fmt.Errorf("failed to generate CSV for %q: %w", file.Config.FileName, err)
	}

	if dryRun {
		log.Info("Dry run enabled; skipping post-processing steps", "file", file.Config.FileName)
		return nil
	}

	if err := Delete(ctx, file); err != nil {
		return fmt.Errorf("failed to delete rows for file %q: %w", file.Config.FileName, err)
	}

	if err := Insert(ctx, file, localPath); err != nil {
		return fmt.Errorf("database insert failed for %q: %w", file.Config.FileName, err)
	}

	if err := UploadToS3(ctx, file, localPath); err != nil {
		return fmt.Errorf("S3 upload failed for %q: %w", file.Config.FileName, err)
	}

	return nil
}

func Delete(ctx context.Context, file models.Entity) error {
	pp := file.Postprocess
	if !pp.Enabled || !strings.EqualFold(pp.Location, "database") || !strings.EqualFold(pp.Operation, "delete") {
		return nil
	}

	csvPath := file.Config.FileName
	log.Debug("delete from", "schema", pp.Schema, "table", pp.Table, "csv", csvPath)

	db, err := database.NewDBConnector().OpenConnection(*file.CacheConfig)
	if err != nil {
		return fmt.Errorf("open db for delete: %w", err)
	}

	rows, err := db.DeleteRowsByKeyFromFile(csvPath, file)
	if err != nil {
		return fmt.Errorf("failed to delete rows: %w", err)
	}

	log.Info("Delete completed", "table", pp.Table, "rows", rows)
	return nil
}

func generateCSV(file models.Entity, outDir string, acc *OutputAccumulator) (string, error) {
	var cacheIndex, rowIndex = 0, 1

	log.Info("Generating file", "file", file.Config.FileName)

	cache, err := LoadCache(file.CacheConfig)
	if err != nil {
		return "", fmt.Errorf("could not load cache: %w", err)
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

	if file.Config.IncludeHeaders {
		headers := make([]string, 0, len(file.Fields))
		for _, field := range file.Fields {
			if field.Skip {
				continue
			}
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
		row, generated, err := evaluator.GenerateValues(
			file,
			cache,
			map[string][]map[string]any(fieldCaches),
			rowIndex,
			cacheIndex,
			rng,
		)
		if err != nil {
			s.Stop()
			return "", fmt.Errorf("generate row: %w", err)
		}

		if err := emitOutputHooks(file, generated, acc); err != nil {
			s.Stop()
			return "", fmt.Errorf("output hook: %w", err)
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

func LoadCache(config *models.CacheConfig) ([]map[string]any, error) {
	var (
		cache []map[string]any
		err   error
	)

	if config != nil && (config.Source != "" || config.Statement != "") {
		log.Debug("Loading CSV cache", "source", config.Source)

		switch {
		case strings.HasPrefix(config.Source, "s3://"):
			s3, errConn := s3c.NewS3Connector().OpenConnection(models.CacheConfig{}, config.Region)
			if errConn != nil {
				return nil, fmt.Errorf("open s3 connection: %w", errConn)
			}
			cache, err = s3.LoadCache(*config)
			if err != nil {
				return nil, fmt.Errorf("load cache from s3: %w", err)
			}

		case strings.Contains(config.Source, ".csv") && !strings.HasPrefix(config.Source, "s3://"):
			cache, _, _, err = ReadCSVAsMap(config.Source)
			if err != nil {
				return nil, fmt.Errorf("read local csv cache: %w", err)
			}

		default:
			cache, err = database.NewDBConnector().LoadCache(*config)
			if err != nil {
				return nil, fmt.Errorf("load cache from db: %w", err)
			}
		}
	}

	return cache, nil
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

	f, err := os.Create(outputFile)
	if err != nil {
		return nil, "", err
	}

	return f, outputFile, nil
}

type fieldCache map[string][]map[string]any

func preloadFieldSources(fields []models.Field) fieldCache {
	fc := make(fieldCache)
	seen := make(map[string]struct{})
	preloadFieldSourcesRecursive(fields, fc, seen)
	return fc
}

func preloadFieldSourcesRecursive(fields []models.Field, fc fieldCache, seen map[string]struct{}) {
	for _, f := range fields {
		if f.Source != "" && strings.Contains(f.Source, ".csv") {
			if _, ok := seen[f.Source]; !ok {
				rows, _, _, err := ReadCSVAsMap(f.Source)
				if err != nil {
					log.Error("Failed to preload source CSV; will skip injection for this source",
						"source", f.Source, "err", err)
				} else {
					fc[f.Source] = rows
					seen[f.Source] = struct{}{}
					log.Debug("Preloaded CSV source", "source", f.Source, "rows", len(rows))
				}
			}
		}

		if len(f.Fields) > 0 {
			preloadFieldSourcesRecursive(f.Fields, fc, seen)
		}
	}
}

func stringToSeed(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return int64(h.Sum64())
}

func CreateRNGSeed(seedIn string) (*rand.Rand, string) {
	var s string
	if seedIn != "" {
		s = seedIn
	} else {
		s = uuid.NewString()
	}

	seed := stringToSeed(s)
	return rand.New(rand.NewSource(seed)), s
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

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	if ext == "" {
		return fmt.Sprintf("%s_%d", base, i+1)
	}

	return fmt.Sprintf("%s_%d%s", base, i+1, ext)
}

func validateEntityConfig(file models.Entity) error {
	var missing []string

	if file.Config.FileName == "" {
		missing = append(missing, "config.fileName")
	}

	if file.Config.Delimiter == "" {
		missing = append(missing, "config.delimiter")
	}

	// Database checks for insert/delete
	if file.Postprocess.Enabled && strings.EqualFold(file.Postprocess.Location, "database") {
		if file.CacheConfig == nil {
			return fmt.Errorf(
				"missing database config for %s: you must pass a `profile` or inline `cache` configuration",
				file.Config.FileName,
			)
		}

		if file.CacheConfig.Hostname == "" {
			return fmt.Errorf(
				"missing database config for %s: cacheConfig.hostname is empty (pass a `profile` or cacheConfig)",
				file.Config.FileName,
			)
		}

		if file.CacheConfig.Name == "" {
			missing = append(missing, "cacheConfig.name")
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
		return fmt.Errorf("missing config for %s: %s", file.Config.FileName, strings.Join(missing, ", "))
	}

	return nil
}

func emitOutputHooks(file models.Entity, generated map[string]string, acc *OutputAccumulator) error {
	if acc == nil || len(file.Output) == 0 {
		return nil
	}

	for _, mapping := range file.Output {
		out := make(map[string]any, len(mapping))
		for outKey, sourcePath := range mapping {
			v, ok := resolveOutputPath(generated, sourcePath)
			if !ok {
				out[outKey] = nil
				continue
			}
			out[outKey] = v
		}
		acc.Add(out)
	}

	return nil
}

func resolveOutputPath(generated map[string]string, p string) (any, bool) {
	p = strings.TrimSpace(p)
	if p == "" {
		return nil, false
	}

	parts := strings.Split(p, ".")
	root := parts[0]

	rootVal, ok := generated[root]
	if !ok {
		return nil, false
	}

	if len(parts) == 1 {
		return rootVal, true
	}

	var node any
	if err := jsonstd.Unmarshal([]byte(rootVal), &node); err != nil {
		return nil, false
	}

	cur := node
	for _, tok := range parts[1:] {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			return nil, false
		}

		switch typed := cur.(type) {
		case map[string]any:
			nxt, exists := typed[tok]
			if !exists {
				return nil, false
			}
			cur = nxt

		case []any:
			idx, err := strconv.Atoi(tok)
			if err != nil || idx < 0 || idx >= len(typed) {
				return nil, false
			}
			cur = typed[idx]

		default:
			return nil, false
		}
	}

	return cur, true
}

type OutputAccumulator struct {
	items []map[string]any
}

func (a *OutputAccumulator) Add(item map[string]any) {
	if item == nil {
		return
	}
	a.items = append(a.items, item)
}

func (a *OutputAccumulator) FlushToStdout(fileName string) error {
	if len(a.items) == 0 {
		return nil
	}

	out := map[string]any{
		fileName: a.items,
	}

	b, err := jsonstd.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}

	fmt.Fprintln(os.Stdout, string(b))
	a.items = nil
	return nil
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
