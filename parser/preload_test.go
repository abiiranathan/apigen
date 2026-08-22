package parser

import (
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/abiiranathan/apigen/config"
)

// Helper to build test structs
func makeSampleStructs() []StructMeta {
	return []StructMeta{
		{
			Name: "User",
			Fields: []Field{
				{Name: "ID", Type: "int", BaseType: "int"},
				{Name: "Role", Type: "Role", BaseType: "Role", Preload: true},
				{Name: "Tags", Type: "[]Tag", BaseType: "Tag", Preload: true},
			},
		},
		{
			Name: "Role",
			Fields: []Field{
				{Name: "ID", Type: "int64", BaseType: "int64"},
				{Name: "Name", Type: "string", BaseType: "string"},
				{Name: "Permissions", Type: "[]Permission", BaseType: "Permission", Preload: true},
			},
		},
		{
			Name: "Permission",
			Fields: []Field{
				{Name: "ID", Type: "int64", BaseType: "int64"},
				{Name: "Name", Type: "string", BaseType: "string"},
			},
		},
		{
			Name: "Tag",
			Fields: []Field{
				{Name: "ID", Type: "int64", BaseType: "int64"},
				{Name: "Issues", Type: "[]Issue", BaseType: "Issue", Preload: true},
				{Name: "Role", Type: "Role", BaseType: "Role", Preload: true},
			},
		},
		{
			Name: "Issue",
			Fields: []Field{
				{Name: "ID", Type: "int64", BaseType: "int64"},
				{Name: "Comments", Type: "[]Comment", BaseType: "Comment", Preload: true},
			},
		},
		{
			Name: "Comment",
			Fields: []Field{
				{Name: "ID", Type: "int", BaseType: "int"},
				// Self-referencing relationship (replies)
				{Name: "Comments", Type: "[]Comment", BaseType: "Comment", Preload: true},
			},
		},
		{
			// Mutually recursive loop (A -> B -> A)
			Name: "NodeA",
			Fields: []Field{
				{Name: "ID", Type: "int", BaseType: "int"},
				{Name: "B", Type: "NodeB", BaseType: "NodeB", Preload: true},
			},
		},
		{
			Name: "NodeB",
			Fields: []Field{
				{Name: "ID", Type: "int", BaseType: "int"},
				{Name: "A", Type: "NodeA", BaseType: "NodeA", Preload: true},
			},
		},
	}
}

func TestDedupePreloads(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single element",
			input:    []string{"Role"},
			expected: []string{"Role"},
		},
		{
			name:     "direct prefix subsumption",
			input:    []string{"Tags", "Tags.Issues", "Tags.Issues.Comments"},
			expected: []string{"Tags.Issues.Comments"},
		},
		{
			name:     "siblings with common prefix retained",
			input:    []string{"Tags.Role", "Tags.Issues", "Tags"},
			expected: []string{"Tags.Issues", "Tags.Role"},
		},
		{
			name:     "duplicate entries removed",
			input:    []string{"Role", "Role", "Role.Permissions", "Role"},
			expected: []string{"Role.Permissions"},
		},
		{
			name:     "exact duplicates without sub-paths",
			input:    []string{"Tags", "Tags", "Role", "Role"},
			expected: []string{"Role", "Tags"},
		},
		{
			name:     "similar prefix that is not a path separator",
			input:    []string{"User", "UserTag", "User.Role"},
			expected: []string{"User.Role", "UserTag"},
		},
		{
			name:     "complex mixed paths",
			input:    []string{"A", "A.B", "A.B.C", "A.D", "X", "X.Y", "Z"},
			expected: []string{"A.B.C", "A.D", "X.Y", "Z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupePreloads(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("dedupePreloads() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetPreloadMap_DepthZero(t *testing.T) {
	structs := makeSampleStructs()
	cfg := &config.Config{
		PreloadDepth: 0,
		OutputJSON:   false,
	}

	preloads := GetPreloadMap(structs, cfg)

	expectedUser := []string{"Role", "Tags"}
	if !slices.Equal(preloads["User"], expectedUser) {
		t.Errorf("User preloads at depth 0 = %v, want %v", preloads["User"], expectedUser)
	}

	expectedRole := []string{"Permissions"}
	if !slices.Equal(preloads["Role"], expectedRole) {
		t.Errorf("Role preloads at depth 0 = %v, want %v", preloads["Role"], expectedRole)
	}
}

func TestGetPreloadMap_DepthOne(t *testing.T) {
	structs := makeSampleStructs()
	cfg := &config.Config{
		PreloadDepth: 1,
		OutputJSON:   false,
	}

	preloads := GetPreloadMap(structs, cfg)

	// User -> Role.Permissions
	// User -> Tags.Issues
	// User -> Tags.Role
	expectedUser := []string{
		"Role.Permissions",
		"Tags.Issues",
		"Tags.Role",
	}

	if !slices.Equal(preloads["User"], expectedUser) {
		t.Errorf("User preloads at depth 1 = %v, want %v", preloads["User"], expectedUser)
	}
}

func TestGetPreloadMap_DepthTwo(t *testing.T) {
	structs := makeSampleStructs()
	cfg := &config.Config{
		PreloadDepth: 2,
		OutputJSON:   false,
	}

	preloads := GetPreloadMap(structs, cfg)

	// User -> Role.Permissions
	// User -> Tags.Issues.Comments
	// User -> Tags.Role.Permissions
	expectedUser := []string{
		"Role.Permissions",
		"Tags.Issues.Comments",
		"Tags.Role.Permissions",
	}

	if !slices.Equal(preloads["User"], expectedUser) {
		t.Errorf("User preloads at depth 2 = %v, want %v", preloads["User"], expectedUser)
	}
}

func TestGetPreloadMap_CycleDetection(t *testing.T) {
	structs := makeSampleStructs()
	cfg := &config.Config{
		PreloadDepth: 10, // High depth to trigger infinite loop if cycle detection fails
		OutputJSON:   false,
	}

	// Should terminate cleanly and not stack overflow or panic
	preloads := GetPreloadMap(structs, cfg)

	// Comment -> Comments (self-referencing: should stop after first recursion)
	if len(preloads["Comment"]) == 0 {
		t.Errorf("expected Comment preloads to have at least one entry")
	}

	// NodeA -> B -> A (mutual loop: should terminate)
	expectedNodeA := []string{"B.A"}
	if !slices.Equal(preloads["NodeA"], expectedNodeA) {
		t.Errorf("NodeA preloads = %v, want %v", preloads["NodeA"], expectedNodeA)
	}
}

func TestGetPreloadMap_JSONOutput(t *testing.T) {
	structs := makeSampleStructs()
	cfg := &config.Config{
		PreloadDepth: 1,
		OutputJSON:   true,
	}

	const outPath = "preload.json"
	_ = os.Remove(outPath)
	defer os.Remove(outPath)

	_ = GetPreloadMap(structs, cfg)

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Fatalf("expected %s to be created when OutputJSON=true", outPath)
	}
}

// ---------------- Benchmarks ----------------

func BenchmarkDedupePreloads(b *testing.B) {
	input := []string{
		"User",
		"User.Role",
		"User.Role.Permissions",
		"User.Tags",
		"User.Tags.Issues",
		"User.Tags.Issues.Comments",
		"User.Tags.Role",
		"User.Tags.Role.Permissions",
		"Account",
		"Account.Billing",
		"Account.Billing.Invoices",
		"Settings",
	}

	b.ReportAllocs()

	for b.Loop() {
		// Clone slice since dedupePreloads modifies in place
		cp := make([]string, len(input))
		copy(cp, input)
		_ = dedupePreloads(cp)
	}
}

func BenchmarkGetPreloadMap(b *testing.B) {
	structs := makeSampleStructs()
	cfg := &config.Config{
		PreloadDepth: 3,
		OutputJSON:   false,
	}

	b.ReportAllocs()

	for b.Loop() {
		_ = GetPreloadMap(structs, cfg)
	}
}
