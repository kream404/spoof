package fakers

import (
	"math"
	"math/rand"
	"strconv"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

type NumberFaker struct {
	datatype models.Type
	format   string
	min 		 float64
	max 		 float64
	rng 		 *rand.Rand
}

func (f *NumberFaker) Generate() (any, error) {
	rawValue := f.min + f.rng.Float64()*(f.max-f.min)
	decimals, err := strconv.Atoi(f.format);

	if err != nil{
		decimals = 2 //default value of 2 decimals
	}
	return roundToDecimal(rawValue, decimals), nil
}

func roundToDecimal(value float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(value*factor) / factor
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
		format:   "format",
		min: min,
		max: max,
		rng: rng,
	}
}

func init() {
	RegisterFaker("number", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		return NewNumberFaker(field.Format, field.Min, field.Max, rng), nil
	})
}
