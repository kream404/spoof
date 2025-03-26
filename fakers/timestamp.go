package fakers

import (
	"fmt"
	"time"

	"github.com/kream404/scratch/models"
)

type TimestampFaker struct {
	datatype models.Type
	format   string
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

func NewTimestampFaker(format string) *TimestampFaker {
	return &TimestampFaker{
		datatype: models.Type("Timestamp"),
		format:   format,
	}
}

func init() {
	RegisterFaker("timestamp", &TimestampFaker{})
}
