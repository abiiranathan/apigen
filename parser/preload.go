package parser

import (
	"slices"
	"sort"
	"strings"
)

// filterPreloads returns a map of the longest unique strings representing the Gorm preload string.
// Keys are Model(Struct) names.
// Forexample: {"User": []string{"Profile", "Profile.Addresses", "Profile.Cards"}}
// can omit "Profile" since its captured in other 2.
func filterPreloads(dependencies map[string][]string) map[string][]string {
	filteredPreloads := make(map[string][]string)

	// Sort dependencies in descending order
	for _, deps := range dependencies {
		sort.Slice(deps, func(i, j int) bool {
			return len(deps[i]) > len(deps[j])
		})
	}

	for model, preloads := range dependencies {
		uniquePreloads := make([]string, 0, len(preloads))

		for _, preload := range preloads {
			if !slices.Contains(filteredPreloads[model], preload) {
				// Check if preload is a substring of any other preload
				// If yes, then skip this preload
				shouldSkip := false
				for _, existingPreload := range uniquePreloads {
					if strings.HasPrefix(existingPreload, preload) {
						shouldSkip = true
						break
					}
				}

				if shouldSkip {
					continue
				}
				uniquePreloads = append(uniquePreloads, preload)
			}
		}

		filteredPreloads[model] = uniquePreloads
	}

	return filteredPreloads
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
