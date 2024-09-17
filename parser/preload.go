package parser

import (
	"fmt"
	"slices"
	"strings"
)

// dedupePreloads removes redundant preloads where a prefix is a subset of another preload.
func dedupePreloads(preloads []string) []string {
	if len(preloads) == 0 {
		return preloads
	}

	uniquePreloads := make([]string, 0, len(preloads))

	for _, preload := range preloads {
		isUnique := true
		for i, unique := range uniquePreloads {
			if strings.HasPrefix(preload+".", unique+".") {
				uniquePreloads = append(uniquePreloads[:i], uniquePreloads[i+1:]...)
			} else if strings.HasPrefix(unique+".", preload+".") {
				isUnique = false
				break
			}
		}
		if isUnique {
			uniquePreloads = append(uniquePreloads, preload)
		}
	}

	// sort them name then from short to long
	slices.SortStableFunc(uniquePreloads, func(i, j string) int {
		return strings.Compare(i, j)
	})
	return uniquePreloads
}

// GetPreloadMap returns a map of struct names to their respective preload fields, including nested relationships.
func GetPreloadMap(structs []StructMeta, preloadDepth uint) map[string][]string {
	preloads := make(map[string][]string)

	for _, st := range structs {
		preloadFields := getPreloadFieldsRecursive(st, structs, "", 0, int(preloadDepth))
		preloads[st.Name] = preloadFields
	}

	for key, fields := range preloads {
		preloads[key] = dedupePreloads(fields)
	}
	return preloads
}

func getPreloadFieldsRecursive(st StructMeta, allStructs []StructMeta, prefix string, depth int, maxDepth int) []string {
	preloadFields := make([]string, 0)

	for _, field := range st.Fields {
		if field.Preload {
			preloadFields = append(preloadFields, field.Name)

			for _, nestedStruct := range allStructs {
				if nestedStruct.Name == field.BaseType {
					if !strings.Contains(prefix, fmt.Sprintf(".%s.", field.Name)) {
						// Build the new prefix for the recursive call
						newPrefix := fmt.Sprintf("%s.%s.", prefix, field.Name)

						newDepth := depth + 1

						// Only call recursively if we haven't reached the max depth
						if newDepth <= maxDepth {
							nestedPreloadFields := getPreloadFieldsRecursive(nestedStruct, allStructs, newPrefix, newDepth, maxDepth)
							for _, nestedField := range nestedPreloadFields {
								// Append the nested preload with the current field name
								preloadFields = append(preloadFields, fmt.Sprintf("%s.%s", field.Name, nestedField))
							}
						}
					}
				}
			}
		}
	}

	return preloadFields
}
