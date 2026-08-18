# rawgen Package Documentation

## Overview

`rawgen` is a code generator that produces raw PostgreSQL Go database code for model structs. It generates type-safe CRUD (Create, Read, Update, Delete, Query) functions that use `database/sql` directly without any ORM overhead.

**Package Path:** `github.com/abiiranathan/apigen/rawgen`

## Purpose

The `rawgen` package provides a declarative way to generate efficient, low-level database access functions. Instead of manually writing SQL queries and row scanning logic, you define generation options and `rawgen` produces:

- `Insert<Model>()` - Create new records
- `Get<Model>()` - Retrieve single record by primary key
- `Update<Model>()` - Update record by primary key
- `Delete<Model>()` - Delete record by primary key
- `Query<Model>s()` - Query multiple records with optional filters

## Key Features

- **Zero ORM overhead** - Uses `database/sql` directly
- **Context-aware** - All functions accept `context.Context` for cancellation and timeouts
- **Flexible field selection** - Include or exclude specific columns
- **Custom WHERE clauses** - Define filters with nullable parameter support
- **Reserved keyword handling** - Automatically quotes PostgreSQL reserved words in identifiers
- **Automatic table naming** - Defaults to snake_case plural of model name
- **Foreign key awareness** - Excludes relation fields from SQL queries
- **Primary key handling** - Generates RETURNING clauses for auto-increment PKs
- **Comprehensive testing** - Extensive test coverage via `rawgen_test.go`

## Architecture

### Types

#### `Options`
Configures code generation behavior:

```go
type Options struct {
    ModelName    string   // Struct name to generate for (required, e.g. "User")
    ModelPkg     string   // Import path of models package
    SelectFields []string // Limit SELECT columns (empty = all non-relation fields)
    OmitFields   []string // Exclude columns from all queries
    Filters      []Filter // Custom WHERE clauses for Query function
    TableName    string   // Override default snake_case plural table name
}
```

#### `Filter`
Defines custom WHERE clause conditions in the Query function:

```go
type Filter struct {
    Column   string // Go field name (will be snake_cased for SQL)
    Op       string // SQL operator (=, >, <, ILIKE, IN, etc.)
    GoType   string // Go type for function parameter (string, int, *int, etc.)
    Nullable bool   // When true, wraps condition in nil-check for optional filters
}
```

### Internal Types

#### `column`
Resolved metadata for a single database column:
- `GoName` - Go field name (e.g., "Name")
- `ColName` - SQL identifier, quoted if reserved keyword (e.g., "name" or `"order"`)
- `GoType` - Go type from struct definition (e.g., "string")
- `IsFK` - True for foreign key/relation fields (excluded from queries)
- `IsPK` - True for primary key field ("ID")
- `BaseType` - Type with pointer/slice wrappers stripped

#### `templateData`
Aggregated data passed to code generation functions:
- Model and package information
- Resolved columns with metadata
- Primary key column reference
- Applied filters

## Public API

### Main Function

#### `Generate(w io.Writer, meta []parser.StructMeta, opts Options) error`

Generates raw PostgreSQL Go code and writes it to the provided writer.

**Parameters:**
- `w` - Output writer (file, buffer, etc.)
- `meta` - Parsed struct metadata from parser package
- `opts` - Generation options

**Returns:**
- Error if model not found, no columns after filtering, or write fails

**Example:**
```go
opts := rawgen.Options{
    ModelName: "User",
    ModelPkg:  "github.com/myapp/models",
}
if err := rawgen.Generate(os.Stdout, parsedMeta, opts); err != nil {
    log.Fatal(err)
}
```

## Code Generation Details

### Generated Output Structure

All generated code is placed in a `queries` package with standardized imports:
```go
package queries

import (
    "context"
    "database/sql"
    "fmt"
    "strings"        // Only if filters present
    
    "github.com/user/app/models"
)
```

### Insert Function

**Signature:** `func Insert<Model>(ctx context.Context, db *sql.DB, m *models.<Model>) error`

**Behavior:**
- Excludes primary key (assumes auto-increment)
- Uses `RETURNING id` to populate the generated ID
- For models without PK, uses `ExecContext` instead of `QueryRowContext`
- Accepts context for cancellation/timeout control

**Example Output (User model):**
```go
func InsertUser(ctx context.Context, db *sql.DB, u *models.User) error {
    const query = `INSERT INTO users (name, age, discount, role_id) VALUES ($1, $2, $3, $4) RETURNING id`
    return db.QueryRowContext(ctx, query,
        u.Name,
        u.Age,
        u.Discount,
        u.RoleID,
    ).Scan(&u.ID)
}
```

### Get Function

**Signature:** `func Get<Model>(ctx context.Context, db *sql.DB, id <PK_Type>) (*models.<Model>, error)`

**Behavior:**
- Retrieves single record by primary key
- Selects all non-relation columns
- Returns pointer to populated model or error

**Example Output:**
```go
func GetUser(ctx context.Context, db *sql.DB, id int) (*models.User, error) {
    const query = `SELECT id, name, age, discount, role_id FROM users WHERE id = $1`
    var u models.User
    if err := db.QueryRowContext(ctx, query, id).Scan(
        &u.ID, &u.Name, &u.Age, &u.Discount, &u.RoleID,
    ); err != nil {
        return nil, err
    }
    return &u, nil
}
```

### Query Function

**Signature:** `func Query<Model>s(ctx context.Context, db *sql.DB[, filter1 type1, ...]) ([]*models.<Model>, error)`

**Behavior:**
- Retrieves multiple records
- Supports zero or more filter parameters
- For nullable filters, only applies condition when parameter is non-nil
- Uses `strings.Join` to build WHERE clause dynamically
- Handles proper parameter indexing with `fmt.Sprintf`

**Example Output (with filters):**
```go
func QueryUsers(ctx context.Context, db *sql.DB, age int, name string) ([]*models.User, error) {
    baseQuery := `SELECT id, name, age, discount, role_id FROM users`
    
    conditions := make([]string, 0, 2)
    args := make([]any, 0, 2)
    paramIdx := 1

    conditions = append(conditions, fmt.Sprintf("age > $%d", paramIdx))
    args = append(args, age)
    paramIdx++

    conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", paramIdx))
    args = append(args, name)
    paramIdx++

    baseQuery += " WHERE " + strings.Join(conditions, " AND ")

    rows, err := db.QueryContext(ctx, baseQuery, args...)
    // ... row iteration and scanning
}
```

**Example Output (with nullable filter):**
```go
func QueryUsers(ctx context.Context, db *sql.DB, age *int) ([]*models.User, error) {
    baseQuery := `SELECT id, name, age, discount, role_id FROM users`
    
    conditions := make([]string, 0, 1)
    args := make([]any, 0, 1)
    paramIdx := 1

    if age != nil {
        conditions = append(conditions, fmt.Sprintf("age > $%d", paramIdx))
        args = append(args, *age)
        paramIdx++
    }
    
    if len(conditions) > 0 {
        baseQuery += " WHERE " + strings.Join(conditions, " AND ")
    }
    // ... rest of function
}
```

### Update Function

**Signature:** `func Update<Model>(ctx context.Context, db *sql.DB, m *models.<Model>) error`

**Behavior:**
- Updates all non-PK columns for a record
- Matches by primary key
- Returns `sql.ErrNoRows` if no row matched

**Example Output:**
```go
func UpdateUser(ctx context.Context, db *sql.DB, u *models.User) error {
    const query = `UPDATE users SET name = $1, age = $2, discount = $3, role_id = $4 WHERE id = $5`
    result, err := db.ExecContext(ctx, query,
        u.Name,
        u.Age,
        u.Discount,
        u.RoleID,
        u.ID,
    )
    if err != nil {
        return err
    }
    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rows == 0 {
        return sql.ErrNoRows
    }
    return nil
}
```

### Delete Function

**Signature:** `func Delete<Model>(ctx context.Context, db *sql.DB, id <PK_Type>) error`

**Behavior:**
- Deletes single record by primary key
- Returns `sql.ErrNoRows` if no row matched

**Example Output:**
```go
func DeleteUser(ctx context.Context, db *sql.DB, id int) error {
    const query = `DELETE FROM users WHERE id = $1`
    result, err := db.ExecContext(ctx, query, id)
    if err != nil {
        return err
    }
    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rows == 0 {
        return sql.ErrNoRows
    }
    return nil
}
```

## Column Resolution

### Field Filtering

Generated code includes only columns meeting ALL criteria:
1. Field is not marked as `Preload: true` (i.e., not a relation)
2. If `SelectFields` specified, field must be in the list
3. If `OmitFields` specified, field must NOT be in the list

### Identifier Formatting

Column identifiers are processed via `formatIdent()`:
1. **Reserved keywords** - Quoted if in PostgreSQL reserved words list
2. **Invalid characters** - Quoted if contains non-alphanumeric/underscore characters
3. **Normal identifiers** - Lowercased without quotes

**Examples:**
- `"order"` (reserved keyword)
- `user_name` (normal)
- `"user"` (reserved keyword)

## Usage Patterns

### Basic Generation (All Fields)

```go
opts := rawgen.Options{
    ModelName: "User",
    ModelPkg:  "github.com/myapp/models",
}
if err := rawgen.Generate(file, parsedMeta, opts); err != nil {
    log.Fatal(err)
}
```

### Select Specific Columns Only

```go
opts := rawgen.Options{
    ModelName:    "User",
    ModelPkg:     "github.com/myapp/models",
    SelectFields: []string{"ID", "Name", "Email"},
}
```

### Omit Sensitive Fields

```go
opts := rawgen.Options{
    ModelName:  "User",
    ModelPkg:   "github.com/myapp/models",
    OmitFields: []string{"PasswordHash", "RefreshToken"},
}
```

### Add Query Filters

```go
opts := rawgen.Options{
    ModelName: "User",
    ModelPkg:  "github.com/myapp/models",
    Filters: []rawgen.Filter{
        {Column: "Age", Op: ">", GoType: "int"},
        {Column: "Name", Op: "ILIKE", GoType: "string"},
    },
}
```

### Nullable Filters (Optional Parameters)

```go
opts := rawgen.Options{
    ModelName: "User",
    ModelPkg:  "github.com/myapp/models",
    Filters: []rawgen.Filter{
        {Column: "Age", Op: ">", GoType: "int", Nullable: true},
        {Column: "Status", Op: "=", GoType: "string", Nullable: true},
    },
}
// Generates function:
// func QueryUsers(ctx context.Context, db *sql.DB, age *int, status *string)
// Filter only applied if parameter is non-nil
```

### Custom Table Name

```go
opts := rawgen.Options{
    ModelName: "User",
    ModelPkg:  "github.com/myapp/models",
    TableName: "app_users", // Instead of default "users"
}
```

## CLI Tool Usage

The `rawgen` CLI in `cmd/rawgen/main.go` provides command-line access to code generation:

### Basic Usage

```bash
rawgen -model User
```

### Common Flags

- `-config string` - Path to apigen.toml (default: "apigen.toml")
- `-model string` - Model struct name (required)
- `-table string` - Override table name
- `-select string` - Comma-separated field names to include
- `-omit string` - Comma-separated field names to exclude
- `-filter string` - Custom filters as "Column:Op:GoType" (repeatable)
- `-nullable-filter string` - Same as -filter but with nullable parameters

### Examples

```bash
# Generate all fields for User model
rawgen -model User

# Select specific fields only
rawgen -model User -select "ID,Name,Email"

# Exclude fields
rawgen -model User -omit "Discount"

# Add filters
rawgen -model User -filter "Age:>:int,Name:ILIKE:string"

# Nullable filters (optional parameters)
rawgen -model User -nullable-filter "Age:>:int"

# Output to file
rawgen -model User > queries/user.go && gofmt -w queries/user.go

# Custom table name
rawgen -model User -table "app_users"
```

## Testing

### Test Coverage

The `rawgen_test.go` file provides comprehensive test coverage:

#### Basic Tests
- `TestBasicGeneration` - All generated functions and SQL statements
- `TestRelationFieldsExcluded` - Foreign key fields excluded from SQL

#### Field Selection Tests
- `TestSelectFields` - Only selected fields included
- `TestOmitFields` - Specified fields excluded
- `TestOmitMultipleFields` - Multiple fields excluded correctly

#### Advanced Tests
- `TestCustomTableName` - Table name override works
- `TestReservedKeywordsAreQuoted` - PostgreSQL keywords properly quoted
- `TestRequiredFilters` - Multiple required filter parameters
- `TestSingleFilter` - Single filter parameter
- `TestNullableFilters` - Nullable filter with nil-check
- `TestMixedFilters` - Mix of required and nullable filters
- `TestModelNotFound` - Error handling for non-existent model
- `TestModelWithoutPK` - Models without primary keys still generate Insert/Query

### Running Tests

```bash
go test ./rawgen -v
```

### Test Data

Tests use a consistent test metadata structure with sample models:
- `User` (int PK) - User model with roles and tags
- `Role` (int64 PK) - Simple role model
- `Issue` (int64 PK) - Simple issue model
- `Order` - Tests PostgreSQL reserved keywords

## Implementation Details

### Helper Functions

#### `findStruct(meta []parser.StructMeta, name string) (*parser.StructMeta, error)`
Locates a struct in parsed metadata by name.

#### `receiverIdent(name string) string`
Generates a single-character receiver identifier from model name (e.g., "User" → "u").

#### `formatIdent(ident string) string`
Quotes SQL identifiers if they are reserved keywords or contain invalid characters.

#### `resolveColumns(st *parser.StructMeta, opts Options) []column`
Builds column metadata list, filtering by SelectFields/OmitFields/relations.

#### `findPK(cols []column) *column`
Returns pointer to primary key column or nil.

#### `writeCode(w io.Writer, d templateData) error`
Orchestrates code generation by calling individual write functions.

#### Code Writing Functions
- `writeHeader()` - Package declaration and imports
- `writeInsert()` - Insert function
- `writeGet()` - Get by PK function
- `writeQuery()` - Query with filters function
- `writeDelete()` - Delete by PK function
- `writeUpdate()` - Update function

#### `columnList(cols []column) string`
Formats column names into CSV for SELECT statements.

#### `scanFieldList(cols []column, ident string) string`
Formats column references for row.Scan() arguments.

#### `nonPKCols(cols []column) []column`
Filters out primary key column.

## PostgreSQL Reserved Keywords

The package maintains a comprehensive map of PostgreSQL reserved keywords (`postgresReserved`). Identifiers matching any keyword are automatically quoted with double quotes.

**Examples:** all, analyze, and, any, array, as, asc, false, foreign, from, group, order, select, table, true, user, where, with

## Error Handling

The Generate function returns errors in these cases:

1. **Empty ModelName** - `"rawgen: Options.ModelName must not be empty"`
2. **Model Not Found** - `"rawgen: model "X" not found in parsed structs"`
3. **No Columns After Filtering** - `"rawgen: model "X" has no columns after applying SelectFields/OmitFields"`
4. **Write Error** - `"rawgen: failed to write generated code: [underlying error]"`

## Notes for Users

- **Import Paths** - Generated code always imports the specified ModelPkg; ensure it's the correct import path
- **Receiver Names** - Single-letter receivers follow Go conventions (u for User, r for Role, etc.)
- **Context Usage** - All generated functions accept context for timeouts and cancellation
- **SQL Escaping** - Use parameterized queries with `$N` placeholders; never interpolate user input
- **Error Types** - Functions return `sql.ErrNoRows` when records not found via Get/Delete/Update
- **Relation Handling** - Foreign key fields (marked with Preload) are automatically excluded; retrieve relations separately
- **Package Name** - Generated code always uses `queries` as the package name

## Dependencies

- `"fmt"` - String formatting
- `"io"` - Writer interface
- `"slices"` - Go 1.22+ slice utilities
- `"strings"` - String manipulation
- `"unicode"`, `"unicode/utf8"` - Character analysis
- `"github.com/abiiranathan/apigen/parser"` - Struct metadata
- `"github.com/iancoleman/strcase"` - Case conversion
