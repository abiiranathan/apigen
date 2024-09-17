package parser

import (
	"sort"
	"strings"
)

func filterPreloads(dependencies map[string][]string) map[string][]string {
	filteredPreloads := make(map[string][]string)

	for model, preloads := range dependencies {
		// Sort preloads by length in descending order
		sort.Slice(preloads, func(i, j int) bool {
			return len(preloads[i]) > len(preloads[j])
		})

		uniquePreloads := make([]string, 0, len(preloads))
		for _, preload := range preloads {
			isUnique := true
			for i := 0; i < len(uniquePreloads); i++ {
				if strings.HasPrefix(preload+".", uniquePreloads[i]+".") {
					// Current preload is a parent of an existing preload, replace it
					uniquePreloads = append(uniquePreloads[:i], uniquePreloads[i+1:]...)
					i--
				} else if strings.HasPrefix(uniquePreloads[i]+".", preload+".") {
					// Current preload is a child of an existing preload, skip it
					isUnique = false
					break
				}
			}
			if isUnique {
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
