package parser

import (
	"strings"
	"testing"

	"github.com/abiiranathan/apigen/config"
)

func boolPtr(value bool) *bool {
	return &value
}

func TestGenerateGORMServicesUsesPerQueryConfig(t *testing.T) {
	cfg := &config.Config{
		PreloadAll:   true,
		PreloadDepth: 1,
		OutputJSON:   false,
		Models: struct {
			Pkgs     []string `toml:"Pkgs"`
			Skip     []string `toml:"Skip"`
			ReadOnly []string `toml:"ReadOnly"`
		}{
			Pkgs: []string{"github.com/example/project/models"},
		},
		Output: struct {
			ServiceName string `toml:"ServiceName"`
			OutDir      string `toml:"OutDir"`
		}{
			ServiceName: "services",
			OutDir:      "gen",
		},
		Queries: config.Queries{
			Default: config.QuerySet{
				GetAll: config.QuerySettings{PreloadAll: boolPtr(false)},
			},
			Models: map[string]config.QuerySet{
				"User": {
					Get:    config.QuerySettings{PreloadAll: boolPtr(true)},
					Create: config.QuerySettings{RefetchAfterWrite: boolPtr(false)},
				},
			},
		},
	}

	structs := []StructMeta{
		{
			Name:    "User",
			PKType:  "int",
			Package: "github.com/example/project/models",
			Fields: []Field{
				{Name: "ID", Type: "int", BaseType: "int", Parent: "User"},
				{Name: "Role", Type: "Role", BaseType: "Role", Parent: "User", Preload: true},
			},
		},
	}

	generated, err := generateGORMServices(structs, cfg)
	if err != nil {
		t.Fatalf("generateGORMServices returned error: %v", err)
	}

	output := string(generated)
	if !strings.Contains(output, "return repo.getByID(id, repo.shouldPreload(true), options...)") {
		t.Fatalf("expected Get to use the model-specific preload default")
	}
	if !strings.Contains(output, "db := repo.applyConfiguredPreloads(repo.DB, repo.shouldPreload(false))") {
		t.Fatalf("expected GetAll to use the default query preload override")
	}
	if strings.Contains(output, "tmpRecord, err := repo.getByID(user.ID, true, options...)") {
		t.Fatalf("expected Create refetch block to be omitted when disabled")
	}
	if !strings.Contains(output, "preloadConfigured bool") {
		t.Fatalf("expected generated repo to track runtime preload overrides")
	}
}
