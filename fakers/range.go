package fakers

import (
	"log"
	"math/rand"
	s "strings"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

//picks value at random from given input array
type RangeFaker struct {
	datatype models.Type
	format   string
	values 	 []any
	rng    	 *rand.Rand
}

func (f *RangeFaker) Generate() (any, error) {
	size := len(f.values)
	if !(size > 0){
		log.Fatal("Must provide input to use Range.")
	}

	return f.values[f.rng.Intn(size)], nil
}

func (f *RangeFaker) GetType() models.Type {
	return f.datatype
}

func (f *RangeFaker) GetFormat() string {
	return f.format
}

//can pass a single value, multiple, string or int to store
func NewRangeFaker(format string, valuesArray string, rng *rand.Rand) *RangeFaker {
	if(valuesArray == ""){
		log.Fatal("You must provide values attribute in schema when using 'range'.")
	}
	var parsedValues []any

	parts := s.Split(valuesArray, ",")
	for _, part := range parts {
		trimmed := s.TrimSpace(part)
		parsedValues = append(parsedValues, trimmed)
	}

	return &RangeFaker{
		datatype: "Range",
		format:   format,
		values:   parsedValues,
		rng: rng,
	}
}

func init() {
	RegisterFaker("range", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		return NewRangeFaker(field.Format, field.Values, rng), nil
	})
}
