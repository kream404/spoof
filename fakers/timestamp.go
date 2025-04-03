package fakers

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kream404/spoof/models"
)

type TimestampFaker struct {
	datatype models.Type
	format   string
	rng 		 *rand.Rand
}

func (f *TimestampFaker) Generate() (string, error) {
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
	RegisterFaker("timestamp", &TimestampFaker{})
}
