package parser

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"sort"

	"github.com/abiiranathan/apigen/config"
)

// GetModulePath retrieves the module path for the given target directory using the Go build context.
func GetModulePath(targetDir string) (string, error) {
	// Retrieve the build context
	ctx := build.Default

	// Find the package information for the current directory
	pkg, err := ctx.ImportDir(targetDir, build.ImportComment)
	if err != nil {
		return "", err
	}
	// Return the import path
	return pkg.ImportPath, nil
}

// GenerateGORMServices generates service files in the configured output package.
func GenerateGORMServices(cfg *config.Config, structMetaData []StructMeta) (err error) {
	files, err := generateGORMServiceFiles(structMetaData, cfg)
	if err != nil {
		return err
	}

	targetDir := filepath.Join(cfg.Output.OutDir, cfg.Output.ServiceName)
	err = createDirectory(targetDir)
	if err != nil {
		return fmt.Errorf("error creating directory %s: %w", cfg.Output.ServiceName, err)
	}

	legacyPath := filepath.Join(targetDir, cfg.Output.ServiceName+".go")
	if _, statErr := os.Stat(legacyPath); statErr == nil {
		if removeErr := os.Remove(legacyPath); removeErr != nil {
			return fmt.Errorf("error removing legacy file %q: %w", legacyPath, removeErr)
		}
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		targetPath := filepath.Join(targetDir, name)
		err = writeFile(targetPath, files[name])
		if err != nil {
			return fmt.Errorf("error writing to file %q: %w", targetPath, err)
		}
	}

	// Generate the postgres database connection helpers
	dbPath := filepath.Join(targetDir, "database.go")
	err = writeFile(dbPath, fmt.Appendf(nil, dbText, cfg.Output.ServiceName))
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
