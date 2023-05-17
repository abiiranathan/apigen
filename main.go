package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/abiiranathan/apigen/config"
	"github.com/abiiranathan/apigen/enums"
	"github.com/abiiranathan/apigen/parser"
	"github.com/abiiranathan/apigen/typescript"
)

var defaultConfigFileName = "apigen.toml"

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

var (
	shouldInit           bool
	generateOnlyServices bool
	configFileName       string
	generateEnums        bool

	tsTypesPath string

	// Alternative pkg for enums (if different from models pkg specified in config file.)
	enumsPkg string
)

func Usage() {
	fmt.Fprintf(os.Stderr, "apigen - The go API generator. See https://github.com/abiiranathan/apigen\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "Options:")
	flag.VisitAll(func(f *flag.Flag) {
		format := "  -%-14s %s (Default: %v)\n"
		if f.DefValue == "" {
			f.DefValue = "\"\""
		}

		fmt.Fprintf(os.Stderr, format, f.Name, f.Usage, f.DefValue)
	})
}

func init() {
	flag.StringVar(&configFileName, "config", defaultConfigFileName, "Path to config filename.")
	flag.BoolVar(&shouldInit, "init", false, "Initialize project and generate apigen.toml")
	flag.BoolVar(&generateOnlyServices, "services", false, "Generate only services.")
	flag.BoolVar(&generateEnums, "enums", true, "Generate enums code present in the package.")
	flag.StringVar(&enumsPkg, "enums-pkg", "", "Alternative pkg for enums.")
	flag.StringVar(&tsTypesPath, "typescript", "", "Generate typescript types in this file.")
}

func main() {
	flag.Usage = Usage
	flag.Parse()

	// Initialize project and exit
	if shouldInit {
		initConfigFile()
		os.Exit(0)
	}

	// Load configuration file
	cfg, err := config.LoadConfig(configFileName)
	if err != nil {
		log.Panicln(err)
	}

	// Generate code for enumerated constants
	if generateEnums {
		packageName := cfg.Models.Pkg
		if enumsPkg != "" {
			packageName = enumsPkg
		}
		err = enums.GenerateEnums(packageName)
		if err != nil {
			log.Panicln(err)
		}
	}

	// Generate only services and exit
	if generateOnlyServices {
		metadata := parser.Parse(cfg.Models.Pkg)
		err := parser.GenerateGORMServices(cfg, metadata)
		if err != nil {
			log.Panicln(err)
		}
		os.Exit(0)
	}

	// Otherwise generate only
	if err := parser.GenerateCode(cfg); err != nil {
		log.Panicln(err)
	}

	// If tsTypesPath is not empty generate the types
	if tsTypesPath != "" {
		f, err := os.OpenFile(tsTypesPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("error creating file: %s - %v\n", tsTypesPath, err)
		}
		meta := parser.Parse(cfg.Models.Pkg)
		mapMeta := parser.Map(meta)
		typescript.GenerateTypescriptInterfaces(f, mapMeta)
	}
}
