// Package rawgen generates raw PostgreSQL Go code for a given model struct.
// It produces functions that use database/sql directly (no ORM) for
// Insert, Get, Query, Delete, and Update operations.
package rawgen

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/abiiranathan/apigen/parser"
	"github.com/iancoleman/strcase"
)

// Options controls what code is generated.
type Options struct {
	// ModelName is the struct name to generate for (e.g. "User"). Required.
	ModelName string

	// ModelPkg is the import path of the models package (e.g. "github.com/user/app/models").
	ModelPkg string

	// SelectFields limits the SELECT columns. Empty means all non-relation columns.
	SelectFields []string

	// OmitFields excludes specific columns from all generated queries.
	OmitFields []string

	// Filters are custom WHERE clauses appended to Query (e.g. "age > $N", "name ILIKE $N").
	Filters []Filter

	// TableName overrides the default snake_case plural table name.
	TableName string
}

// Filter represents a custom WHERE clause added to the generated Query function.
type Filter struct {
	// Column is the Go field name (will be snake_cased for SQL).
	Column string
	// Op is the SQL operator (e.g. "=", ">", "ILIKE", "IN").
	Op string
	// GoType is the Go type for the function parameter (e.g. "string", "int").
	GoType string
	// Nullable when true wraps the filter in an if-check so it's only applied when non-nil.
	Nullable bool
}

// column holds resolved column metadata for code generation.
type column struct {
	GoName   string // Field name in Go, e.g. "Name".
	ColName  string // Formatted SQL identifier (clean unless reserved keyword), e.g. "name" or `"order"`.
	GoType   string // Go type as declared on the struct, e.g. "string".
	IsFK     bool   // True for foreign key / relation fields — excluded from raw queries.
	IsPK     bool   // True for the primary key field ("ID").
	BaseType string // Field type with pointer/slice wrappers stripped.
}

type templateData struct {
	ModelName    string
	ModelPkg     string
	ModelPkgName string
	TableName    string
	Ident        string
	Columns      []column
	PKCol        *column
	Filters      []Filter
}

// Generate writes raw PostgreSQL Go code to w for the specified model.
func Generate(w io.Writer, meta []parser.StructMeta, opts Options) error {
	if opts.ModelName == "" {
		return fmt.Errorf("rawgen: Options.ModelName must not be empty")
	}

	target, err := findStruct(meta, opts.ModelName)
	if err != nil {
		return err
	}

	cols := resolveColumns(target, opts)
	if len(cols) == 0 {
		return fmt.Errorf("rawgen: model %q has no columns after applying SelectFields/OmitFields", opts.ModelName)
	}

	tableName := opts.TableName
	if tableName == "" {
		tableName = strcase.ToSnake(opts.ModelName) + "s"
	}

	pkgName := opts.ModelPkg
	if idx := strings.LastIndex(opts.ModelPkg, "/"); idx >= 0 {
		pkgName = opts.ModelPkg[idx+1:]
	}
	if pkgName == "" {
		pkgName = "models"
	}

	data := templateData{
		ModelName:    opts.ModelName,
		ModelPkg:     opts.ModelPkg,
		ModelPkgName: pkgName,
		TableName:    formatIdent(tableName),
		Ident:        receiverIdent(opts.ModelName),
		Columns:      cols,
		PKCol:        findPK(cols),
		Filters:      opts.Filters,
	}

	return writeCode(w, data)
}

func findStruct(meta []parser.StructMeta, name string) (*parser.StructMeta, error) {
	for i := range meta {
		if meta[i].Name == name {
			return &meta[i], nil
		}
	}
	return nil, fmt.Errorf("rawgen: model %q not found in parsed structs", name)
}

func receiverIdent(name string) string {
	if name == "" {
		return "m"
	}
	r, _ := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError || !(unicode.IsLetter(r) || r == '_') {
		return "m"
	}
	return strings.ToLower(string(r))
}

// postgresReserved contains standard PostgreSQL reserved keywords that require quotes.
var postgresReserved = map[string]bool{
	"all": true, "analyse": true, "analyze": true, "and": true, "any": true, "array": true,
	"as": true, "asc": true, "asymmetric": true, "both": true, "case": true, "cast": true,
	"check": true, "collate": true, "collation": true, "column": true, "constraint": true,
	"create": true, "cross": true, "current_catalog": true, "current_date": true,
	"current_role": true, "current_schema": true, "current_time": true,
	"current_timestamp": true, "current_user": true, "default": true, "deferrable": true,
	"desc": true, "distinct": true, "do": true, "else": true, "end": true, "except": true,
	"false": true, "fetch": true, "for": true, "foreign": true, "from": true, "grant": true,
	"group": true, "having": true, "in": true, "initially": true, "intersect": true,
	"into": true, "lateral": true, "leading": true, "limit": true, "localtime": true,
	"localtimestamp": true, "not": true, "null": true, "offset": true, "on": true,
	"only": true, "or": true, "order": true, "placing": true, "primary": true,
	"references": true, "returning": true, "select": true, "session_user": true,
	"some": true, "symmetric": true, "table": true, "then": true, "to": true,
	"trailing": true, "true": true, "union": true, "unique": true, "user": true,
	"using": true, "variadic": true, "when": true, "where": true, "window": true, "with": true,
}

// formatIdent quotes an identifier ONLY if it is a PostgreSQL reserved keyword
// or contains characters outside [a-z0-9_].
func formatIdent(ident string) string {
	lower := strings.ToLower(ident)
	if postgresReserved[lower] {
		return `"` + ident + `"`
	}

	for i, r := range lower {
		if (i == 0 && !unicode.IsLetter(r) && r != '_') || (!unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_') {
			return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
		}
	}
	return lower
}

func resolveColumns(st *parser.StructMeta, opts Options) []column {
	cols := make([]column, 0, len(st.Fields))
	for _, f := range st.Fields {
		if f.Preload {
			continue
		}
		if len(opts.SelectFields) > 0 && !slices.Contains(opts.SelectFields, f.Name) {
			continue
		}
		if slices.Contains(opts.OmitFields, f.Name) {
			continue
		}
		cols = append(cols, column{
			GoName:   f.Name,
			ColName:  formatIdent(strcase.ToSnake(f.Name)),
			GoType:   f.Type,
			IsFK:     f.Preload,
			IsPK:     f.Name == "ID",
			BaseType: f.BaseType,
		})
	}
	return cols
}

func findPK(cols []column) *column {
	for i := range cols {
		if cols[i].IsPK {
			return &cols[i]
		}
	}
	return nil
}

func writeCode(w io.Writer, d templateData) error {
	var b strings.Builder
	b.Grow(4 * 1024)

	writeHeader(&b, d)
	writeInsert(&b, d)
	if d.PKCol != nil {
		writeGet(&b, d)
		writeDelete(&b, d)
		writeUpdate(&b, d)
	}
	writeQuery(&b, d)

	_, err := io.WriteString(w, b.String())
	if err != nil {
		return fmt.Errorf("rawgen: failed to write generated code: %w", err)
	}
	return nil
}

func writeHeader(b *strings.Builder, d templateData) {
	fmt.Fprintf(b, "// Code generated by apigen rawgen; DO NOT EDIT.\n")
	fmt.Fprintf(b, "package queries\n\n")
	fmt.Fprintf(b, "import (\n")
	fmt.Fprintf(b, "\t\"context\"\n")
	fmt.Fprintf(b, "\t\"database/sql\"\n")
	fmt.Fprintf(b, "\t\"fmt\"\n")
	if len(d.Filters) > 0 {
		fmt.Fprintf(b, "\t\"strings\"\n")
	}
	fmt.Fprintf(b, "\n\t%q\n", d.ModelPkg)
	fmt.Fprintf(b, ")\n\n")
}

func writeInsert(b *strings.Builder, d templateData) {
	insertCols := nonPKCols(d.Columns)

	colNames := make([]string, len(insertCols))
	placeholders := make([]string, len(insertCols))
	args := make([]string, len(insertCols))
	for i, c := range insertCols {
		colNames[i] = c.ColName
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = fmt.Sprintf("%s.%s", d.Ident, c.GoName)
	}

	fmt.Fprintf(b, "// Insert%s inserts a new %s", d.ModelName, d.ModelName)
	if d.PKCol != nil {
		fmt.Fprintf(b, " and populates its generated %s.\n", d.PKCol.GoName)
	} else {
		fmt.Fprintf(b, ".\n")
	}
	fmt.Fprintf(b, "func Insert%s(ctx context.Context, db *sql.DB, %s *%s.%s) error {\n",
		d.ModelName, d.Ident, d.ModelPkgName, d.ModelName)
	fmt.Fprintf(b, "\tconst query = `INSERT INTO %s (%s) VALUES (%s)",
		d.TableName, strings.Join(colNames, ", "), strings.Join(placeholders, ", "))
	if d.PKCol != nil {
		fmt.Fprintf(b, " RETURNING %s", d.PKCol.ColName)
	}
	fmt.Fprintf(b, "`\n")

	if d.PKCol != nil {
		fmt.Fprintf(b, "\treturn db.QueryRowContext(ctx, query,\n")
		for _, a := range args {
			fmt.Fprintf(b, "\t\t%s,\n", a)
		}
		fmt.Fprintf(b, "\t).Scan(&%s.%s)\n", d.Ident, d.PKCol.GoName)
	} else {
		fmt.Fprintf(b, "\t_, err := db.ExecContext(ctx, query,\n")
		for _, a := range args {
			fmt.Fprintf(b, "\t\t%s,\n", a)
		}
		fmt.Fprintf(b, "\t)\n")
		fmt.Fprintf(b, "\treturn err\n")
	}
	fmt.Fprintf(b, "}\n\n")
}

func writeGet(b *strings.Builder, d templateData) {
	colList := columnList(d.Columns)
	scanFields := scanFieldList(d.Columns, d.Ident)

	fmt.Fprintf(b, "// Get%s retrieves a single %s by primary key.\n", d.ModelName, d.ModelName)
	fmt.Fprintf(b, "func Get%s(ctx context.Context, db *sql.DB, id %s) (*%s.%s, error) {\n",
		d.ModelName, d.PKCol.GoType, d.ModelPkgName, d.ModelName)
	fmt.Fprintf(b, "\tconst query = `SELECT %s FROM %s WHERE %s = $1`\n",
		colList, d.TableName, d.PKCol.ColName)
	fmt.Fprintf(b, "\tvar %s %s.%s\n", d.Ident, d.ModelPkgName, d.ModelName)
	fmt.Fprintf(b, "\tif err := db.QueryRowContext(ctx, query, id).Scan(\n")
	fmt.Fprintf(b, "\t\t%s,\n", scanFields)
	fmt.Fprintf(b, "\t); err != nil {\n")
	fmt.Fprintf(b, "\t\treturn nil, err\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\treturn &%s, nil\n", d.Ident)
	fmt.Fprintf(b, "}\n\n")
}

func writeQuery(b *strings.Builder, d templateData) {
	colList := columnList(d.Columns)
	scanFields := scanFieldList(d.Columns, d.Ident)

	var params strings.Builder
	params.WriteString("ctx context.Context, db *sql.DB")
	for _, f := range d.Filters {
		paramName := strcase.ToLowerCamel(f.Column)
		if f.Nullable {
			fmt.Fprintf(&params, ", %s *%s", paramName, f.GoType)
		} else {
			fmt.Fprintf(&params, ", %s %s", paramName, f.GoType)
		}
	}

	fmt.Fprintf(b, "// Query%ss retrieves %s records matching the provided filters.\n", d.ModelName, d.ModelName)
	fmt.Fprintf(b, "func Query%ss(%s) ([]*%s.%s, error) {\n", d.ModelName, params.String(), d.ModelPkgName, d.ModelName)
	fmt.Fprintf(b, "\tbaseQuery := `SELECT %s FROM %s`\n", colList, d.TableName)

	if len(d.Filters) > 0 {
		fmt.Fprintf(b, "\n\tconditions := make([]string, 0, %d)\n", len(d.Filters))
		fmt.Fprintf(b, "\targs := make([]any, 0, %d)\n", len(d.Filters))
		fmt.Fprintf(b, "\tparamIdx := 1\n\n")

		for _, f := range d.Filters {
			paramName := strcase.ToLowerCamel(f.Column)
			colName := formatIdent(strcase.ToSnake(f.Column))
			if f.Nullable {
				fmt.Fprintf(b, "\tif %s != nil {\n", paramName)
				fmt.Fprintf(b, "\t\tconditions = append(conditions, fmt.Sprintf(\"%s %s $%%d\", paramIdx))\n", colName, f.Op)
				fmt.Fprintf(b, "\t\targs = append(args, *%s)\n", paramName)
				fmt.Fprintf(b, "\t\tparamIdx++\n")
				fmt.Fprintf(b, "\t}\n")
			} else {
				fmt.Fprintf(b, "\tconditions = append(conditions, fmt.Sprintf(\"%s %s $%%d\", paramIdx))\n", colName, f.Op)
				fmt.Fprintf(b, "\targs = append(args, %s)\n", paramName)
				fmt.Fprintf(b, "\tparamIdx++\n\n")
			}
		}

		fmt.Fprintf(b, "\tif len(conditions) > 0 {\n")
		fmt.Fprintf(b, "\t\tbaseQuery += \" WHERE \" + strings.Join(conditions, \" AND \")\n")
		fmt.Fprintf(b, "\t}\n\n")
		fmt.Fprintf(b, "\trows, err := db.QueryContext(ctx, baseQuery, args...)\n")
	} else {
		fmt.Fprintf(b, "\trows, err := db.QueryContext(ctx, baseQuery)\n")
	}

	fmt.Fprintf(b, "\tif err != nil {\n")
	fmt.Fprintf(b, "\t\treturn nil, err\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\tdefer rows.Close()\n\n")
	fmt.Fprintf(b, "\tvar results []*%s.%s\n", d.ModelPkgName, d.ModelName)
	fmt.Fprintf(b, "\tfor rows.Next() {\n")
	fmt.Fprintf(b, "\t\tvar %s %s.%s\n", d.Ident, d.ModelPkgName, d.ModelName)
	fmt.Fprintf(b, "\t\tif err := rows.Scan(\n")
	fmt.Fprintf(b, "\t\t\t%s,\n", scanFields)
	fmt.Fprintf(b, "\t\t); err != nil {\n")
	fmt.Fprintf(b, "\t\t\treturn nil, err\n")
	fmt.Fprintf(b, "\t\t}\n")
	fmt.Fprintf(b, "\t\tresults = append(results, &%s)\n", d.Ident)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\tif err := rows.Err(); err != nil {\n")
	fmt.Fprintf(b, "\t\treturn nil, err\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\treturn results, nil\n")
	fmt.Fprintf(b, "}\n\n")
}

func writeDelete(b *strings.Builder, d templateData) {
	fmt.Fprintf(b, "// Delete%s deletes a %s by primary key.\n", d.ModelName, d.ModelName)
	fmt.Fprintf(b, "func Delete%s(ctx context.Context, db *sql.DB, id %s) error {\n",
		d.ModelName, d.PKCol.GoType)
	fmt.Fprintf(b, "\tconst query = `DELETE FROM %s WHERE %s = $1`\n", d.TableName, d.PKCol.ColName)
	fmt.Fprintf(b, "\tresult, err := db.ExecContext(ctx, query, id)\n")
	fmt.Fprintf(b, "\tif err != nil {\n")
	fmt.Fprintf(b, "\t\treturn err\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\trows, err := result.RowsAffected()\n")
	fmt.Fprintf(b, "\tif err != nil {\n")
	fmt.Fprintf(b, "\t\treturn err\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\tif rows == 0 {\n")
	fmt.Fprintf(b, "\t\treturn sql.ErrNoRows\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\treturn nil\n")
	fmt.Fprintf(b, "}\n\n")
}

func writeUpdate(b *strings.Builder, d templateData) {
	updateCols := nonPKCols(d.Columns)
	setClauses := make([]string, len(updateCols))
	for i, c := range updateCols {
		setClauses[i] = fmt.Sprintf("%s = $%d", c.ColName, i+1)
	}
	pkParam := fmt.Sprintf("$%d", len(updateCols)+1)

	fmt.Fprintf(b, "// Update%s updates all columns of a %s by primary key.\n", d.ModelName, d.ModelName)
	fmt.Fprintf(b, "func Update%s(ctx context.Context, db *sql.DB, %s *%s.%s) error {\n",
		d.ModelName, d.Ident, d.ModelPkgName, d.ModelName)
	fmt.Fprintf(b, "\tconst query = `UPDATE %s SET %s WHERE %s = %s`\n",
		d.TableName, strings.Join(setClauses, ", "), d.PKCol.ColName, pkParam)
	fmt.Fprintf(b, "\tresult, err := db.ExecContext(ctx, query,\n")
	for _, c := range updateCols {
		fmt.Fprintf(b, "\t\t%s.%s,\n", d.Ident, c.GoName)
	}
	fmt.Fprintf(b, "\t\t%s.%s,\n", d.Ident, d.PKCol.GoName)
	fmt.Fprintf(b, "\t)\n")
	fmt.Fprintf(b, "\tif err != nil {\n")
	fmt.Fprintf(b, "\t\treturn err\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\trows, err := result.RowsAffected()\n")
	fmt.Fprintf(b, "\tif err != nil {\n")
	fmt.Fprintf(b, "\t\treturn err\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\tif rows == 0 {\n")
	fmt.Fprintf(b, "\t\treturn sql.ErrNoRows\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\treturn nil\n")
	fmt.Fprintf(b, "}\n\n")
}

func nonPKCols(cols []column) []column {
	out := make([]column, 0, len(cols))
	for _, c := range cols {
		if !c.IsPK {
			out = append(out, c)
		}
	}
	return out
}

func columnList(cols []column) string {
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.ColName
	}
	return strings.Join(names, ", ")
}

func scanFieldList(cols []column, ident string) string {
	fields := make([]string, len(cols))
	for i, c := range cols {
		fields[i] = fmt.Sprintf("&%s.%s", ident, c.GoName)
	}
	return strings.Join(fields, ", ")
}
