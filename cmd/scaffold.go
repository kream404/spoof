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
	"github.com/kream404/spoof/models"
	"math/rand"
)

type {{.Name}}Faker struct {
	datatype models.Type
	format   string
	rng 		 *rand.Rand
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

func New{{.Name}}Faker(format string, rng *rand.Rand) *{{.Name}}Faker {
	return &{{.Name}}Faker{
		datatype: models.Type("{{.DataType}}"),
		format:   "format",
		rng: rng,
	}
}

func init() {
	RegisterFaker("{{.Name | toLower}}", &{{.Name}}Faker{
		datatype: models.Type("{{.Name}}"),
		format:   "",
	})
}
`

func GenerateFaker(config FakerConfig) error {
	fileName := fmt.Sprintf("fakers/%s.go", config.Name)
	fileName = strings.ToLower(fileName)

	funcMap := template.FuncMap{
			"toLower": strings.ToLower,
	}

	_, err := os.Stat(fileName)
	if(err == nil){
		return fmt.Errorf("file %s already exists", fileName)
	}

	f, err := os.Create(fileName)
	if(err != nil){
		return err
	}
	defer f.Close()

	tmpl, err := template.New("faker").Funcs(funcMap).Parse(fakerTemplate)
	if(err != nil){
		return err
	}

	return tmpl.Execute(f, config)
}
