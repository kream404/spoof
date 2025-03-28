package fakers

import (
	"fmt"
	"github.com/kream404/scratch/models"
)

type RangeFaker struct {
	datatype models.Type
	format   string
	values []any
}

func (f *RangeFaker) Generate() (any, error) {
	//TODO: Implement generation logic
	//TODO: Add proper generation logic here
	fmt.Println("spoofed Range:", "")
	return "value", nil
}

func (f *RangeFaker) GetType() models.Type {
	return f.datatype
}

func (f *RangeFaker) GetFormat() string {
	return f.format
}

func NewRangeFaker(format string, values []any) *RangeFaker {
	fmt.Println(values)
	return &RangeFaker{
		datatype: models.Type("Range"),
		format:   "format",
		values: values,
	}
}

func init() {
	RegisterFaker("range", &RangeFaker{
		datatype: models.Type("Range"),
		format:   "",
	})
}
