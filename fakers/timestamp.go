package fakers

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

type TimestampFaker struct {
	datatype models.Type
	format   string
	rng 		 *rand.Rand
}

func (f *TimestampFaker) Generate() (any, error) {
	value := time.Now();
	if f.format != "" {
		formattedTime := value.Format(f.format);
		return fmt.Sprint(formattedTime), nil
	}
	//fmt.Println("spoofed Timestamp:", value)
	return fmt.Sprint(value), nil
}

func (f *TimestampFaker) GetType() models.Type {
	return f.datatype
}

func (f *TimestampFaker) GetFormat() string {
	return f.format
}

func NewTimestampFaker(format string, rng *rand.Rand) *TimestampFaker {
	return &TimestampFaker{
		datatype: models.Type("Timestamp"),
		format:   format,
		rng: rng,
	}
}

func init() {
	RegisterFaker("timestamp", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		return NewTimestampFaker(field.Format, rng), nil
	})
}
