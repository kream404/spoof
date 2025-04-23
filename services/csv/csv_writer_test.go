package csv_test

import (
	"math/rand"
	"os"
	"testing"

	"github.com/kream404/spoof/models"
	csvgen "github.com/kream404/spoof/services/csv"
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
	row, err := csvgen.GenerateValues(entity, nil, 0, 0, rng)

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
	row, err := csvgen.GenerateValues(entity, seed, 0, 0, rng)

	assert.NoError(t, err)
	assert.Equal(t, []string{"42"}, row)
}

func TestMakeOutputDir(t *testing.T) {
	cfg := models.Config{FileName: "test_output.csv"}

	file, err := csvgen.MakeOutputDir(cfg)
	assert.NoError(t, err)
	assert.FileExists(t, file.Name())

	file.Close()

	err = os.RemoveAll("output")
	assert.NoError(t, err)
}
