# apigen

`apigen` is a tool written in go that parses go structs and generates code for querying the database based on [GORM](https://gorm.io) ORM. It also generates sql for creating postgres TYPES (enums) for enumerated constants in the module.

## Features

- generates services for all structs in your models unless skipped in the configuration file(`apigen.toml`)
- Created database connection helper with sane defaults.
- Preloads all relationships(even nested relationships - for now) by default. Because the parser knows foreign keys and tree, we are able to do that for all `foreignKey` and `many2many` fields.
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
# Note that toml(v2) requires single quotes for strings

[Models]
# package containing your models.
Pkg='github.com/abiiranathan/apigen/models'

# List of models to skip when generating services and handlers
Skip=['User', 'Payment']

[Output]
# The directory to put the generated code. (Default is current working directory)
OutDir='.'

# Name for your services package
ServiceName='services'

# Overrides for enumerated constants
# These will be used for the typescript fields.
[overrides]
  [overrides.types]
    Sex='"Male" | "Female"'

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

    "github.com/gofiber/fiber/v2"
	"gorm.io/gorm/logger"
)

func main() {
	db, err := services.PostgresConnection(dsn, "Africa/kampala", logger.Silent)
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&models.User{}, ...more models)
	if err != nil {
		log.Fatalf("Error performing database migrations: %v\n", err)
	}

	app := fiber.New()
	api := app.Group("/api/v1")
	svc := services.NewService(db)

    users, err := svc.Users.GetAll()

	app.Listen(":8000")

}
```
