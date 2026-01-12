package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"

	"github.com/abiiranathan/apigen/config"
	"github.com/abiiranathan/apigen/parser"
	"github.com/abiiranathan/apigen/typescript"
	"github.com/abiiranathan/goflag"
)

var (
	configName  = "apigen.toml"
	tsTypesPath = ""
)

//go:embed apigen.toml
var defaultConfig []byte

func defineFlags(cli *goflag.CLI) {
	cli.FilePath("config", "c", &configName, "Path to config filename")
	cli.SubCommand("init", "Initialize project and generate apigen.toml", initConfigFile)
	cli.SubCommand("generate", "Generate code", generateCode).
		String("typescript", "t", &tsTypesPath, "File path to write the typescript types")
}

func main() {
	cli := goflag.New()
	defineFlags(cli)

	if len(os.Args) < 2 {
		cli.PrintUsage(os.Stderr)
		os.Exit(1)
	}

	// Parse flags
	subcmd, err := cli.Parse(os.Args)
	if err != nil {
		log.Fatalln(err)
	}

	// If a subcommand is found, execute it and exit
	if subcmd != nil {
		subcmd.Handler()
		os.Exit(0)
	}
}

func generateCode() {
	// Load configuration file
	cfg, err := config.LoadConfig(configName)
	if err != nil {
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

	metadata := parser.Parse(cfg.Models.Pkgs)
	err = parser.GenerateGORMServices(cfg, metadata)
	if err != nil {
		log.Fatalln(err)
	}

	os.Exit(0)
}

func initConfigFile() {
	// If config file already exists, print message and return
	if _, err := os.Stat(configName); err == nil {
		fmt.Printf("config file already exists: %s\n", configName)
		os.Exit(0)
	}

	// Write default text to config
	err := os.WriteFile(configName, defaultConfig, 0644)
	if err != nil {
		log.Fatalf("error creating config file: %v\n", err)
	}
	os.Exit(0)
}
