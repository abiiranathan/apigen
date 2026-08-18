package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/abiiranathan/apigen/config"
)

// dedupePreloads removes redundant prefix paths in O(N log N) time and zero extra allocations.
// Because '.' (ASCII 46) sorts before all Go identifier characters ('0'-'9', 'A'-'Z', '_', 'a'-'z'),
// any path extending "A" (e.g. "A.B") will sort immediately after "A".
func dedupePreloads(preloads []string) []string {
	n := len(preloads)
	if n <= 1 {
		return preloads
	}

	slices.Sort(preloads)

	// In-place deduplication and leaf filtering
	w := 0
	for i := 0; i < n; i++ {
		current := preloads[i]

		// Skip duplicate identical entries
		if i+1 < n && preloads[i+1] == current {
			continue
		}

		// If the next item starts with "current.", then current is a parent prefix and redundant.
		if i+1 < n {
			next := preloads[i+1]
			if len(next) > len(current) && next[len(current)] == '.' && strings.HasPrefix(next, current) {
				continue
			}
		}

		preloads[w] = current
		w++
	}

	return preloads[:w]
}

// GetPreloadMap returns a map of struct names to their respective leaf preload fields.
func GetPreloadMap(structs []StructMeta, cfg *config.Config) map[string][]string {
	structMap := make(map[string]*StructMeta, len(structs))
	for i := range structs {
		structMap[structs[i].Name] = &structs[i]
	}

	maxDepth := int(cfg.PreloadDepth)
	preloads := make(map[string][]string, len(structs))
	memo := make(map[string][]string, len(structs))

	for _, st := range structs {
		raw := collectPreloads(st.Name, structMap, maxDepth, memo)
		// Clone and deduplicate so memoized entries remain intact
		copied := make([]string, len(raw))
		copy(copied, raw)
		preloads[st.Name] = dedupePreloads(copied)
	}

	if cfg.OutputJSON {
		if err := writeJSONToFile("preload.json", preloads); err != nil {
			fmt.Printf("Error writing preload.json: %v\n", err)
		}
	}

	return preloads
}

// collectPreloads traverses the relation graph using a DFS stack with cycle detection and memoization.
func collectPreloads(
	rootName string,
	structMap map[string]*StructMeta,
	maxDepth int,
	memo map[string][]string,
) []string {
	if res, ok := memo[rootName]; ok {
		return res
	}

	root, ok := structMap[rootName]
	if !ok || len(root.Fields) == 0 {
		return nil
	}

	var results []string
	visited := []string{rootName}

	var dfs func(curr *StructMeta, prefix string, depth int)
	dfs = func(curr *StructMeta, prefix string, depth int) {
		for _, f := range curr.Fields {
			if !f.Preload {
				continue
			}

			fullPath := f.Name
			if prefix != "" {
				fullPath = prefix + "." + f.Name
			}
			results = append(results, fullPath)

			if depth >= maxDepth {
				continue
			}

			nested, exists := structMap[f.BaseType]
			if !exists {
				continue
			}

			// Prevent cyclic recursion
			if slices.Contains(visited, f.BaseType) {
				continue
			}

			visited = append(visited, f.BaseType)
			dfs(nested, fullPath, depth+1)
			visited = visited[:len(visited)-1] // Backtrack
		}
	}

	dfs(root, "", 0)
	memo[rootName] = results
	return results
}

func writeJSONToFile(filename string, data map[string][]string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
