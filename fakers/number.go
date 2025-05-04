package fakers

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

type NumberFaker struct {
	datatype models.Type
	format   string
	min      float64
	max      float64
	rng      *rand.Rand
}

func (f *NumberFaker) Generate() (any, error) {
	rawValue := f.min + f.rng.Float64()*(f.max-f.min)
	decimals, _ := strconv.Atoi(f.format)
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, rawValue), nil
}

func (f *NumberFaker) GetType() models.Type {
	return f.datatype
}

func (f *NumberFaker) GetFormat() string {
	return f.format
}

func NewNumberFaker(format string, min float64, max float64, rng *rand.Rand) *NumberFaker {
	return &NumberFaker{
		datatype: models.Type("Number"),
		format:   format,
		min:      min,
		max:      max,
		rng:      rng,
	}
}

func init() {
	RegisterFaker("number", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		return NewNumberFaker(field.Format, field.Min, field.Max, rng), nil
	})
}
