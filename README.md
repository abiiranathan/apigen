# apigen

`apigen` is a tool written in Go that parses Go structs and generates code for querying the database based on [GORM](https://gorm.io) ORM. It also includes `rawgen`, a companion tool that generates raw `database/sql` PostgreSQL code for when you need low-level control and maximum performance.

## Features

- Generates GORM services for all structs in your models unless skipped in the configuration file (`apigen.toml`)
- Created database connection helper with sane defaults
- Optionally preloads all relationships (even nested relationships) by default. Because the parser knows foreign keys and the tree, we are able to do that for all `foreignKey` and `many2many` fields
- Allows for customizing all queries by specifying optional Where, ordering, grouping, select `options ...services.Options`. These options are passed to the callable handlers that are designed with the decorator pattern
- Generates typescript interfaces for your models
- **`rawgen`** — generates raw PostgreSQL Go functions (`database/sql`) for Insert, Get, Delete, Update, and Query with full control over selected fields, omitted fields, custom filters, and table names

## Performance Tuning

`apigen` includes configuration options to reduce query overhead in the generated GORM services:

```toml
# Set to true to disable automatic preloading — callers opt in per-query
# via .PreloadAll(true) or the Preload() option
LazyPreload = false

# Set to false to skip re-fetching records after Create/Update
# Callers can call .Get(id) explicitly when they need associations loaded
RefetchAfterWrite = true
```

The generated services also include generic raw SQL helpers for escape-hatch queries:

```go
// For complex queries, reports, aggregates, CTEs
results, err := services.RawQuery[MyStruct](db, "SELECT * FROM users WHERE age > $1", 18)
row, err := services.RawQueryRow[MyStruct](db, "SELECT * FROM users WHERE id = $1", 1)
```

## Installation

### apigen CLI

```console
go install github.com/abiiranathan/apigen@latest
```

### rawgen CLI (separate binary)

```console
go install github.com/abiiranathan/apigen/cmd/rawgen@latest
```

### As a library

```console
go get -u github.com/abiiranathan/apigen
```

#### Initialize project and create a configuration file `apigen.toml`

```console
apigen init
```

The configuration file has the following format:

```toml
# apigen configuration file.
# Strings should use single quotes

# PreloadAll all relationships with gorm.Preload
PreloadAll = false

# PreloadDepth is the depth of the relationships to preload
# e.g Patient.Visit.Doctor will be preloaded if PreloadDepth is 3
PreloadDepth = 3

# Output preload.json
OutputJson = true

[Models]
# ModelPkg is the package name for the models to look for struct definitions
Pkgs = [
  'github.com/abiiranathan/apigen/models',
]

# Skip is a list of models to skip. Names are case sensitive
Skip = []

# ReadOnly is a list of models that should not be generated with CRUD operations
# Only Get methods will be generated for these models.
ReadOnly = []

[Output]

# OutDir is the directory to write the generated files to
OutDir = 'gen'

# ServiceName is the name of the service to generate
ServiceName = 'services'

[overrides]
# Overrides for types and fields. Forexample, here we are overriding the type
# as would appear in typescript for Sex to be an enum.
[overrides.types]
Sex = '"Male" | "Female"'

# Here we are overriding gener to of type Sex. Sex is a go type.
[overrides.fields]
gender = 'Sex'
```

## Generate code

If apigen.toml is the root of your project.

```bash
apigen generate
```

Or

```bash
apigen --config config.toml generate
```

to specify the custom path to the configuration file.

### Using the generated code

```go

import(
    // These will point to the generated code
    "github.com/yourname/cool-api/gen/handlers"
	"github.com/yourname/cool-api/gen/services"

	"gorm.io/gorm/logger"
)

func main() {
	db, err := services.PostgresConnection(dsn, "Africa/kampala", logger.Silent, os.Stdout)
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("Error performing database migrations: %v\n", err)
	}

	svc := services.NewService(db)
    users, err := svc.Users.GetAll()
}
```

## rawgen — Raw PostgreSQL Code Generator

`rawgen` generates type-safe `database/sql` Go functions for a given model. It reads your `apigen.toml` to find model packages and outputs Go code to stdout. Use it when you need low-level control over SQL without GORM overhead.

### Install

```bash
go install github.com/abiiranathan/apigen/cmd/rawgen@latest
```

### Usage

```
rawgen [flags]

Flags:
  -config string          Path to apigen.toml (default "apigen.toml")
  -model string           Model struct name (required, e.g. "User")
  -table string           Override table name (default: snake_case plural of model)
  -select string          Comma-separated field names to include (default: all)
  -omit string            Comma-separated field names to exclude
  -filter string          Custom filters as "Column:Op:GoType" (comma-separated)
  -nullable-filter string Same as -filter but wraps in nil check (pointer param)
```

### Examples

**Basic — all fields, no filters:**

```bash
go run ./cmd/rawgen -model User > queries/user.go
```

Generated output:

```go
func InsertUser(ctx context.Context, db *sql.DB, u *models.User) error { ... }
func GetUser(ctx context.Context, db *sql.DB, id int) (*models.User, error) { ... }
func QueryUsers(ctx context.Context, db *sql.DB) ([]*models.User, error) { ... }
func UpdateUser(ctx context.Context, db *sql.DB, u *models.User) error { ... }
func DeleteUser(ctx context.Context, db *sql.DB, id int) error { ... }
```

**Select specific fields:**

```bash
go run ./cmd/rawgen -model User -select "ID,Name,Age"
```

Only `id`, `name`, `age` columns appear in all generated SQL — no `discount`, no `role_id`.

**Omit fields:**

```bash
go run ./cmd/rawgen -model User -omit "Discount"
```

All columns except `discount` are included.

**With required filters (typed function params):**

```bash
go run ./cmd/rawgen -model User -filter "Age:>:int,Name:ILIKE:string"
```

Generates `QueryUsers` with typed parameters that build a dynamic WHERE clause:

```go
func QueryUsers(ctx context.Context, db *sql.DB, age int, name string) ([]*models.User, error) {
    // ... WHERE age > $1 AND name ILIKE $2
}
```

**With optional/nullable filters (pointer params, nil-checked):**

```bash
go run ./cmd/rawgen -model User -nullable-filter "Age:>:int"
```

Generates `QueryUsers` with pointer parameters — filters are only applied when non-nil:

```go
func QueryUsers(ctx context.Context, db *sql.DB, age *int) ([]*models.User, error) {
    // if age != nil { ... WHERE age > $1 }
}
```

**Custom table name:**

```bash
go run ./cmd/rawgen -model User -table my_users
```

All generated SQL uses `my_users` instead of the default `users`.

**Pipe to a file and format:**

```bash
go run ./cmd/rawgen -model User -filter "Age:>:int" > queries/user.go && gofmt -w queries/user.go
```
