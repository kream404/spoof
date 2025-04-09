package csv_test

import (
	"math/rand"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kream404/spoof/models"
	csvgen "github.com/kream404/spoof/services/csv"
	"github.com/kream404/spoof/services/database"
	"github.com/stretchr/testify/assert"
)

func TestGenerateValues_WithFaker(t *testing.T) {
	field := models.Field{
		Name: "customerid",
		Type: "uuid",
	}

	entity := models.Entity{
		Fields: []models.Field{field},
	}

	rng := rand.New(rand.NewSource(42))
	row, err := csvgen.GenerateValues(entity, nil, 0, rng)

	assert.NoError(t, err)
	assert.Len(t, row, 1)
	assert.NotEmpty(t, row[0])
}

func TestGenerateValues_WithSeed(t *testing.T) {
	entity := models.Entity{
		Fields: []models.Field{
			{Name: "id", SeedType: "db"},
		},
	}

	seed := []map[string]any{
		{"id": 42},
	}

	rng := rand.New(rand.NewSource(42))
	row, err := csvgen.GenerateValues(entity, seed, 0, rng)

	assert.NoError(t, err)
	assert.Equal(t, []string{"42"}, row)
}

func TestMakeOutputDir(t *testing.T) {
	cfg := models.Config{FileName: "test_output.csv"}
	file, err := csvgen.MakeOutputDir(cfg)
	assert.NoError(t, err)
	assert.FileExists(t, file.Name())

	file.Close()
	defer os.Remove(file.Name())
}

func TestLoadCache(t *testing.T) {
	_, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// Mock connector to use sqlmock

	mock.ExpectQuery("SELECT id FROM test_table").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))

	cache := models.CacheConfig{
		Hostname: "localhost",
		Port:     "5432",
		Username: "user",
		Password: "pass",
		Name:     "db",
		Statement: "SELECT id FROM test_table",
	}

	result, err := database.LoadCache(cache)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, int64(1), result[0]["id"])
	assert.Equal(t, int64(2), result[1]["id"])
}
