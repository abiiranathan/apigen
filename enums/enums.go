package enums

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/tools/go/packages"
)

var enCaser = cases.Title(language.English)

// Data to pass to the template
type templateData struct {
	PkgName      string   // Name of the package.
	TypeName     string   // The Enum type in generated go code.
	Values       []string // Enum options
	WritePKGDecl bool     // Whether to write imports and package declaration
}

// Enumer interface embeds sql.Scanner interface, database/driver.Valuer.
// Has 2 methods ValidValues()[]string to return valid enums and DatabaseType()
// that returns the sql data type.
type Enumer interface {
	sql.Scanner
	driver.Valuer
	ValidValues() []string
	DatabaseType() string
}

func ImplementsEnumer(field reflect.StructField) bool {
	fieldPtr := reflect.New(field.Type)
	interfaceType := reflect.TypeOf((*Enumer)(nil)).Elem()
	return fieldPtr.Type().AssignableTo(interfaceType)
}

type pkgInfo struct {
	Name string
	Path string
}

const mode packages.LoadMode = packages.NeedName |
	packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo

// Map representing all packages processed.
// Each key is a unique package path in the file system.
//
//	pkgInfoMap := [pkgInfo][fileName][enumType][]{"type1", "type2",...}
type pkgInfoMap map[pkgInfo]map[string]map[string][]string

func getPKGEnumMap(pkgName string) (pkgInfoMap, error) {
	cfg := &packages.Config{
		Mode:  mode,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, pkgName)
	if err != nil {
		packages.PrintErrors(pkgs)
		return nil, err
	}

	mapInfo := make(pkgInfoMap)
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			absPath, err := filepath.Abs(pkg.Fset.File(file.Pos()).Name())
			if err != nil {
				return nil, err
			}

			for _, decl := range file.Decls {
				// We only care about enumerated constants
				if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.CONST {
					for _, spec := range genDecl.Specs {
						if valueSpec, ok := spec.(*ast.ValueSpec); ok && valueSpec.Type != nil {
							// Only care about identifiers
							keySpec, ok := valueSpec.Type.(*ast.Ident)
							if !ok {
								continue
							}

							enumTypeName := keySpec.Name
							for _, value := range valueSpec.Values {
								if basicLit, ok := value.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
									pkgKey := pkgInfo{Name: pkg.Name, Path: pkg.PkgPath}
									if mapInfo[pkgKey] == nil {
										mapInfo[pkgKey] = make(map[string]map[string][]string)
									}

									if mapInfo[pkgKey][absPath] == nil {
										mapInfo[pkgKey][absPath] = make(map[string][]string)
									}

									constantValue := strings.Trim(basicLit.Value, `"`)
									mapInfo[pkgKey][absPath][enumTypeName] = append(mapInfo[pkgKey][absPath][enumTypeName], constantValue)
								}
							}
						}
					}
				}
			}
		}
	}

	return mapInfo, nil
}

// Surround each value with single quote and return a comma-seperated string
func QuoteEnums(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("'%s'", v)
	}
	return strings.Join(quoted, ",")
}

// Passes data to the string template that is executed into w.
func parseTemplate(w io.Writer, data templateData) error {
	tmpl, err := template.New("tmpl").Funcs(template.FuncMap{
		"toCamelCase": func(s string) string {
			return strcase.ToCamel(enCaser.String(s))
		},
		"ToSnakeCase": func(s string) string {
			return strcase.ToSnake(enCaser.String(s))
		},
	}).Parse(templateString)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}

	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	return nil
}

// For all enumerated constants, generate enums that satisfy the Enummer interface
// suffix is appended to each filename e.g Uses _enum by default.
// Returns sql string that you can use to create these ENUM types in postgres
func GenerateEnums(pkgNames []string, suffix ...string) (sql string, err error) {
	if len(suffix) == 0 {
		suffix = append(suffix, "_enum")
	}

	base := suffix[0] + ".go"

	globalBuffer := new(bytes.Buffer)
	for pkgIndex, pkgName := range pkgNames {
		enums, err := getPKGEnumMap(pkgName)
		if err != nil {
			return "", fmt.Errorf("getPKGEnumMap(): %w", err)
		}

		// Create types for postgres
		type enum_type struct {
			name   string
			values []string
		}
		enumsTypes := []enum_type{}

		for pkgPath, pkgData := range enums {
			for filename, enumDict := range pkgData {
				buffer := bytes.Buffer{}
				count := 0
				for key, values := range enumDict {
					err := parseTemplate(&buffer, templateData{
						PkgName:      pkgPath.Name,
						TypeName:     key,
						Values:       values,
						WritePKGDecl: count == 0 && pkgIndex == 0,
					})

					if err != nil {
						return "", err
					}
					enumsTypes = append(enumsTypes, enum_type{
						name:   key,
						values: values,
					})
				}

				// increase count
				count++

				filebase := strings.Split(filepath.Base(filename), ".")[0]
				absPath := filepath.Join(filepath.Dir(filename), filebase+base)

				// Format source
				b, err := format.Source(buffer.Bytes())
				if err != nil {
					return "", fmt.Errorf("error format source file: %w, source: %s", err, b)
				}

				// Write contents to the file.
				if err := os.WriteFile(absPath, b, 0644); err != nil {
					return "", fmt.Errorf("error writing buffer: %w", err)
				}
			}
		}

		// Write the enum types
		buf := new(bytes.Buffer)

		for _, item := range enumsTypes {
			// convert to name to snake case.
			fmt.Fprintf(buf, "CREATE TYPE %s AS ENUM(%s);\n", strcase.ToSnake(item.name), QuoteEnums(item.values))
		}
		globalBuffer.Write(buf.Bytes())
	}

	return globalBuffer.String(), nil
}

var templateString = `
{{$typeName := .TypeName}}{{$values := .Values}}
{{if .WritePKGDecl}}
// Code generated by "apigen"; DO NOT EDIT.
package {{.PkgName}}
import (
	"database/sql/driver"
	"fmt"
)
{{end}}



func (e {{.TypeName}}) IsValid() bool {
	for _, val := range e.ValidValues() {
		if val == string(e) {
			return true
		}
	}
	return false
}

func (e {{.TypeName}}) ValidValues() []string {
	return []string{
		{{range $val := $values -}}
			   "{{$val -}}",
		{{end -}}
	}
}

func (e *{{.TypeName}}) Scan(src interface{}) error {
	source, ok := src.(string)
	if !ok {
		return fmt.Errorf("invalid value for %s: %s", "{{.TypeName}}", source)
	}
	*e = {{.TypeName}}(source)
	return nil
}

func (e {{.TypeName}}) Value() (driver.Value, error) {
	if !e.IsValid() {
		return nil, fmt.Errorf("invalid value for %s", "{{.TypeName}}")
	}
	return string(e), nil
}

func (e {{.TypeName}}) DatabaseType() string {
	return "{{.TypeName | ToSnakeCase }}"
}

func (e {{.TypeName}}) GormDataType() string {
	return "{{.TypeName | ToSnakeCase }}"
}

`
