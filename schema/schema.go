package schema

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/abiiranathan/apigen/enums"
	"github.com/iancoleman/strcase"
)

var tagName = "sql"

// Set custom struct tag for database types. default is "sql"
func SetTagName(tag string) {
	tagName = tag
}

// Parse struct tag and extract field types, constraints and override types.
// tag "-" indicates that this field should not be created in the database.
func parseStructTags(t reflect.StructTag) (map[string]string, bool) {
	sqlTag, ok := t.Lookup(tagName)
	sqlRules := make(map[string]string)

	// No sql tag, use defaults
	if !ok {
		return sqlRules, true
	}

	// Do not create this field in the database
	if sqlTag == "-" {
		return nil, false
	}

	rules := strings.Split(sqlTag, ";")
	for _, rule := range rules {
		trimmed := strings.TrimSpace(rule)
		if trimmed != "" {
			keyValue := strings.Split(trimmed, ":")
			var key, value string
			if len(keyValue) == 2 {
				key = strings.TrimSpace(keyValue[0])
				value = strings.TrimSpace(keyValue[1])
			} else {
				key = strings.TrimSpace(keyValue[0])
				value = strings.TrimSpace(keyValue[0])
			}
			sqlRules[key] = value
		}
	}
	return sqlRules, true
}

// Keep track of tables whose schema has already been generated.
var tablesProccessed = make(map[string]string)

func getSQLType(goType string) string {
	switch goType {
	case "int":
		return "integer"
	case "uint":
		return "serial"
	case "int8":
		return "smallint"
	case "int16":
		return "smallint"
	case "int32":
		return "integer"
	case "int64":
		return "bigint"
	case "float32":
		return "real"
	case "float64":
		return "double precision"
	case "bool":
		return "boolean"
	case "time.Time":
		return "timestamptz"
	case "string":
		return "text"
	default:
		return ""
	}
}

func mapSlice[T any](s []T, predicate func(val T) T) []T {
	arr := make([]T, len(s))
	for index, val := range s {
		arr[index] = predicate(val)
	}
	return arr
}

func mergeTableMaps(m1 TableMap, m2 TableMap) TableMap {
	merged := make(TableMap)
	for k, v := range m1 {
		merged[k] = v
	}
	for k, v := range m2 {
		merged[k] = v
	}
	return merged
}

func generateSQL(model interface{}) (schemas []string, tableMap TableMap) {
	var fields []string
	var exactFieldNames []Field
	var primaryKeys []string
	var uniqueKeys []string
	var constraints []string
	var foreignKeys []string
	var foreignKeyDetails []ForeignKey
	var generatedTables []string
	var pk string
	var defaultValue string

	enumsMap := make(map[string][]string)
	tableMap = make(TableMap)

	v := reflect.ValueOf(model).Type()
	tableName := strcase.ToSnake(v.Name())

	// Pluralize if not already ending in s
	if !strings.HasSuffix(tableName, "s") {
		tableName += "s"
	}

	// Table already proccessed
	if _, ok := tablesProccessed[tableName]; ok {
		return []string{}, tableMap
	}

	// Iterate through all struct fields
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sqlRules, ok := parseStructTags(field.Tag)
		// Skipped fields (-)
		if !ok && sqlRules == nil {
			continue
		}

		fieldName := strcase.ToSnake(field.Name)
		fieldNullable := false

		goType := field.Type.String()
		dataType := getSQLType(goType)
		var fieldDefBuilder strings.Builder
		fieldDef := io.StringWriter(&fieldDefBuilder)
		var typeOverridden bool

		initFieldDef := func() {
			fieldDefBuilder.Reset()
			fieldDef.WriteString(fieldName)
			fieldDef.WriteString(" ")

			// Override type if custom type is specified
			if overrideType, ok := sqlRules["type"]; ok {
				dataType = overrideType
				typeOverridden = true
			}

			fieldDef.WriteString(dataType)

			// Add null constraint if null specified in tag, otherwise it's
			// NOT NULL by default.
			if _, ok := sqlRules["null"]; ok {
				fieldDef.WriteString(" NULL")
				fieldNullable = true
			} else {
				fieldDef.WriteString(" NOT NULL")
				fieldNullable = false
			}
		}

		initFieldDef()

		// Custom types
		if dataType == "" {
			// Check if column implements the enumer interfaces
			if enums.ImplementsEnumer(field) {
				structValue := reflect.New(field.Type).Elem()
				enumer := structValue.Addr().Interface().(enums.Enumer)
				dataType = enumer.DatabaseType()
				enumsMap[fieldName] = enumer.ValidValues()
				initFieldDef()
			} else {
				panic(fmt.Sprintf("Unknown type %s for field %s", goType, fieldName))
			}
		}

		// Flag to indicate whether to skip the field definition
		// For foreign keys and many to many fields.
		skipDef := false
		for rule, value := range sqlRules {
			switch {
			case rule == "primaryKey" || rule == "autoIncrement":
				primaryKeys = append(primaryKeys, field.Name)

				// Check for autoIncrement
				if _, ok := sqlRules["autoIncrement"]; ok && !typeOverridden {
					if dataType == "bigint" {
						dataType = "bigserial"
					} else {
						dataType = "serial"
					}

					initFieldDef()
				}
			case rule == "unique":
				uniqueKeys = append(uniqueKeys, field.Name)
			case rule == "default":
				fieldDef.WriteString(fmt.Sprintf(" DEFAULT %s", value))
				defaultValue = value
			case rule == "check":
				fieldDef.WriteString(fmt.Sprintf(" CHECK %s", value))
			case rule == "constraint":
				fieldDef.WriteString(fmt.Sprintf(" CONSTRAINT %s", value))
			case rule == "index":
				fields = append(fields, fmt.Sprintf("INDEX(%s)", strcase.ToSnake(field.Name)))
			case rule == "uniqueIndex":
				if value == "uniqueIndex" {
					value = fmt.Sprintf("unique_%s_%s", strcase.ToSnake(v.Name()), strcase.ToSnake(field.Name))
				}
				constraints = append(constraints,
					fmt.Sprintf("CONSTRAINT %s UNIQUE (%s)", value, strcase.ToSnake(field.Name)),
				)

			case rule == "foreignKey":
				keyParts := strings.Split(value, ".")
				if len(keyParts) != 2 {
					panic("Foreign key declaration must be in format 'TableName.FieldName'")
				}

				foreignTable := strcase.ToSnake(keyParts[0])
				foreignField := strcase.ToSnake(keyParts[1])
				fkDef := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s)", strcase.ToSnake(field.Name), foreignTable, foreignField)

				foreignKeyDetails = append(foreignKeyDetails, ForeignKey{
					Name:                strcase.ToSnake(field.Name),
					ReferencedTable:     foreignTable,
					ReferencedFieldName: foreignField,
				})

				if onDelete, ok := sqlRules["onDelete"]; ok {
					switch strings.ToLower(onDelete) {
					case "cascade":
						fkDef += " ON DELETE CASCADE"
					case "set_null":
						fkDef += " ON DELETE SET NULL"
					case "set_default":
						fkDef += " ON DELETE SET DEFAULT"
					case "restrict":
						fkDef += " ON DELETE RESTRICT"
					default:
						panic("unknown constraint" + onDelete)
					}
				}

				if onUpdate, ok := sqlRules["onUpdate"]; ok {
					switch strings.ToLower(onUpdate) {
					case "cascade":
						fkDef += " ON UPDATE CASCADE"
					case "set_null":
						fkDef += " ON UPDATE SET NULL"
					case "set_default":
						fkDef += " ON UPDATE SET DEFAULT"
					case "restrict":
						fkDef += " ON UPDATE RESTRICT"
					default:
						panic("unknown constraint" + onUpdate)
					}
				}
				foreignKeys = append(foreignKeys, fkDef)
			case rule == "many2many":
				// Create intermediate table and then join table
				skipDef = true
				instance := reflect.New(field.Type.Elem()).Elem().Interface()
				newSchema, newMap := generateSQL(instance)
				generatedTables = append(generatedTables, newSchema...)
				tableMap = mergeTableMaps(tableMap, newMap)

				// Create join table
				joinTable := fmt.Sprintf("%s_%s", tableName, fieldName)
				singularTableName := strings.TrimSuffix(tableName, "s")
				singularFieldName := strings.TrimSuffix(strcase.ToSnake(fieldName), "s")

				tableFields := []string{}
				col1 := fmt.Sprintf("%s_id bigint NOT NULL", singularTableName)
				col2 := fmt.Sprintf("%s_id bigint NOT NULL", singularFieldName)

				colName1 := fmt.Sprintf("%s_id", singularTableName)
				colName2 := fmt.Sprintf("%s_id", singularFieldName)

				tableFields = append(tableFields, col1)
				tableFields = append(tableFields, col2)

				// Create the Join table.
				tableStmt := fmt.Sprintf(
					`CREATE TABLE IF NOT EXISTS %s (%s , PRIMARY KEY(%s_id, %s_id), FOREIGN KEY(%s) REFERENCES %s(id), FOREIGN KEY(%s) REFERENCES %s(id));
					`,
					joinTable, strings.Join(tableFields, ", "),
					singularTableName, singularFieldName,
					colName1, tableName,
					colName2, strcase.ToSnake(fieldName),
				)
				generatedTables = append(generatedTables, tableStmt)
			case rule == "ref":
				skipDef = true
				fieldValue := reflect.New(field.Type).Elem().Interface()
				newSchema, newMap := generateSQL(fieldValue)
				generatedTables = append(generatedTables, newSchema...)
				tableMap = mergeTableMaps(tableMap, newMap)
			}
		}

		if !skipDef {
			exactFieldNames = append(exactFieldNames, Field{
				Name:         field.Name,
				ColumnName:   fieldName,
				DType:        dataType,
				Nullable:     fieldNullable,
				DefaultValue: defaultValue,
			})
			fields = append(fields, fieldDefBuilder.String())
		}
	}

	// Init sql string builder
	var sql strings.Builder

	// Create custom types(enums) if any
	for key, values := range enumsMap {
		sql.WriteString(fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);\n", key, strings.Join(values, ", ")))
	}

	sql.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", tableName))
	sql.WriteString(strings.Join(fields, ", "))

	mapToSnakeCase := func(s []string) []string { return mapSlice(s, strcase.ToSnake) }

	if len(primaryKeys) > 0 {
		sql.WriteString(
			fmt.Sprintf(", PRIMARY KEY(%s)", strings.Join(mapToSnakeCase(primaryKeys), ", ")),
		)
		pk = primaryKeys[0]
	} else {
		// No primary key explicitly specified, consider the ID field
		for _, f := range exactFieldNames {
			if f.ColumnName == "id" {
				sql.WriteString(", PRIMARY KEY(id)")
				pk = "id"
				break
			}
		}

	}

	if len(uniqueKeys) > 0 {
		sql.WriteString(fmt.Sprintf(", UNIQUE(%s)", strings.Join(mapToSnakeCase(uniqueKeys), ", ")))
	}

	if len(constraints) > 0 {
		sql.WriteString(fmt.Sprintf(", %s", strings.Join(constraints, ", ")))
	}

	if len(foreignKeys) > 0 {
		sql.WriteString(fmt.Sprintf(", %s", strings.Join(foreignKeys, ", ")))
	}

	sql.WriteString(");")

	generatedTables = append(generatedTables, sql.String())
	tablesProccessed[tableName] = tableName

	// Create a tables map data structure

	table := Table{
		TableName:   tableName,
		Model:       model,
		PrimaryKey:  pk,
		ForeignKeys: foreignKeyDetails,
		Fields:      exactFieldNames,
	}

	tableMap[v.Name()] = table
	for i := range exactFieldNames {
		tableMap[v.Name()].Fields[i].Table = &table
	}
	return generatedTables, tableMap
}

/*
we use (\w+(?:,\s*\w+){1,}) to match one or more word characters
followed by one or more comma-separated lists of word characters
optionally separated by whitespace, and capture the entire list of
primary key columns as a single group.
The {1,} quantifier requires at least one repetition of the comma-separated list,
effectively requiring at least two primary key columns.
*/
var pattern = `PRIMARY KEY\s*\((\w+(?:,\s*\w+){1,})\)`
var compositePKRegex = regexp.MustCompile(pattern)

// Sort schema based on which tables should be created first.
// Tables with foreignKey relationships must be created last.
// Join tables must be created after primary tables.
func sortSchema(model []string) {
	sort.SliceStable(model, func(i, j int) bool {
		iHasFK := strings.Contains(model[i], "FOREIGN KEY")
		jHasFK := strings.Contains(model[j], "FOREIGN KEY")

		hasCompPK := compositePKRegex.MatchString(model[i])
		hasCompPK2 := compositePKRegex.MatchString(model[j])
		// sort by tables with no foreign keys first.
		// Then sort tables so that tables with composite primary keys
		// come last. This avoid referencing tables that do not exist,
		// reducing database errors while doing migrations.
		return (!iHasFK && jHasFK) || (!hasCompPK && hasCompPK2)
	})
}

// Parse the models(structs) and generate sql statements to create the tables.
// Returns the table create statements and a list of table names.
// Schema is sorted(as best as possible) to avoid errors during migrations.
func GenerateSchema(models []any) (schemas []string, tableMap TableMap, table_names []string) {
	var allSchema []string
	tableMap = make(TableMap)

	for _, table := range models {
		schemas, newMap := generateSQL(table)
		allSchema = append(allSchema, schemas...)
		tableMap = mergeTableMaps(tableMap, newMap)
	}

	// Sort the schemas
	sortSchema(allSchema)
	for t := range tablesProccessed {
		table_names = append(table_names, t)
	}

	return allSchema, tableMap, table_names
}
