package interfaces

import(
	"github.com/kream404/scratch/models"
)
var format string
var datatype models.Type

type Faker[T any] interface {
	Generate() (T, error)
	GetType() models.Type
	GetFormat() string
}

func GetType() models.Type{
	return datatype
}

func GetFormat() string{
	return "this is a format"
}
