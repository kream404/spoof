package models

import "fmt"

type Type string

// Enum values
const (
	String    Type = "String"
	Timestamp Type = "Timestamp"
	Int       Type = "Int"
	UUID      Type = "UUID"

)

func GetType(t Type) Type {
	fmt.Println(t)
	return t
}
