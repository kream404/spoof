package database

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/json"
	log "github.com/kream404/spoof/services/logger"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "user"
	password = "password"
	dbname   = "database"
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
		result, err = db.FetchRows(sql)
	} else {
		log.Debug("Cache statement ", "sql", config.Statement)
		result, err = db.FetchRows(config.Statement)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rows: %w", err)
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

	fmt.Println(json.ToJSONString(results))
	return results, nil
}
