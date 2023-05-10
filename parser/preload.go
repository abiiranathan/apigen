package parser

import (
	"sort"
)

// filterPreloads returns a map of the longest unique strings representing the Gorm preload string.
// Keys are Model(Struct) names.
func filterPreloads(dependencies map[string][]string) map[string][]string {
	// Sort dependencies in descending order
	for _, deps := range dependencies {
		sort.Slice(deps, func(i, j int) bool {
			return len(deps[i]) > len(deps[j])
		})
	}

	// Map to keep track of filtered dependencies
	filtered := make(map[string][]string)
	for key, deps := range dependencies {
		var res []string
		used := make(map[string]bool)
		for _, dep := range deps {
			skip := false
			for i := 1; i < len(dep); i++ {
				prefix := dep[:i]
				if used[prefix] {
					skip = true
					break
				}
			}
			if !skip {
				res = append(res, dep)
				for i := 1; i < len(dep); i++ {
					prefix := dep[:i]
					used[prefix] = true
				}
			}
		}
		filtered[key] = res
	}
	return filtered
}

// GetPreloadMap returns a map of preload statements for each table.
// Preload statements are generated for all foreignKey fields and many2many fields.
// Forexample:
//
//	[User][]string{"Profile.Avatars"}
func GetPreloadMap(structs []StructMeta) map[string][]string {
	inputs := Map(structs)
	dict := make(map[string][]string)

	var visit func(structName string, visited map[string]bool) []string
	visit = func(structName string, visited map[string]bool) []string {
		if visited[structName] {
			// This struct has already been visited,
			// to avoid cycles we return an empty list of dependencies
			return []string{}
		}

		visited[structName] = true
		s := inputs[structName]

		var dependencies []string
		for _, field := range s.Fields {
			if field.Preload {
				dependencies = append(dependencies, field.Name)

				// Get field dependencies if any
				if _, ok := inputs[field.BaseType]; ok {
					fieldDependencies := visit(field.BaseType, visited)
					for _, dep := range fieldDependencies {
						dependencies = append(dependencies, field.Name+"."+dep)
					}
				}
			}
		}
		return dependencies
	}

	for _, s := range structs {
		visited := make(map[string]bool)
		dict[s.Name] = visit(s.Name, visited)
	}
	return filterPreloads(dict)
}
