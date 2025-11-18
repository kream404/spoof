package fakers

import (
	"math/rand"

	"github.com/kream404/spoof/interfaces"
	"github.com/kream404/spoof/models"
)

type CountryCode string

type CountryCodeFaker struct {
	datatype models.Type
	format   string
	rng      *rand.Rand
}

var isoAlpha3 = []CountryCode{
	"GBR", "USA", "CAN", "AUS", "NZL",
	"FRA", "DEU", "ESP", "ITA", "NLD",
	"IRL", "PRT", "SWE", "NOR", "FIN",
	"CHE", "AUT", "BEL", "DNK", "ISL",
	"POL", "CZE", "SVK", "HUN", "ROU",
	"HRV", "SRB", "BGR", "GRC", "TUR",
	"CHN", "JPN", "KOR", "HKG", "SGP",
	"IND", "PAK", "BGD", "IDN", "PHL",
}

func (f *CountryCodeFaker) Generate() (any, error) {
	var r *rand.Rand
	if f.rng != nil {
		r = f.rng
	} else {
		r = rand.New(rand.NewSource(rand.Int63()))
	}

	value := isoAlpha3[r.Intn(len(isoAlpha3))]
	return value, nil
}

func (f *CountryCodeFaker) GetType() models.Type {
	return f.datatype
}

func (f *CountryCodeFaker) GetFormat() string {
	return f.format
}

func NewCountryCodeFaker(format string, rng *rand.Rand) *CountryCodeFaker {
	return &CountryCodeFaker{
		datatype: models.Type("CountryCode"),
		format:   format,
		rng:      rng,
	}
}

func init() {
	RegisterFaker("countrycode", func(field models.Field, rng *rand.Rand) (interfaces.Faker[any], error) {
		return NewCountryCodeFaker(field.Format, rng), nil
	})
}
