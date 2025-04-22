package parser

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"

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
	b, err := generateGORMServices(structMetaData, cfg)
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
	dbPath := filepath.Join(targetDir, "database.go")
	err = writeFile(dbPath, []byte(fmt.Sprintf(dbText, cfg.Output.ServiceName)))
	if err != nil {
		fmt.Printf("error writing to database.go helper %q: %v", dbPath, err)
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
