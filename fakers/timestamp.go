package fakers

import (
	"math/rand"
	"time"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

type TimestampFaker struct {
	datatype models.Type
	format   string
	interval time.Duration
	rng      *rand.Rand
}

func (f *TimestampFaker) Generate() (any, error) {
	now := time.Now()
	offset := time.Duration(int64(f.interval))
	value := now.Add(offset)

	if f.format != "" {
		return value.Format(f.format), nil
	}
	return value, nil // Return time.Time if no formatting is specified
}

func (f *TimestampFaker) GetType() models.Type {
	return f.datatype
}

func (f *TimestampFaker) GetFormat() string {
	return f.format
}

func NewTimestampFaker(format string, intervalSeconds int64, rng *rand.Rand) *TimestampFaker {
	return &TimestampFaker{
		datatype: models.Type("Timestamp"),
		format:   format,
		interval: time.Duration(intervalSeconds) * time.Second,
		rng:      rng,
	}
}

func init() {
	RegisterFaker("timestamp", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		return NewTimestampFaker(field.Format, field.Interval, rng), nil
	})
}
