package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config struct represents the configuration parameters
type Config struct {
	RootPkg string `toml:"RootPkg"` // Root package name for the project
	Models  struct {
		Pkg  string   `toml:"Pkg"`  // absolute package name where models are located
		Skip []string `toml:"Skip"` // Slice of models(Structs) to skip
	} `toml:"Models"`

	Output struct {
		ServiceName  string `toml:"ServiceName"`  // simple name for the services default: services
		HandlersName string `toml:"HandlersName"` // simple name for the handlers default: handlers
		OutDir       string `toml:"OutDir"`       // Directory where to create new packages: default "."
	} `toml:"Output"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := Config{}
	err = toml.Unmarshal(b, &cfg)

	if err != nil {
		return nil, err
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, err
}

func validateConfig(cfg *Config) error {
	if cfg.Models.Pkg == "" {
		return fmt.Errorf("error: Models.Pkg configuration is a required in apigen.toml")
	}

	if cfg.Output.OutDir == "." {
		cfg.Output.OutDir, _ = os.Getwd()
	} else {
		cfg.Output.OutDir, _ = filepath.Abs(cfg.Output.OutDir)
	}
	return nil
}
