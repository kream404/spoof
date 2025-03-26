package models
type Type string

// Enum values
const (
	String    Type = "String"
	Timestamp Type = "Timestamp"
	Int       Type = "Int"
	UUID      Type = "UUID"
	Email      Type = "Email"


)

func GetType(t Type) Type {
	return t
}
