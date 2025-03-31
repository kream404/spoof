package fakers

import (
	"fmt"
	"math/rand"

	"github.com/kream404/scratch/models"
)

var domains = []string{"gmail.com", "outlook.com", "example.com"}

type EmailFaker struct {
	datatype models.Type
	format   string
	seed     *int64 // Optional seed for deterministic generation
	rng      *rand.Rand
}

func (f *EmailFaker) Generate() (string, error) {
	email, err := f.NewEmail()
	if err != nil {
		return email, fmt.Errorf("failed to generate Email: %w", err)
	}
	return email, nil
}

func (f *EmailFaker) GetType() models.Type {
	return f.datatype
}

func (f *EmailFaker) GetFormat() string {
	return f.format
}

// NewEmail generates a deterministic or random email based on the provided seed
func (f *EmailFaker) NewEmail() (string, error) {
	name := f.RandomString(8)
	domain := domains[f.rng.Intn(len(domains))]
	return fmt.Sprintf("%s@%s", name, domain), nil
}

// NewEmailFaker creates an EmailFaker with optional deterministic behavior
func NewEmailFaker(format string, rng *rand.Rand) *EmailFaker {
	return &EmailFaker{
		datatype: models.Type("Email"),
		format:   format,
		rng:      rng, // Store the RNG instance
	}
}


// RandomString generates a random string using the instance's RNG
func (f *EmailFaker) RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[f.rng.Intn(len(charset))]
	}
	return string(result)
}

func init() {
	RegisterFaker("email", NewEmailFaker("", nil))
}
