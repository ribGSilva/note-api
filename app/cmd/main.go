package main

import (
	"github.com/ribgsilva/note-api/app/cmd/schema"
	"os"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		printOpts()
		return
	}
	switch args[1] {
	case "schema":
		schema.Run(args[2:])
	case "help":
		fallthrough
	default:
		printOpts()
	}
}

func printOpts() {
	println("Person API Commands")
	println("\tschema\t\t\t- Schema migrations")
}
