package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

type Arg struct {
	Name      string
	Type      string
	IsPointer bool
}

type Function struct {
	Name        string
	Pkg         string
	Receiver    string
	PointerRecv bool
	Args        []Arg
	ReturnTypes []Arg
}

// ParseFunctions parses Go source code and returns a slice of functions that are declared in the code.
// It supports functions with pointer receivers, variadic arguments, and return types that are not simple identifiers.
//
// Returns:
// A slice of Function structs, each representing a function declared in the input code.
//
// Example:
//
//	functions := ParseFunctions(`
//	  package main
//	  import "fmt"
//
//	  type Person struct {
//	      Name string
//	      Age  int
//	  }
//
//	  func (p *Person) Greet(other *Person, msg string) (greeting string, err error) {
//	      return fmt.Sprintf("%s says hello to %s", p.Name, other.Name), nil
//	  }
//
//	  func (p Person) Grow(n int) []int {
//	      return make([]int, n)
//	  }
//	`)
//
//	for _, fxn := range functions {
//	    fmt.Println(fxn.Name)
//	    fmt.Println(fxn.Pkg)
//	    fmt.Println(fxn.PointerRecv)
//	    for _, arg := range fxn.Args {
//	        fmt.Printf("%s %s %v\n", arg.Name, arg.Type, arg.IsPointer)
//	    }
//	    for _, ret := range fxn.ReturnTypes {
//	        fmt.Printf("%s %s %v\n", ret.Name, ret.Type, ret.IsPointer)
//	    }
//	}
func ParseFunctions(src string) (funcs []Function) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		panic(err)
	}

	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Recv != nil {
			var ident *ast.Ident
			var ptrReciever bool

			if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
				if identFier, ok := starExpr.X.(*ast.Ident); ok {
					ident = identFier
					ptrReciever = true
				}
			} else {
				if identFier, ok := funcDecl.Recv.List[0].Type.(*ast.Ident); ok {
					ident = identFier
					ptrReciever = false
				}
			}

			if ident == nil {
				continue
			}

			fxn := Function{
				Name:        funcDecl.Name.Name,
				Pkg:         file.Name.Name,
				Receiver:    ident.Name,
				PointerRecv: ptrReciever,
				Args:        []Arg{},
				ReturnTypes: []Arg{},
			}

			// Iterate through all parameters
			for _, param := range funcDecl.Type.Params.List {
				argType, isPtr := formatFieldType(param.Type)
				for _, name := range param.Names {
					fxn.Args = append(fxn.Args, Arg{
						Name:      name.Name,
						Type:      argType,
						IsPointer: isPtr,
					})
				}
			}

			// Iterate through all return types
			if funcDecl.Type.Results != nil {
				for _, result := range funcDecl.Type.Results.List {
					resultType, isPtr := formatFieldType(result.Type)
					var name string
					if len(result.Names) > 0 {
						name = result.Names[0].Name
					}

					fxn.ReturnTypes = append(fxn.ReturnTypes, Arg{
						Name:      name,
						Type:      resultType,
						IsPointer: isPtr,
					})
				}
			}
			funcs = append(funcs, fxn)
		}

	}

	return funcs
}

func formatFieldType(t ast.Expr) (fieldType string, isPtr bool) {
	switch typ := t.(type) {
	case *ast.Ident:
		return typ.Name, false
	case *ast.StarExpr:
		fieldType, _ = formatFieldType(typ.X)
		return "*" + fieldType, true
	case *ast.ArrayType:
		arrayLen := getArrayLen(typ)
		fieldType, _ = formatFieldType(typ.Elt)
		if arrayLen == 0 {
			return "[]" + fieldType, false
		} else {
			return fmt.Sprintf("[%d]%s", arrayLen, fieldType), false
		}
	default:
		panic(fmt.Sprintf("unsupported field type %T", typ))
	}
}
