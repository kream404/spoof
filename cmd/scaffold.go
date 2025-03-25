package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

type FakerConfig struct {
	Name       string // Faker name (e.g., "UUIDFaker", "EmailFaker")
	DataType   string // Underlying type (e.g., "uuid.UUID", "string")
	Format     string // Format information
}

const fakerTemplate = `package fakers

import (
	"fmt"
	"github.com/kream404/scratch/models"
)

type {{.Name}}Faker struct {
	datatype models.Type
	format   string
}

func (f *{{.Name}}Faker) Generate() ({{.DataType}}, error) {
	//TODO: Implement generation logic
	var value {{.DataType}}
	//TODO: Add proper generation logic here
	fmt.Println("spoofed {{.Name}}:", value)
	return value, nil
}

func (f *{{.Name}}Faker) GetType() models.Type {
	return f.datatype
}

func (f *{{.Name}}Faker) GetFormat() string {
	return f.format
}

func New{{.Name}}Faker() *{{.Name}}Faker {
	return &{{.Name}}Faker{
		datatype: models.Type("{{.DataType}}"),
		format:   "{{.Format}}",
	}
}

func init() {
	RegisterFaker("{{.Name}}Faker", &{{.Name}}Faker{
		datatype: models.Type("{{.Name}}"),
		format:   "",
	})
}
`

func GenerateFaker(config FakerConfig) error {
	fileName := fmt.Sprintf("fakers/%s.go", config.Name)
	fileName = strings.ToLower(fileName)

	_, err := os.Stat(fileName)
	if(err == nil){
		return fmt.Errorf("file %s already exists", fileName)
	}

	f, err := os.Create(fileName)
	if(err != nil){
		return err
	}
	defer f.Close()

	tmpl, err := template.New("faker").Parse(fakerTemplate)
	if(err != nil){
		return err
	}

	return tmpl.Execute(f, config)
}
