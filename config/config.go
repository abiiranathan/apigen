package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config struct represents the configuration parameters
type Config struct {
	Models struct {
		Pkgs     []string `toml:"Pkgs"`     // absolute package names where models are located
		Skip     []string `toml:"Skip"`     // Slice of models(Structs) to skip
		ReadOnly []string `toml:"ReadOnly"` // For SQL Views
	} `toml:"Models"`

	Output struct {
		ServiceName string `toml:"ServiceName"` // simple name for the services default: services
		OutDir      string `toml:"OutDir"`      // Directory where to create new packages: default "."
	} `toml:"Output"`
	Overrides Overrides `toml:"overrides"`
}

type Overrides struct {
	Types  map[string]string `toml:"types"`
	Fields map[string]string `toml:"fields"`
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
	if len(cfg.Models.Pkgs) == 0 {
		return fmt.Errorf("models.Pkgs is empty in apigen.toml")
	}

	for _, pkg := range cfg.Models.Pkgs {
		if pkg == "" {
			return fmt.Errorf("error: Models.Pkgs has an empty pkg in apigen.toml")
		}

		if cfg.Output.OutDir == "." {
			cfg.Output.OutDir, _ = os.Getwd()
		} else {
			cfg.Output.OutDir, _ = filepath.Abs(cfg.Output.OutDir)
		}

	}
	return nil
}
