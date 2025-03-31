package fakers

import (
	"math"
	"math/rand"
	"strconv"

	"github.com/kream404/scratch/models"
)

type NumberFaker struct {
	datatype models.Type
	format   string
	min 		 float64
	max 		 float64
	rng 		 *rand.Rand
}

func (f *NumberFaker) Generate() (float64, error) {
	rawValue := f.min + f.rng.Float64()*(f.max-f.min)
	decimals, err := strconv.Atoi(f.format);

	if err != nil{
		decimals = 2 //default value of 2 decimals
	}
	return roundToDecimal(rawValue, decimals), nil
}

// roundToDecimal rounds a float to a specified number of decimal places
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
	RegisterFaker("number", &NumberFaker{
		datatype: models.Type("Number"),
		format:   "",
	})
}
