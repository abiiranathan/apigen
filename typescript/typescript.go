package typescript

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/abiiranathan/apigen/config"
	"github.com/abiiranathan/apigen/parser"
)

// regex pattern to extract the struct json tag.
var pattern = regexp.MustCompile(`json:"(.*?)"`)

// returns the field name of the struct field name given it's tag.
// If the tag does not match the expected json:"name", returns ""
func getJSONFieldName(tag string) string {
	matches := pattern.FindStringSubmatch(tag)

	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Helper recursive method to generate the typescript types.
// If recursive, do not recreate the override types.
func generateInterfaces(
	output io.Writer,
	inputs map[string]parser.StructMeta,
	overrides config.Overrides,
	recursive bool,
	generated map[string]bool,
) {
	// Code to generate only once.
	if !recursive {
		// Create custom override types
		for key, value := range overrides.Types {
			fmt.Fprintf(output, "type %s = %s\n\n", key, value)
		}
	}

	for _, input := range inputs {
		// skip structs with empty fields
		if len(input.Fields) == 0 {
			continue
		}

		// Check if already generated
		if _, exists := generated[input.Name]; exists {
			continue
		}

		// Mark as generated here before calling recursive functions below
		// Otherwise, produces infinite recursion and panics.
		generated[input.Name] = true

		builder := strings.Builder{}
		builder.WriteString(`interface `)
		builder.WriteString(input.Name)
		builder.WriteString(" {\n")

		for _, f := range input.Fields {
			fieldName := getJSONFieldName(f.Tag)

			// Fields with tag name - are skipped by json encoder
			if fieldName == "-" {
				continue
			}

			// If json tag is missing, use exact FieldName
			if fieldName == "" {
				fieldName = f.Name
			}

			// Add field to interface
			builder.WriteRune('\t')
			builder.WriteString(fieldName)

			// Check if there is an override for this field
			if overrideType, ok := overrides.Fields[fieldName]; ok {
				builder.WriteString(": ")
				builder.WriteString(overrideType + ";\n")
				continue
			}

			// Ignore pointer
			f.Type = strings.TrimPrefix(f.Type, "*")

			if strings.HasPrefix(f.Type, "[]") || strings.HasPrefix(f.Type, "[") {
				// Get the element type of the slice or array
				elementType := f.Type[strings.IndexByte(f.Type, ']')+1:]

				if input, ok := inputs[elementType]; ok {
					generateInterfaces(output,
						map[string]parser.StructMeta{input.Name: input}, overrides, true, generated)
					builder.WriteString(": ")
					builder.WriteString(input.Name)
					builder.WriteString("[];")
				} else {
					builder.WriteString(": ")
					writeSliceType(&builder, elementType)
				}
			} else {
				// Check if f.Type references a struct in the inputs map
				if structMeta, ok := inputs[f.Type]; ok {
					// Recursively generate interface for referenced struct
					generateInterfaces(output,
						map[string]parser.StructMeta{f.Type: structMeta}, overrides, true, generated)
					builder.WriteString(": ")
					builder.WriteString(structMeta.Name)
					builder.WriteString(";")
				} else {
					writeFieldType(&builder, f.Type)
				}
			}
			builder.WriteString("\n")
		}
		builder.WriteString("}\n\n")
		_, _ = output.Write([]byte(builder.String()))
	}
}

// Generate typescript interfaces given the map of parser StructMeta.
// Generated code is written to w.
func GenerateTypescriptInterfaces(
	output io.Writer,
	inputs map[string]parser.StructMeta,
	overrides config.Overrides,
) {
	generated := make(map[string]bool)
	generateInterfaces(output, inputs, overrides, false, generated)
}

// Write typescript field type based on fieldType to builder.
// Derived types can not be inferred, in which case it return writeFieldType unmodified.
//
//	e.g type Sex string
//
// When a struct is created with field Sex, we are unable to infer underlying type as string.
// Feel free to submit a pull request addressing this issue.
func writeFieldType(builder *strings.Builder, fieldType string) {
	switch fieldType {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		builder.WriteString(": number;")
	case "string":
		builder.WriteString(": string;")
	case "bool":
		builder.WriteString(": boolean;")
	case "time.Time":
		builder.WriteString(": string;")
	default:
		builder.WriteString(": ")
		builder.WriteString(fieldType)
	}
}

// Writes the slice type by switching on the elementType.
func writeSliceType(builder *strings.Builder, elemType string) {
	switch elemType {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		builder.WriteString("number[];")
	case "string":
		builder.WriteString("string[];")
	case "bool":
		builder.WriteString("boolean[];")
	default:
		builder.WriteString(elemType)
		builder.WriteString("[];")
	}
}
