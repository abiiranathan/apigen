package schema

// Represents a database field.
type Field struct {
	Name         string
	ColumnName   string
	DType        string
	Nullable     bool
	DefaultValue string
	Table        *Table
}

// Represents a foreign key
type ForeignKey struct {
	Name                string
	ReferencedTable     string
	ReferencedFieldName string
}

// Represents
type Table struct {
	Model       any
	TableName   string
	PrimaryKey  string
	Fields      []Field
	ForeignKeys []ForeignKey
}

type TableMap map[string]Table
