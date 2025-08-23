# apigen

`apigen` is a tool written in go that parses go structs and generates code for querying the database based on [GORM](https://gorm.io) ORM.

## Features

- generates services for all structs in your models unless skipped in the configuration file(`apigen.toml`)
- Created database connection helper with sane defaults.
- Optionally Preloads all relationships(even nested relationships) by default. Because the parser knows foreign keys and tree, we are able to do that for all `foreignKey` and `many2many` fields.
- Allows for customizing all queries by specifying optional Where, ordering, grouping, select `options ...services.Options` These options are passed to the callable handlers that are designed with the decorator pattern.
- Generates typescript interfaces for your models.

## Installation

### As a CLI tool

```console
go install github.com/abiiranathan/apigen@latest
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
