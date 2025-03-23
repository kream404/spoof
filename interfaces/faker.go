package interfaces

import(
	"github.com/kream404/scratch/models"
)
var format string;
var datatype models.Type;

type Faker[T any] interface {
	Generate() (T, error);
	GetType() models.Type;
}

func getType() models.Type{
	return datatype;
}
