package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/abiiranathan/apigen/config"
)

// dedupePreloads removes redundant preloads where a prefix is a subset of another preload.
func dedupePreloads(preloads []string) []string {
	if len(preloads) == 0 {
		return preloads
	}

	uniquePreloads := make([]string, 0, len(preloads))

	for _, preload := range preloads {
		toDelete := []int{}
		isUnique := true

		// Check against existing uniquePreloads
		for i, unique := range uniquePreloads {
			if strings.HasPrefix(preload+".", unique+".") {
				// Mark existing for deletion
				toDelete = append(toDelete, i)
			} else if strings.HasPrefix(unique+".", preload+".") {
				isUnique = false
				break
			}
		}

		// Delete marked entries in reverse order to preserve indices
		slices.Reverse(toDelete)
		for _, i := range toDelete {
			uniquePreloads = slices.Delete(uniquePreloads, i, i+1)
		}

		if isUnique {
			uniquePreloads = append(uniquePreloads, preload)
		}
	}

	slices.Sort(uniquePreloads)

	return uniquePreloads
}

func writeJSONToFile(filename string, data map[string][]string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(data)
}

// GetPreloadMap returns a map of struct names to their respective preload fields, including nested relationships.
func GetPreloadMap(structs []StructMeta, cfg *config.Config) map[string][]string {
	preloads := make(map[string][]string)
	preloadDepth := cfg.PreloadDepth

	for _, st := range structs {
		preloadFields := getPreloadFieldsRecursive(st, structs, "", 0, int(preloadDepth))
		if _, ok := preloads[st.Name]; ok {
			// Merge existing preloads with new ones
			preloadFields = append(preloadFields, preloads[st.Name]...)
		}
		preloads[st.Name] = preloadFields
	}

	for key, fields := range preloads {
		preloads[key] = dedupePreloads(fields)
	}

	if cfg.OutputJson {
		outpath := "preload.json"
		if err := writeJSONToFile(outpath, preloads); err != nil {
			fmt.Printf("Error writing to JSON file: %v\n", err)
		} else {
			fmt.Printf("Preload JSON file created at %s\n", outpath)
		}
	}
	return preloads
}

func getPreloadFieldsRecursive(st StructMeta, allStructs []StructMeta, prefix string, depth int, maxDepth int) []string {
	preloadFields := make([]string, 0, len(st.Fields))

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
