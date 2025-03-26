package main

import (
    "github.com/kream404/scratch/cmd"
)

func main() {
	//usage: go run main.go --verbose --scaffold --scaffold_name Phone
 	//usage: go run main.go --config ./config/config.json --verbose
	cmd.Execute()
}
