package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/abiiranathan/apigen/config"
	"github.com/abiiranathan/apigen/parser"
	"github.com/abiiranathan/apigen/rawgen"
)

func usage() {
	fmt.Fprintf(os.Stderr, `rawgen - Generate raw PostgreSQL Go code for a model

Usage:
  rawgen [flags]

Flags:
  -config string        Path to apigen.toml (default "apigen.toml")
  -all                  Generate queries for all models in separate files
  -out string           Output directory when using -all (default "queries")
  -model string         Model struct name (required unless -dbtx or -all is set, e.g. "User")
  -table string         Override table name (default: snake_case plural of model)
  -select string        Comma-separated field names to include (default: all)
  -omit string          Comma-separated field names to exclude
  -rows int             Estimated result-row count; preallocates the Query<Model>s
                        results slice with this capacity (default: no preallocation)
  -filter string        Custom filters as "Column:Op:GoType" (repeatable, comma-separated)
                        e.g. "Age:>:int,Name:ILIKE:string"
  -nullable-filter      Same as -filter but wraps in nil check (pointer param)
                        e.g. "Age:>:int" becomes func param "age *int"
  -dbtx                 Write the DBTX interface instead of a model's queries.
                        DBTX is package-level, not per-model: generate it once
                        into the same output package as your Query<Model> files.

Examples:
  # Generate queries for ALL models into separate files inside queries/
  rawgen -all -out queries

  # Write the DBTX interface once per output package
  rawgen -dbtx > queries/dbtx.go && gofmt -w queries/dbtx.go

  # All fields, no filters
  rawgen -model User > queries/user.go && gofmt -w queries/user.go

  # Select specific fields
  rawgen -model User -select "ID,Name,Age"

  # Omit fields
  rawgen -model User -omit "Discount"

  # Preallocate the results slice for Query<Model>s (e.g. typical page size)
  rawgen -model User -rows 50

  # With filters
  rawgen -model User -filter "Age:>:int,Name:ILIKE:string"

  # Nullable filters (applied only when non-nil)
  rawgen -model User -nullable-filter "Age:>:int"
`)
	os.Exit(1)
}

func main() {
	args := parseArgs(os.Args[1:])

	if args.dbtx {
		if err := rawgen.WriteDBTX(os.Stdout); err != nil {
			log.Fatalf("error: %v\n", err)
		}
		return
	}

	if !args.all && args.model == "" {
		fmt.Fprintln(os.Stderr, "error: -model is required (or pass -all for batch generation, or -dbtx to write the DBTX interface)")
		usage()
	}

	cfg, err := config.LoadConfig(args.configPath)
	if err != nil {
		log.Fatalf("error loading config: %v\n", err)
	}

	meta, err := parser.Parse(cfg.Models.Pkgs)
	if err != nil {
		log.Fatalf("parse error: %v\n", err)
	}

	// Mode 1: Batch generate all models in separate files
	if args.all {
		opts := rawgen.PackageOptions{
			OutputDir: args.outDir,
			Skip:      cfg.Models.Skip,
		}
		if err := rawgen.GeneratePackage(meta, opts); err != nil {
			log.Fatalf("batch generation error: %v\n", err)
		}
		return
	}

	// Mode 2: Single model generation
	modelPkg := ""
	for _, m := range meta {
		if m.Name == args.model {
			modelPkg = m.Package
			break
		}
	}
	if modelPkg == "" {
		log.Fatalf("model %q not found in configured packages\n", args.model)
	}

	opts := rawgen.Options{
		ModelName:     args.model,
		ModelPkg:      modelPkg,
		TableName:     args.table,
		SelectFields:  args.selectFields,
		OmitFields:    args.omitFields,
		Filters:       args.filters,
		EstimatedRows: args.estimatedRows,
	}

	if err := rawgen.Generate(os.Stdout, meta, opts); err != nil {
		log.Fatalf("error: %v\n", err)
	}
}

type cliArgs struct {
	configPath    string
	all           bool   // True when -all was passed: generate all models in separate files.
	outDir        string // Target directory when -all is passed (default: "queries").
	model         string
	table         string
	selectFields  []string
	omitFields    []string
	filters       []rawgen.Filter
	estimatedRows int
	dbtx          bool // True when -dbtx was passed: write the DBTX interface only.
}

func parseArgs(args []string) cliArgs {
	c := cliArgs{
		configPath: "apigen.toml",
		outDir:     "queries",
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-config":
			i++
			if i < len(args) {
				c.configPath = args[i]
			}
		case "-all":
			c.all = true
		case "-out":
			i++
			if i < len(args) {
				c.outDir = args[i]
			}
		case "-model":
			i++
			if i < len(args) {
				c.model = args[i]
			}
		case "-table":
			i++
			if i < len(args) {
				c.table = args[i]
			}
		case "-select":
			i++
			if i < len(args) {
				c.selectFields = splitCSV(args[i])
			}
		case "-omit":
			i++
			if i < len(args) {
				c.omitFields = splitCSV(args[i])
			}
		case "-rows":
			i++
			if i < len(args) {
				c.estimatedRows = parseEstimatedRows(args[i])
			}
		case "-filter":
			i++
			if i < len(args) {
				c.filters = append(c.filters, parseFilters(args[i], false)...)
			}
		case "-nullable-filter":
			i++
			if i < len(args) {
				c.filters = append(c.filters, parseFilters(args[i], true)...)
			}
		case "-dbtx":
			c.dbtx = true
		case "-h", "-help", "--help":
			usage()
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[i])
			usage()
		}
	}
	return c
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseFilters(s string, nullable bool) []rawgen.Filter {
	parts := splitCSV(s)
	filters := make([]rawgen.Filter, 0, len(parts))
	for _, p := range parts {
		segs := strings.SplitN(p, ":", 3)
		if len(segs) != 3 {
			log.Fatalf("invalid filter format %q (expected Column:Op:GoType)\n", p)
		}
		filters = append(filters, rawgen.Filter{
			Column:   strings.TrimSpace(segs[0]),
			Op:       strings.TrimSpace(segs[1]),
			GoType:   strings.TrimSpace(segs[2]),
			Nullable: nullable,
		})
	}
	return filters
}

func parseEstimatedRows(s string) int {
	n := 0
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		log.Fatalf("invalid -rows value %q: expected an integer\n", s)
	}
	if n < 0 {
		log.Fatalf("invalid -rows value %q: must not be negative\n", s)
	}
	return n
}
