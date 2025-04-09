package database_test

import (
	"testing"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kream404/spoof/models"
	"github.com/stretchr/testify/assert"
	"github.com/kream404/spoof/services/database"
)

func TestOpenConnection(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Create CacheConfig for test
	cfg := models.CacheConfig{
		Hostname: "localhost",
		Port:     "5432",
		Username: "user",
		Password: "password",
		Name:     "database",
		Statement: "SELECT 1",
	}

	connector := &database.DBConnector{DB: db}
	connector.LoadCache(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, connector)
	assert.NotNil(t, connector.DB)

	assert.NoError(t, mock.ExpectationsWereMet())
}
