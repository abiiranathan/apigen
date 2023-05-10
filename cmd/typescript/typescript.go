package main

import (
	"fmt"
	"log"
	"os"

	"github.com/abiiranathan/apigen/parser"
	"github.com/abiiranathan/apigen/typescript"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s [pkg] [outfile]", os.Args[0])
		os.Exit(1)
	}

	f, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatalf("unable to create file: %v\n", err)
	}
	defer f.Close()

	meta := parser.Parse(os.Args[1])
	mapMeta := parser.Map(meta)
	typescript.GenerateTypescriptInterfaces(f, mapMeta)
}
