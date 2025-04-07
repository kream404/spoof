package fakers

import (
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

type UUIDFaker struct {
	datatype models.Type
	format string
	rng *rand.Rand
}

func (f *UUIDFaker) Generate() (any, error) {
	uuid, err := uuid.NewV7()
	if err != nil {
		return uuid, fmt.Errorf("failed to generate UUID: %w", err)
	}
	//fmt.Println("spoofed: ", uuid)
	return uuid, nil
}

func (f *UUIDFaker) GetType() models.Type {
	return f.datatype
}

func (f *UUIDFaker) GetFormat() string {
	return f.format
}

func NewUUIDFaker(format string, rng *rand.Rand) *UUIDFaker {
	return &UUIDFaker{
		datatype: models.Type("uuid.UUID"),
		format: "Format not supported for UUID",
		rng: rng,
	}
}

func init() {
	RegisterFaker("uuid", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		return NewUUIDFaker(field.Format, rng), nil
	})
}
