package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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
Pkgs=['github.com/username/module/models']
Skip=[]

[Output]
OutDir='.'
ServiceName='services'
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
	pgtypesPath          string
	tsTypesPath          string
	enumsPkg             string // Alternative pkgs for enums, seperated by commas

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
	flag.StringVar(&pgtypesPath, "pgtypes", "", "File path to write the sql for the postgres enums. If empty, no sql is written")
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
		log.Fatalln(err)
	}

	// Generate code for enumerated constants
	if generateEnums {
		packageNames := cfg.Models.Pkgs
		if enumsPkg != "" {
			packageNames = strings.Split(enumsPkg, ",")
		}

		sql, err := enums.GenerateEnums(packageNames)
		if err != nil {
			log.Fatalln(err)
		}

		if pgtypesPath != "" {
			// Create intermediate dirs if not exists
			dirname := filepath.Dir(pgtypesPath)
			if err := os.MkdirAll(dirname, 0755); err != nil {
				log.Fatalf("error creating directory: %s: %v\n", dirname, err)
			}
			os.WriteFile(pgtypesPath, []byte(sql), 06400)
		}
	}

	// Generate only services and exit
	if generateOnlyServices {
		metadata := parser.Parse(cfg.Models.Pkgs)
		err := parser.GenerateGORMServices(cfg, metadata)
		if err != nil {
			log.Fatalln(err)
		}
		os.Exit(0)
	}

	// Otherwise generate only
	if err := parser.GenerateCode(cfg); err != nil {
		log.Fatalln(err)
	}

	// If tsTypesPath is not empty generate the types
	if tsTypesPath != "" {
		f, err := os.OpenFile(tsTypesPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("error creating file: %s - %v\n", tsTypesPath, err)
		}

		meta := parser.Parse(cfg.Models.Pkgs)
		mapMeta := parser.Map(meta)
		typescript.GenerateTypescriptInterfaces(f, mapMeta, cfg.Overrides)
	}
}
