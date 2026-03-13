package config

import "testing"

func boolPtr(value bool) *bool {
	return &value
}

func TestQueryConfigUsesGlobalDefaults(t *testing.T) {
	cfg := &Config{
		PreloadAll:  false,
		LazyPreload: true,
	}

	settings := cfg.QueryConfig("User", "GetAll")
	if settings.PreloadAll {
		t.Fatalf("expected lazy preload to disable default preloading")
	}
	if !settings.RefetchAfterWrite {
		t.Fatalf("expected refetch after write to default to true")
	}
}

func TestQueryConfigAppliesDefaultAndModelOverrides(t *testing.T) {
	cfg := &Config{
		PreloadAll: false,
		Queries: Queries{
			Default: QuerySet{
				GetAll: QuerySettings{PreloadAll: boolPtr(true)},
				Create: QuerySettings{RefetchAfterWrite: boolPtr(true)},
			},
			Models: map[string]QuerySet{
				"User": {
					GetAll: QuerySettings{PreloadAll: boolPtr(false)},
					Create: QuerySettings{RefetchAfterWrite: boolPtr(false)},
				},
			},
		},
	}

	getAllSettings := cfg.QueryConfig("User", "GetAll")
	if getAllSettings.PreloadAll {
		t.Fatalf("expected model-specific GetAll preload override to win")
	}

	createSettings := cfg.QueryConfig("User", "Create")
	if createSettings.RefetchAfterWrite {
		t.Fatalf("expected model-specific Create refetch override to win")
	}

	otherModelSettings := cfg.QueryConfig("Role", "GetAll")
	if !otherModelSettings.PreloadAll {
		t.Fatalf("expected default GetAll preload override for other models")
	}
}
