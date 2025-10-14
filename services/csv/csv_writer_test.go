package csv_test

import (
	"math/rand"
	"os"
	"testing"

	"github.com/kream404/spoof/models"
	csvgen "github.com/kream404/spoof/services/csv"

	// "github.com/kream404/spoof/services/json"
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
	row, err := csvgen.GenerateValues(entity, nil, nil, 0, 0, rng)

	assert.NoError(t, err)
	assert.Len(t, row, 1)
	assert.NotEmpty(t, row[0])
}

func TestMakeOutputDir(t *testing.T) {
	cfg := models.Config{FileName: "test_output.csv"}

	file, _, err := csvgen.MakeOutputDir(cfg.FileName)
	assert.NoError(t, err)
	assert.FileExists(t, file.Name())

	file.Close()

	err = os.RemoveAll("output")
	assert.NoError(t, err)
}
