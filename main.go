package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/abiiranathan/apigen/config"
	"github.com/abiiranathan/apigen/enums"
	"github.com/abiiranathan/apigen/parser"
	"github.com/abiiranathan/apigen/typescript"
	"github.com/abiiranathan/goflag"
)

var (
	configName    = "apigen.toml"
	generateEnums = true
	pgtypesPath   = "enums.sql"
	tsTypesPath   = ""
)

func initConfigFile() {
	var configData = []byte(`# apigen configuration file.
# Strings should use single quotes

[Models]
Pkgs=['github.com/username/module/models']
Skip=[]

[Output]
OutDir='.'
ServiceName='services'
`)

	// If config file already exists, print message and return
	if _, err := os.Stat(configName); err == nil {
		fmt.Println(configName, "already initialized")
		return
	}

	// Write default text to config
	err := os.WriteFile(configName, configData, 0644)
	if err != nil {
		log.Fatalf("error creating config file: %v\n", err)
	}
	os.Exit(0)
}

func generateSubcommand() {
	// Load configuration file
	cfg, err := config.LoadConfig(configName)
	if err != nil {
		log.Fatalln(err)
	}

	// Validate flags
	if generateEnums && pgtypesPath == "" {
		log.Fatalln("pgtypes flag is required to generate enums")
	}

	// Generate code for enumerated constants
	if generateEnums {
		// Generate sql for postgres enums
		sql, err := enums.GenerateEnums(cfg.Models.Pkgs)
		if err != nil {
			log.Fatalln(err)
		}

		// Create intermediate dirs if not exists
		dirname := filepath.Dir(pgtypesPath)
		if err := os.MkdirAll(dirname, 0755); err != nil {
			log.Fatalf("error creating directory: %s: %v\n", dirname, err)
		}
		os.WriteFile(pgtypesPath, []byte(sql), 06400)

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

	metadata := parser.Parse(cfg.Models.Pkgs)
	err = parser.GenerateGORMServices(cfg, metadata)
	if err != nil {
		log.Fatalln(err)
	}

	os.Exit(0)
}

func main() {
	// Initialize goflag context
	ctx := goflag.NewContext()

	// Global flags
	ctx.AddFlag(goflag.FlagFilePath, "config", "c", &configName, "Path to config filename.", false)

	// Subcommands
	ctx.AddSubCommand("init", "Initialize project and generate apigen.toml", initConfigFile)
	genSubcmd := ctx.AddSubCommand("generate", "Generate code", generateSubcommand)

	genSubcmd.AddFlag(goflag.FlagBool, "enums", "e", &generateEnums, "Generate enums code present in the package.", false)
	genSubcmd.AddFlag(goflag.FlagString, "pgtypes", "d", &pgtypesPath, "File path to write the sql for the postgres enums. If empty, no sql is written", false)
	genSubcmd.AddFlag(goflag.FlagString, "typescript", "t", &tsTypesPath, "File path to write the typescript types. If empty, no typescript is written", false)

	if len(os.Args) < 2 {
		ctx.PrintUsage(os.Stderr)
		os.Exit(1)
	}

	// Parse flags
	matchingSubcommand, err := ctx.Parse(os.Args)
	if err != nil {
		log.Fatalln(err)
	}

	if matchingSubcommand != nil {
		matchingSubcommand.Handler()
		os.Exit(0)
	}
}
