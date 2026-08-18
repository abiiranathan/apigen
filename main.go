package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"sync"

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
	cli := goflag.New("apigen", "A code generator for GORM services and TypeScript types")
	defineFlags(cli)

	if len(os.Args) < 2 {
		cli.PrintUsage(os.Stderr)
		os.Exit(1)
	}

	subcmd, err := cli.Parse(os.Args)
	if err != nil {
		log.Fatalln(err)
	}

	if subcmd != nil {
		if err := subcmd.Handler(nil); err != nil {
			log.Fatalf("error executing subcommand: %v\n", err)
		}
		os.Exit(0)
	}
}

func generateCode(any) error {
	cfg, err := config.LoadConfig(configName)
	if err != nil {
		return fmt.Errorf("error loading config file: %w", err)
	}

	// Single parse pass for all downstream generators
	metadata, err := parser.Parse(cfg.Models.Pkgs)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// Concurrently generate TypeScript definitions and Go services
	if tsTypesPath != "" {
		wg.Go(func() {
			f, err := os.OpenFile(tsTypesPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if err != nil {
				errCh <- fmt.Errorf("error opening typescript types file: %w", err)
				return
			}
			defer f.Close()

			mapMeta := parser.Map(metadata)
			typescript.GenerateTypescriptInterfaces(f, mapMeta, cfg.Overrides)
		})
	}

	wg.Go(func() {
		if err := parser.GenerateGORMServices(cfg, metadata); err != nil {
			errCh <- fmt.Errorf("error generating go services: %w", err)
		}
	})

	wg.Wait()
	close(errCh)

	if err, ok := <-errCh; ok {
		return err
	}
	return nil
}

func initConfigFile(any) error {
	if _, err := os.Stat(configName); err == nil {
		fmt.Printf("config file already exists: %s\n", configName)
		return nil
	}

	if err := os.WriteFile(configName, defaultConfig, 0644); err != nil {
		return fmt.Errorf("error creating config file: %w", err)
	}
	return nil
}
