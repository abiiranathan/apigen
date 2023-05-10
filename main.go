package main

import (
	"fmt"
	"log"
	"os"

	"github.com/abiiranathan/apigen/config"
	"github.com/abiiranathan/apigen/parser"
)

var configFileName = "apigen.toml"

var configData = []byte(`# apigen configuration file.
# Strings should use single quotes

RootPkg='github.com/username/module'

[Models]
Pkg='github.com/username/module/models'
Skip=[]

[Output]
OutDir='.'
ServiceName='services'
HandlersName='handlers'
`)

func initConfigFile() {
	// If config file already exists, print message and return
	if _, err := os.Stat(configFileName); err == nil {
		fmt.Println(configFileName, "already initialized")
		return
	}

	// Write default text to config
	err := os.WriteFile(configFileName, configData, 0644)
	if err != nil {
		log.Fatalf("error creating config file: %v\n", err)
	}
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			initConfigFile()
			os.Exit(0)
		default:
			configFileName = os.Args[1]
		}
	}

	cfg, err := config.LoadConfig(configFileName)
	if err != nil {
		log.Fatalln(err)
	}

	if err := parser.GenerateCode(cfg); err != nil {
		log.Fatalln(err)
	}

}
