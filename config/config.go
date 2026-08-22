// Package config provides functionality to load and validate the configuration for the apigen tool.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config struct represents the configuration parameters
type Config struct {
	PreloadAll bool `toml:"PreloadAll"` // Preload all the relations
	OutputJSON bool `toml:"OutputJson"` // Output JSON file with all the relations

	// LazyPreload when true, sets preloadAll=false by default in generated services.
	// Preloads are only applied when the caller explicitly opts in via .PreloadAll(true) or Preload() option.
	// Defaults to false for backward compatibility.
	LazyPreload bool `toml:"LazyPreload"`

	// RefetchAfterWrite when true (default), re-fetches the record after Create/Update to populate associations.
	// Set to false to skip refetch — callers can call .Get(id) explicitly when they need associations.
	RefetchAfterWrite *bool `toml:"RefetchAfterWrite"`

	Models struct {
		Pkgs     []string `toml:"Pkgs"`     // absolute package names where models are located
		Skip     []string `toml:"Skip"`     // Slice of models(Structs) to skip
		ReadOnly []string `toml:"ReadOnly"` // For SQL Views
	} `toml:"Models"`
	Output struct {
		ServiceName string `toml:"ServiceName"` // simple name for the services default: services
		OutDir      string `toml:"OutDir"`      // Directory where to create new packages: default "."
	} `toml:"Output"`
	Overrides    Overrides `toml:"overrides"`
	PreloadDepth uint      `toml:"PreloadDepth"` // Preload depth for nested relations
	Queries      Queries   `toml:"Queries"`
}

type QuerySettings struct {
	PreloadAll        *bool `toml:"PreloadAll"`
	RefetchAfterWrite *bool `toml:"RefetchAfterWrite"`
}

type QuerySet struct {
	Get                  QuerySettings `toml:"Get"`
	GetAll               QuerySettings `toml:"GetAll"`
	GetPaginated         QuerySettings `toml:"GetPaginated"`
	FindOne              QuerySettings `toml:"FindOne"`
	FindMany             QuerySettings `toml:"FindMany"`
	Create               QuerySettings `toml:"Create"`
	CreateMany           QuerySettings `toml:"CreateMany"`
	Update               QuerySettings `toml:"Update"`
	PartialUpdate        QuerySettings `toml:"PartialUpdate"`
	PartialUpdateWithMap QuerySettings `toml:"PartialUpdateWithMap"`
}

type Queries struct {
	Default QuerySet            `toml:"Default"`
	Models  map[string]QuerySet `toml:"Models"`
}

type EffectiveQuerySettings struct {
	PreloadAll        bool
	RefetchAfterWrite bool
}

// ShouldRefetchAfterWrite returns whether to refetch after write operations.
// Defaults to true when not explicitly set (backward compatible).
func (c *Config) ShouldRefetchAfterWrite() bool {
	if c.RefetchAfterWrite == nil {
		return true
	}
	return *c.RefetchAfterWrite
}

func (c *Config) DefaultPreloadAll() bool {
	if c.LazyPreload {
		return false
	}
	return c.PreloadAll
}

func (c *Config) QueryConfig(model, operation string) EffectiveQuerySettings {
	effective := EffectiveQuerySettings{
		PreloadAll:        c.DefaultPreloadAll(),
		RefetchAfterWrite: c.ShouldRefetchAfterWrite(),
	}

	effective = applyQuerySettings(effective, c.Queries.Default.lookup(operation))
	if modelSettings, ok := c.Queries.Models[model]; ok {
		effective = applyQuerySettings(effective, modelSettings.lookup(operation))
	}

	return effective
}

func applyQuerySettings(base EffectiveQuerySettings, override QuerySettings) EffectiveQuerySettings {
	if override.PreloadAll != nil {
		base.PreloadAll = *override.PreloadAll
	}
	if override.RefetchAfterWrite != nil {
		base.RefetchAfterWrite = *override.RefetchAfterWrite
	}
	return base
}

func (q QuerySet) lookup(operation string) QuerySettings {
	switch operation {
	case "Get":
		return q.Get
	case "GetAll":
		return q.GetAll
	case "GetPaginated":
		return q.GetPaginated
	case "FindOne":
		return q.FindOne
	case "FindMany":
		return q.FindMany
	case "Create":
		return q.Create
	case "CreateMany":
		return q.CreateMany
	case "Update":
		return q.Update
	case "PartialUpdate":
		return q.PartialUpdate
	case "PartialUpdateWithMap":
		return q.PartialUpdateWithMap
	default:
		return QuerySettings{}
	}
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

	cfg := new(Config)
	if err = toml.Unmarshal(b, cfg); err != nil {
		return nil, err
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, err
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
