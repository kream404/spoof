package fakers

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/kream404/scratch/models"
)

type UUIDFaker struct {
	datatype models.Type
	format string
}

func (f *UUIDFaker) Generate() (uuid.UUID, error) {
	uuid, err := uuid.NewUUID()
	if err != nil {
		return uuid, fmt.Errorf("failed to generate UUID: %w", err)
	}
	fmt.Println("spoofed: ", uuid)
	return uuid, nil
}

func (f *UUIDFaker) GetType() models.Type {
	return f.datatype
}

func (f *UUIDFaker) GetFormat() string {
	return f.format
}

func NewUUIDFaker() *UUIDFaker {
	return &UUIDFaker{
		datatype: models.Type("uuid.UUID"),
		format: "Format not supported for UUID",
	}
}
