package fakers

import (
	"math/rand"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

type PhoneFaker struct {
	datatype models.Type
	format   string
	rng 		 *rand.Rand
}

//TODO: Add proper generation logic here. You may need to set-up a type

//TODO: should I make a basic random string generator that uses regex?? might be useful for other fakers
//it would be a good excue to get the barebones of reading in a formatter working, would be lift and shift for
//the config version
func (f *PhoneFaker) Generate() (any, error) {
	//TODO: Implement generation logic
	value := "908230912839083"
	//fmt.Println("spoofed phone:", value)
	return value, nil
}

func (f *PhoneFaker) GetType() models.Type {
	return f.datatype
}

func (f *PhoneFaker) GetFormat() string {
	return f.format
}

func NewPhoneFaker(format string, rng *rand.Rand) *PhoneFaker {
	return &PhoneFaker{
		datatype: models.Type("Phone"),
		format:   format,
		rng: rng,
	}
}

func init() {
	RegisterFaker("phone", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		return NewPhoneFaker(field.Format, rng), nil
	})
}
