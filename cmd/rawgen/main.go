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
  -config string     Path to apigen.toml (default "apigen.toml")
  -model string      Model struct name (required, e.g. "User")
  -table string      Override table name (default: snake_case plural of model)
  -select string     Comma-separated field names to include (default: all)
  -omit string       Comma-separated field names to exclude
  -filter string     Custom filters as "Column:Op:GoType" (repeatable, comma-separated)
                     e.g. "Age:>:int,Name:ILIKE:string"
  -nullable-filter   Same as -filter but wraps in nil check (pointer param)
                     e.g. "Age:>:int" becomes func param "age *int"

Examples:
  # All fields, no filters
  rawgen -model User

  # Select specific fields
  rawgen -model User -select "ID,Name,Age"

  # Omit fields
  rawgen -model User -omit "Discount"

  # With filters
  rawgen -model User -filter "Age:>:int,Name:ILIKE:string"

  # Nullable filters (applied only when non-nil)
  rawgen -model User -nullable-filter "Age:>:int"

  # Pipe to a file
  rawgen -model User > queries/user.go && gofmt -w queries/user.go
`)
	os.Exit(1)
}

func main() {
	args := parseArgs(os.Args[1:])

	if args.model == "" {
		fmt.Fprintln(os.Stderr, "error: -model is required")
		usage()
	}

	cfg, err := config.LoadConfig(args.configPath)
	if err != nil {
		log.Fatalf("error loading config: %v\n", err)
	}

	meta := parser.Parse(cfg.Models.Pkgs)

	// Find the model's package
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
		ModelName:    args.model,
		ModelPkg:     modelPkg,
		TableName:    args.table,
		SelectFields: args.selectFields,
		OmitFields:   args.omitFields,
		Filters:      args.filters,
	}

	if err := rawgen.Generate(os.Stdout, meta, opts); err != nil {
		log.Fatalf("error: %v\n", err)
	}
}

type cliArgs struct {
	configPath   string
	model        string
	table        string
	selectFields []string
	omitFields   []string
	filters      []rawgen.Filter
}

func parseArgs(args []string) cliArgs {
	c := cliArgs{configPath: "apigen.toml"}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-config":
			i++
			if i < len(args) {
				c.configPath = args[i]
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
		// Format: Column:Op:GoType
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
