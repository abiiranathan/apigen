# rawgen

`rawgen` is a high-performance code generator for Go that produces type-safe, boilerplate-free PostgreSQL database code from your model structs. It runs directly on Go's standard `database/sql` package without any ORM overhead or runtime reflection.

You can think of it as a simple opposite of `sqlc` for when you want simple CRUD queries for your models.
---

## Key Advantages

* **Zero ORM Overhead**: Generates idiomatic, direct `database/sql` queries and scan loops with maximum execution speed and zero reflection.
* **Unified DB & Transaction Support (`DBTX`)**: Every generated function accepts the `DBTX` interface. You can pass either a `*sql.DB` connection pool or a `*sql.Tx` transaction without changing code.
* **Compile-Time Constant Queries**: Fully optimizes static and fixed-filter queries into Go compile-time `const` strings, eliminating runtime string formatting and slice allocations.
* **Smart / Selective Quoting**: Automatically quotes only PostgreSQL reserved keywords (like `"order"`, `"user"`), preserving clean, readable SQL for standard identifiers.
* **Context-Aware**: All operations accept `context.Context` for cancellation, timeouts, and distributed tracing.
* **Slice Preallocation (`EstimatedRows`)**: Optional slice capacity hints eliminate dynamic slice reallocations on list queries.
* **Clean Separation of Concerns**: Relation fields (`many2many`, `foreignKey`) are automatically detected and omitted from raw table mutations.

---

## Installation

Install the CLI tool using `go install`:

```bash
go install github.com/abiiranathan/apigen/cmd/rawgen@latest
```

Or import the generator directly in your Go code:

```go
import "github.com/abiiranathan/apigen/rawgen"
```

---

## Structure of Generated Code

When you generate code for a model (e.g. `User`), `rawgen` creates standard Go functions in a target package (default `queries`):

### 1. The `DBTX` Interface (`dbtx.go`)
Generated once per output package to enable seamless transactions and connection pooling:

```go
type DBTX interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
```

### 2. CRUD Operations (`<model>.go`)

| Function   | Signature                              | Description                                                               |
| ---------- | -------------------------------------- | ------------------------------------------------------------------------- |
| **Insert** | `Insert<Model>(ctx, db, record)`       | Inserts non-PK fields and populates generated ID via `RETURNING id`.      |
| **Get**    | `Get<Model>(ctx, db, id)`              | Selects a single record by primary key (`WHERE id = $1`).                 |
| **Update** | `Update<Model>(ctx, db, record)`       | Updates all columns by primary key. Returns `sql.ErrNoRows` if not found. |
| **Delete** | `Delete<Model>(ctx, db, id)`           | Deletes a record by primary key. Returns `sql.ErrNoRows` if not found.    |
| **Query**  | `Query<Model>s(ctx, db, [filters...])` | Retrieves a slice of records matching compile-time or dynamic filters.    |

---

## CLI Usage

### Basic Usage

Generate CRUD operations for a `User` struct defined in your models package:

```bash
rawgen -model User > queries/user.go
```

Generate the shared `dbtx.go` interface:

```bash
rawgen -dbtx > queries/dbtx.go
```

### Common Flags

| Flag               | Type     | Description                                                                         |
| ------------------ | -------- | ----------------------------------------------------------------------------------- |
| `-model`           | `string` | Struct name to generate for (**required**, e.g. `User`)                             |
| `-config`          | `string` | Path to configuration file (default: `apigen.toml`)                                 |
| `-table`           | `string` | Override default table name (defaults to snake_case plural)                         |
| `-select`          | `string` | Comma-separated list of fields to include exclusively                               |
| `-omit`            | `string` | Comma-separated list of fields to exclude from queries                              |
| `-filter`          | `string` | Required filters as `Column:Op:GoType` (e.g. `Age:>:int`)                           |
| `-nullable-filter` | `string` | Optional/nullable filters (accepts pointer param e.g. `Age:>:int` $\to$ `age *int`) |
| `-rows`            | `int`    | Preallocation capacity hint for result slices                                       |

---

## CLI Examples

### 1. Select Specific Fields
```bash
rawgen -model User -select "ID,Name,Email" > queries/user.go
```

### 2. Omit Sensitive Columns
```bash
rawgen -model User -omit "PasswordHash,RefreshToken" > queries/user.go
```

### 3. Add Compile-Time Filters
```bash
rawgen -model User -filter "Age:>:int,Status:=:string" > queries/user.go
```
*Generates a zero-allocation, compile-time static query:*
```go
func QueryUsers(ctx context.Context, db DBTX, age int, status string) ([]*models.User, error) {
    const query = `SELECT id, name, age, status FROM users WHERE age > $1 AND status = $2`
    rows, err := db.QueryContext(ctx, query, age, status)
    // ...
}
```

### 4. Add Nullable / Dynamic Filters
```bash
rawgen -model User -nullable-filter "Age:>:int,Status:=:string" > queries/user.go
```
*Generates an optimized dynamic query builder using `strings.Builder.Grow` (only applied when pointers are non-nil):*
```go
func QueryUsers(ctx context.Context, db DBTX, age *int, status *string) ([]*models.User, error)
```

### 5. Custom Table Name & Preallocated Result Slice
```bash
rawgen -model User -table "accounts" -rows 50 > queries/user.go
```

---

## Programmatic Go Usage

You can embed `rawgen` in your code generation scripts:

```go
package main

import (
    "os"
    "github.com/abiiranathan/apigen/parser"
    "github.com/abiiranathan/apigen/rawgen"
)

func main() {
    // 1. Write the shared DBTX interface once
    dbtxFile, _ := os.Create("queries/dbtx.go")
    defer dbtxFile.Close()
    _ = rawgen.WriteDBTX(dbtxFile)

    // 2. Parse model metadata
    meta := parser.Parse([]string{"github.com/example/app/models"})

    // 3. Generate raw SQL queries for the User model
    userFile, _ := os.Create("queries/user.go")
    defer userFile.Close()

    opts := rawgen.Options{
        ModelName: "User",
        ModelPkg:  "github.com/example/app/models",
        Filters: []rawgen.Filter{
            {Column: "Age", Op: ">", GoType: "int", Nullable: true},
        },
        EstimatedRows: 25,
    }

    if err := rawgen.Generate(userFile, meta, opts); err != nil {
        panic(err)
    }
}
```

---

## Using Generated Code in Your Application

### Standard Pool Execution
```go
db, _ := sql.Open("pgx", dsn)

// Insert record
user := &models.User{Name: "Alice", Age: 30}
if err := queries.InsertUser(ctx, db, user); err != nil {
    log.Fatal(err)
}
fmt.Println("Inserted user with ID:", user.ID)

// Fetch by primary key
found, err := queries.GetUser(ctx, db, user.ID)
```

### Atomic Transactions
Because all generated functions take `queries.DBTX`, transactions use the exact same functions:

```go
tx, _ := db.BeginTx(ctx, nil)
defer tx.Rollback()

// Pass the transaction handle `tx` directly
if err := queries.InsertUser(ctx, tx, user); err != nil {
    return err
}

if err := queries.UpdateUser(ctx, tx, user); err != nil {
    return err
}

return tx.Commit()
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.
