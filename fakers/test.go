package fakers

import (
	"fmt"
	"github.com/kream404/scratch/models"
	"math/rand"
)

type TestFaker struct {
	datatype models.Type
	format   string
	rng 		 *rand.Rand
}

func (f *TestFaker) Generate() (Test, error) {
	//TODO: Implement generation logic
	var value Test
	//TODO: Add proper generation logic here
	fmt.Println("spoofed Test:", value)
	return value, nil
}

func (f *TestFaker) GetType() models.Type {
	return f.datatype
}

func (f *TestFaker) GetFormat() string {
	return f.format
}

func NewTestFaker(format string, rng *rand.Rand) *TestFaker {
	return &TestFaker{
		datatype: models.Type("Test"),
		format:   "format",
		rng: rng,
	}
}

func init() {
	RegisterFaker("test", &TestFaker{
		datatype: models.Type("Test"),
		format:   "",
	})
}
