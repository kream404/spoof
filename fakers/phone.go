package fakers

import (
	"fmt"
	"github.com/kream404/scratch/models"
)

type PhoneFaker struct {
	datatype models.Type
	format   string
}

//TODO: Add proper generation logic here. You may need to set-up a type
func (f *PhoneFaker) Generate() (string, error) {
	//TODO: Implement generation logic
	value := "908230912839083"
	fmt.Println("spoofed Phone:", value)
	return value, nil
}

func (f *PhoneFaker) GetType() models.Type {
	return f.datatype
}

func (f *PhoneFaker) GetFormat() string {
	return f.format
}

func NewPhoneFaker() *PhoneFaker {
	return &PhoneFaker{
		datatype: models.Type("Phone"),
		format:   "",
	}
}
