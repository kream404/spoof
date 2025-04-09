package database

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kream404/spoof/models"
	"github.com/stretchr/testify/assert"
)

func TestOpenConnection(t *testing.T) {
	connector := NewDBConnector()
	cfg := models.CacheConfig{
		Hostname: "localhost",
		Port:     "5432",
		Username: "user",
		Password: "password",
		Name:     "database",
	}

	db, err := connector.OpenConnection(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assert.NotNil(t, db.db)
}

func TestCloseConnection(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// Expect that Close will be called
	mock.ExpectClose()

	connector := &DBConnector{db: db}
	assert.NotPanics(t, func() {
		connector.CloseConnection()
	})

	// Make sure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}


func TestFetchRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	connector := &DBConnector{db: db}

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "Alice").
		AddRow(2, "Bob")

	mock.ExpectQuery("SELECT id, name FROM users").WillReturnRows(rows)

	results, err := connector.FetchRows("SELECT id, name FROM users")
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	assert.Equal(t, "Alice", results[0]["name"])
	assert.Equal(t, "Bob", results[1]["name"])
	assert.Equal(t, int64(1), results[0]["id"])
	assert.Equal(t, int64(2), results[1]["id"])
}
