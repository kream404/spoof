package database

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kream404/spoof/models"
	log "github.com/kream404/spoof/services/logger"

	_ "github.com/lib/pq"
)

type DBConnector struct {
	DB *sql.DB
}

func NewDBConnector() *DBConnector {
	return &DBConnector{}
}

func (d *DBConnector) OpenConnection(config models.CacheConfig) (*DBConnector, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+"password=%s dbname=%s sslmode=disable", config.Hostname, config.Port, config.Username, config.Password, config.Name)
	log.Debug("Connection string ", "string", psqlInfo)
	database, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	return &DBConnector{DB: database}, nil
}

func (d *DBConnector) CloseConnection() {
	err := d.DB.Close()
	if err != nil {
		panic(err)
	}
}

func (d *DBConnector) LoadCache(config models.CacheConfig) ([]map[string]any, error) {
	var result []map[string]any
	db, err := NewDBConnector().OpenConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.CloseConnection()

	log.Debug("Populating cache with file", "path", config.Source)

	if config.Source != "" {
		sql, err := loadSQLFromFile(config.Source)
		if err != nil {
			return nil, err
		}
		result, _ = db.FetchRows(sql)
	} else if config.Statement != "" {
		log.Debug("Cache statement ", "sql", config.Statement)
		result, err = db.FetchRows(config.Statement)

		if err != nil {
			return nil, fmt.Errorf("failed to fetch rows: %w", err)
		}
	}

	return result, nil
}

func loadSQLFromFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read SQL file: %w", err)
	}
	sql := string(content)
	if sql == "" {
		return "", fmt.Errorf("SQL file at %s is empty", path)
	}
	return sql, nil
}

func (d *DBConnector) FetchRows(query string) ([]map[string]any, error) {
	rows, err := d.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))

		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]any)
		for i, col := range columns {
			switch v := values[i].(type) {
			case []byte: // convert []byte to string
				rowMap[col] = string(v)
			default:
				rowMap[col] = v
			}
		}

		results = append(results, rowMap)
	}

	log.Debug("Cache populated from db", "sample_row", fmt.Sprint(results[0]))
	return results, nil
}

func (d *DBConnector) InsertRows(path string, config models.Entity) (int, error) {
	log.Info("Inserting rows from file", "path", path)
	if d == nil || d.DB == nil {
		return 0, errors.New("database connection not initialized")
	}
	if config.Postprocess.Table == "" {
		return 0, errors.New("table name is required")
	}

	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open csv: %w", err)
	}
	defer file.Close()

	r := csv.NewReader(file)
	if config.Config.Delimiter[0] != 0 {
		r.Comma = rune(config.Config.Delimiter[0])
	}

	if !config.Postprocess.HasHeader && len(config.Postprocess.Columns) == 0 {
		return 0, errors.New("either HasHeader must be true or Columns must be provided")
	}

	columns := append([]string(nil), config.Postprocess.Columns...)
	if config.Postprocess.HasHeader {
		header, err := r.Read()
		if err != nil {
			return 0, fmt.Errorf("read header: %w", err)
		}
		if len(columns) == 0 {
			for _, h := range header {
				h = strings.TrimSpace(h)
				if h == "" {
					return 0, errors.New("empty column name in header")
				}
				columns = append(columns, h)
			}
		}
	}

	if len(columns) == 0 {
		return 0, errors.New("no columns specified")
	}
	log.Debug("columns to insert", "columns", columns)
	fullTable := quoteIdent(config.Postprocess.Schema) + "." + quoteIdent(config.Postprocess.Table)

	colList := make([]string, len(columns))
	for i, c := range columns {
		colList[i] = quoteIdent(c)
	}
	insertPrefix := fmt.Sprintf("INSERT INTO %s (%s) VALUES ", fullTable, strings.Join(colList, ", "))
	batchSize := config.Postprocess.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}

	rowsInserted := 0
	rowWidth := len(columns)

	var (
		args          []any
		valueGroups   []string
		batchRowCount int
	)

	flush := func() error {
		if batchRowCount == 0 {
			return nil
		}

		query := insertPrefix + strings.Join(valueGroups, ", ")
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}

		rowsInserted += batchRowCount
		args = args[:0]
		valueGroups = valueGroups[:0]
		batchRowCount = 0
		return nil
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("read csv: %w", err)
		}

		if len(rec) != rowWidth {
			_ = tx.Rollback()
			return 0, fmt.Errorf("csv row has %d columns, expected %d", len(rec), rowWidth)
		}

		rowArgs := make([]any, rowWidth)
		for i := 0; i < rowWidth; i++ {
			val := rec[i]
			if val == "" {
				rowArgs[i] = nil
			} else {
				rowArgs[i] = strings.TrimSpace(val)
			}
		}

		start := len(args) + 1
		placeholders := make([]string, rowWidth)
		for i := 0; i < rowWidth; i++ {
			placeholders[i] = fmt.Sprintf("$%d", start+i)
		}

		valueGroups = append(valueGroups, "("+strings.Join(placeholders, ", ")+")")
		args = append(args, rowArgs...)
		batchRowCount++

		if batchRowCount >= batchSize {
			if err := flush(); err != nil {
				_ = tx.Rollback()
				return 0, fmt.Errorf("insert batch: %w", err)
			}
		}
	}

	if err := flush(); err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("final flush: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return rowsInserted, nil
}

func (d *DBConnector) DeleteRowsByKeyFromFile(path string, config models.Entity) (int, error) {
	log.Info("Deleting rows from file", "path", path)
	if d == nil || d.DB == nil {
		return 0, errors.New("database connection not initialized")
	}
	pp := config.Postprocess
	if pp.Table == "" || pp.Schema == "" {
		return 0, errors.New("schema and table are required")
	}
	if pp.Key == "" || pp.Type == "" {
		return 0, errors.New("postprocess.key and postprocess.type are required")
	}

	file, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open csv: %w", err)
	}
	defer file.Close()

	r := csv.NewReader(file)
	if config.Config.Delimiter != "" && config.Config.Delimiter[0] != 0 {
		r.Comma = rune(config.Config.Delimiter[0])
	}

	// Read header (or use provided columns)
	var columns []string
	if pp.HasHeader {
		header, err := r.Read()
		if err != nil {
			return 0, fmt.Errorf("read header: %w", err)
		}
		for _, h := range header {
			h = strings.TrimSpace(h)
			if h == "" {
				return 0, errors.New("empty column name in header")
			}
			columns = append(columns, h)
		}
	} else if len(config.Postprocess.Columns) > 0 {
		columns = append([]string(nil), config.Postprocess.Columns...)
	} else {
		return 0, errors.New("need header or explicit Columns to locate key")
	}

	keyIdx := -1
	for i, c := range columns {
		if strings.EqualFold(c, pp.Key) {
			keyIdx = i
			break
		}
	}
	if keyIdx < 0 {
		return 0, fmt.Errorf("key column %q not found in CSV", pp.Key)
	}

	fullTable := quoteIdent(pp.Schema) + "." + quoteIdent(pp.Table)
	keyIdent := quoteIdent(pp.Key)
	keyType := strings.TrimSpace(pp.Type)

	batchSize := pp.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tx, err := d.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}

	rowsDeleted := 0
	var (
		args          []any
		valueGroups   []string
		batchRowCount int
	)

	flush := func() error {
		if batchRowCount == 0 {
			return nil
		}
		// DELETE ... USING (VALUES ($1::<type>), ...) AS v(key) WHERE t.key = v.key
		query := fmt.Sprintf(
			`DELETE FROM %s AS t USING (VALUES %s) AS v(%s) WHERE t.%s = v.%s`,
			fullTable,
			strings.Join(valueGroups, ", "),
			keyIdent,
			keyIdent, keyIdent,
		)
		res, err := tx.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
		affected, _ := res.RowsAffected()
		rowsDeleted += int(affected)
		// reset
		args = args[:0]
		valueGroups = valueGroups[:0]
		batchRowCount = 0
		return nil
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("read csv: %w", err)
		}
		if keyIdx >= len(rec) {
			_ = tx.Rollback()
			return 0, fmt.Errorf("row missing key column %q", pp.Key)
		}

		val := strings.TrimSpace(rec[keyIdx])
		if val == "" {
			continue
		}

		// placeholder with explicit cast
		start := len(args) + 1
		ph := fmt.Sprintf("$%d::%s", start, keyType)
		valueGroups = append(valueGroups, "("+ph+")")
		args = append(args, val)
		batchRowCount++

		if batchRowCount >= batchSize {
			if err := flush(); err != nil {
				_ = tx.Rollback()
				return 0, fmt.Errorf("delete batch: %w", err)
			}
		}
	}

	if err := flush(); err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("final flush: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return rowsDeleted, nil
}

func quoteIdent(ident string) string {
	ident = strings.TrimSpace(ident)
	if ident == "" {
		return ""
	}
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}
