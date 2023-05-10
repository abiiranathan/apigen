package main

import (
	"fmt"
	"os"

	"github.com/abiiranathan/apigen/flagparse"
)

var dsn, model string

func main() {
	command := flagparse.NewCommand("A tool for generating sql schema")
	genCmd := command.AddSubCommand("generate", "Generate schema", generate)
	connCmd := command.AddSubCommand("connect", "Connect to the database", connect)

	genCmd.StringVar(&model, "model", "", "Name of the model to run migrations against")
	connCmd.StringVar(&dsn, "dsn", "", "data source name")

	err := command.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(args []string) error {
	fmt.Println("generating schema with arg:", model)
	return nil
}

func connect(args []string) error {
	fmt.Println("connecting to database with dsn:", dsn)
	return nil
}
