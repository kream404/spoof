package fakers

import (
	"fmt"
	"github.com/kream404/scratch/models"
	"math/rand"
	"time"
)

var domains = []string{"gmail.com", "outlook.com", "example.com"}

type EmailFaker struct {
	datatype models.Type
	format string
}

func (f *EmailFaker) Generate() (string, error) {
	email, err := NewEmail()
	if err != nil {
		return email, fmt.Errorf("failed to generate Email: %w", err)
	}
	fmt.Println("spoofed: ", email)
	return email, nil
}

func (f *EmailFaker) GetType() models.Type {
	return f.datatype
}

func (f *EmailFaker) GetFormat() string {
	return f.format
}


func NewEmail() (string, error) {
	rand.Seed(time.Now().UnixNano())
	name := RandomString(8)
	domain := domains[rand.Intn(len(domains))]
	return fmt.Sprintf("%s@%s", name, domain), nil
}

func NewEmailFaker() *EmailFaker {
	return &EmailFaker{
		datatype: models.Type("Email"),
		format: "test",
	}
}

//provide seed func: can provide seed file through config to allow more
//representative data
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func init() {
	RegisterFaker("email", &EmailFaker{
		datatype: models.Type("Email"),
		format:   "",
	})
}
