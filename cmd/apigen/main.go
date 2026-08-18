package main

import (
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/abiiranathan/apigen/config"
	"github.com/abiiranathan/apigen/parser"
	"github.com/abiiranathan/apigen/sqlgen"
	"github.com/abiiranathan/apigen/typescript"
	"github.com/abiiranathan/goflag"
)

var (
	apigenConfigPath = "apigen.toml" // Path to the config file.

	typescriptTypesPath = ""     // Generate typescript types if path not empty.
	generateAll         bool     // Generate quries for all models.
	outDir              string   // Output directory for generated code. (default: queries)
	model               string   // Model to generate code for.
	tableOverride       string   // Override table name for -model
	selectFields        []string // Field override for select queries.
	omitFields          []string // Fields to omit from queries.
	filterFields        []string // Optional WHERE filter clauses
	nullableFilter      []string // Where filters for nullable params
	preloadRows         int      // Number of rows to pre-allocate in slices for queries.
	dbtx                bool     // Generate DBTX interface file.
)

//go:embed apigen.toml
var defaultConfig []byte

func defineFlags(root *goflag.CLI) {
	root.FilePath("config", "c", &apigenConfigPath, "Path to config filename")

	root.SubCommand("init", "Initialize project and generate apigen.toml", initConfigFile)
	root.SubCommand("generate", "Generate code", generateCode).
		String("typescript", "t", &typescriptTypesPath, "File path to write the typescript types")

	sqlCmd := root.SubCommand("sql", "Generate raw sql CRUD queries for sql/database for your models", generateSQL)

	sqlCmd.Bool("all", "a", &generateAll, "Generate queries for all models in seperate files").
		String("out", "o", &outDir, "Output directory when using -all").
		String("model", "m", &model, "Model struct name (required unless -dbtx or -all is set)").
		String("table", "t", &tableOverride, "Override table name. Default is snake_case plural of model").
		StringSlice("select", "s", &selectFields, "Comma-seperated field names to include (default: all)").
		StringSlice("filter", "f", &filterFields, "Custom filters as \"Column:Op:GoType\" (repeatable, comma-separated) e.g. \"Age:>:int,Name:ILIKE:string\"").
		StringSlice("omit", "o", &omitFields, "Comma-separated field names to exclude").
		StringSlice("nullable", "n", &nullableFilter, "Same as -filter but wraps in nil check (pointer param) e.g. \"Age:>:int\" becomes func param \"age *int\"").
		Int("rows", "c", &preloadRows, "Estimated result-row count; preallocates the Query<Model>s").
		Bool("dbtx", "d", &dbtx, "Write the DBTX interface instead of a model's queries. DBTX is package-level, not per-model: generate it once into the same output package as your Query<Model> files")
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
	cfg, err := config.LoadConfig(apigenConfigPath)
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
	if typescriptTypesPath != "" {
		wg.Go(func() {
			f, err := os.OpenFile(typescriptTypesPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
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
	if _, err := os.Stat(apigenConfigPath); err == nil {
		fmt.Printf("config file already exists: %s\n", apigenConfigPath)
		return nil
	}

	if err := os.WriteFile(apigenConfigPath, defaultConfig, 0644); err != nil {
		return fmt.Errorf("error creating config file: %w", err)
	}
	return nil
}

func generateSQL(any) error {
	// Generate interface file only and exit.
	if dbtx {
		if err := sqlgen.WriteDBTX(os.Stdout); err != nil {
			return err
		}
		return nil
	}

	if !generateAll && model == "" {
		return errors.New("error: -model is required (or pass -all for batch generation, or -dbtx to write the DBTX interface)")
	}

	cfg, err := config.LoadConfig(apigenConfigPath)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	meta, err := parser.Parse(cfg.Models.Pkgs)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Mode 1: Batch generate all models in separate files
	if generateAll {
		opts := sqlgen.PackageOptions{
			OutputDir: outDir,
			Skip:      cfg.Models.Skip,
		}

		if err := sqlgen.GeneratePackage(meta, opts); err != nil {
			return fmt.Errorf("batch generation error: %v", err)
		}
		return nil
	}

	// Mode 2: Single model generation
	modelPkg := ""
	for _, m := range meta {
		if m.Name == model {
			modelPkg = m.Package
			break
		}
	}

	if modelPkg == "" {
		return fmt.Errorf("model %v not found in configured packages", model)
	}

	filters := parseFilters(filterFields, false)
	filters = append(filters, parseFilters(nullableFilter, true)...)

	opts := sqlgen.Options{
		ModelName:     model,
		ModelPkg:      modelPkg,
		TableName:     tableOverride,
		SelectFields:  selectFields,
		OmitFields:    omitFields,
		Filters:       filters,
		EstimatedRows: preloadRows,
	}

	return sqlgen.Generate(os.Stdout, meta, opts)
}

func parseFilters(parts []string, nullable bool) []sqlgen.Filter {
	filters := make([]sqlgen.Filter, 0, len(parts))
	for _, p := range parts {
		segs := strings.SplitN(p, ":", 3)
		if len(segs) != 3 {
			log.Fatalf("invalid filter format %q (expected Column:Op:GoType)\n", p)
		}
		filters = append(filters, sqlgen.Filter{
			Column:   strings.TrimSpace(segs[0]),
			Op:       strings.TrimSpace(segs[1]),
			GoType:   strings.TrimSpace(segs[2]),
			Nullable: nullable,
		})
	}
	return filters
}
