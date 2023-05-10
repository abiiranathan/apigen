package parser

import (
	"bytes"
	"fmt"
	"go/build"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/abiiranathan/apigen/config"
)

// Returns the go import path for a given directory targetDir
func GetModulePath(targetDir string) (string, error) {
	// Retrieve the build context
	ctx := build.Default

	// Find the package information for the current directory
	pkg, err := ctx.ImportDir(targetDir, build.ImportComment)
	if err != nil {
		return "", err
	}
	// Return the import  path
	return pkg.ImportPath, nil
}

// generateServices generates the service.go file
func GenerateGORMServices(cfg *config.Config, structMetaData []StructMeta) (err error) {
	b, err := generateGORMServices(
		cfg.Output.ServiceName,
		cfg.Models.Pkg,
		structMetaData,
		cfg.Models.Skip...)

	if err != nil {
		return err
	}

	targetDir := filepath.Join(cfg.Output.OutDir, cfg.Output.ServiceName)
	err = createDirectory(targetDir)
	if err != nil {
		return fmt.Errorf("error creating directory %s: %w", cfg.Output.ServiceName, err)
	}

	targetPath := filepath.Join(targetDir, cfg.Output.ServiceName+".go")
	err = writeFile(targetPath, b)
	if err != nil {
		return fmt.Errorf("error writing to file %q: %w", targetPath, err)
	}

	// Generate the postgres database connection helpers
	dbPath := filepath.Join(targetDir, "postgres.go")
	err = writeFile(dbPath, []byte(fmt.Sprintf(dbText, cfg.Output.ServiceName)))
	if err != nil {
		fmt.Printf("error writing to postgres.go helper %q: %v", dbPath, err)
	}
	return nil
}

// generateHandlers generates the handlers.go and validation.go files
func GenerateFiberHandlers(cfg *config.Config, structMetaData []StructMeta) error {
	targetDir := filepath.Join(cfg.Output.OutDir, cfg.Output.ServiceName)
	svcPkg, err := GetModulePath(targetDir)
	if err != nil {
		return fmt.Errorf("unable to find import path for services: %s", err)
	}

	b, bh, err := generateHandlers(
		structMetaData,
		cfg.Models.Pkg,
		svcPkg,
		cfg.Output.HandlersName,
		cfg.Models.Skip...,
	)

	if err != nil {
		return err
	}

	handlerDir := filepath.Join(cfg.Output.OutDir, cfg.Output.HandlersName)
	err = createDirectory(handlerDir)
	if err != nil {
		return fmt.Errorf("error creating directory %s: %w", cfg.Output.HandlersName, err)
	}

	handlerPath := filepath.Join(handlerDir, cfg.Output.HandlersName+".go")
	err = writeFile(handlerPath, b)
	if err != nil {
		return fmt.Errorf("error writing to file %q: %w", handlerPath, err)
	}

	validationPath := filepath.Join(handlerDir, "validation.go")
	err = writeFile(validationPath, bh)
	if err != nil {
		return fmt.Errorf("error writing to file %q: %w", validationPath, err)
	}

	return nil
}

// createDirectory creates a directory with the given path
func createDirectory(path string) error {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}
	return os.Chmod(path, 0755)
}

// writeFile writes data to a file with the given path
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)

}

// Main entry point to generate services and handlers for the your
// go structs as specified in the .toml configuration.
func GenerateCode(cfg *config.Config) error {
	metadata := Parse(cfg.Models.Pkg)

	// Generate code
	err := GenerateGORMServices(cfg, metadata)
	if err != nil {
		return fmt.Errorf("error generating services: %v", err)
	}

	if err := GenerateFiberHandlers(cfg, metadata); err != nil {
		return fmt.Errorf("error generating handlers: %v", err)

	}
	return generateRoutes(cfg, metadata)
}

func generateRoutes(cfg *config.Config, structMetaData []StructMeta) error {
	// Generate routes
	targetDir := filepath.Join(cfg.Output.OutDir, cfg.Output.HandlersName)
	HandlerImport, err := GetModulePath(targetDir)
	if err != nil {
		return fmt.Errorf("unable to find import path for handlers: %s", err)
	}

	models := make([]string, 0, len(structMetaData))
	for _, m := range structMetaData {
		if sliceContains(cfg.Models.Skip, m.Name) {
			continue
		}

		models = append(models, m.Name)
	}

	data := routeTmplData{
		HandlerPkgName: cfg.Output.HandlersName,
		HandlerImport:  HandlerImport,
		Models:         models,
	}

	tmpl, err := template.New("routesTmpl").Funcs(template.FuncMap{
		"ToLower": strings.ToLower,
		"Pluralize": func(s string) string {
			return s + "s"
		},
	}).Parse(routesTemplate)

	if err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	err = tmpl.Execute(buffer, data)
	if err != nil {
		return err
	}

	b, err := format.Source(buffer.Bytes())
	if err != nil {
		fmt.Println(buffer.String())
		return err
	}

	// Must already exist since handlers are already created
	handlerDir := filepath.Join(cfg.Output.OutDir, cfg.Output.HandlersName)
	handlerPath := filepath.Join(handlerDir, "routes.go")
	return writeFile(handlerPath, b)
}
