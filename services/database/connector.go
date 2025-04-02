package database

import (
	"database/sql"
	"fmt"

	"github.com/kream404/scratch/services/json"
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
	db *sql.DB
}

func NewDBConnector() *DBConnector {
	return &DBConnector{}
}

func (d *DBConnector) OpenConnection() (*DBConnector, error){
 	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+ "password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
  database, err := sql.Open("postgres", psqlInfo)
  if err != nil {
    panic(err)
  }
  println("connected to db...")
  return &DBConnector{db: database}, nil
}

func (d *DBConnector) CloseConnection(){
  err := d.db.Close()
  if err != nil {
    panic(err)
  }
  println("closed connection.")
}

func (d *DBConnector) FetchRows(query string) ([]map[string]any, error) {
	rows, err := d.db.Query(query)
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

	fmt.Println(json.MapToJSON(results))
	return results, nil
}
