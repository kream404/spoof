package main

import (
    "github.com/kream404/spoof/cmd"
)

func main() {
	//usage: go run main.go --verbose --scaffold --scaffold_name Phone
 	//usage: go run main.go --config ./test/schema.json --verbose
 	//timestamp: go uses reference, Mon Jan 2 15:04:05 MST 2006
 	cmd.Execute()
}
