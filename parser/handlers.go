package parser

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

type handlerTmplData struct {
	PkgName      string     // Package name for the generated service.
	ModelPkg     string     // Absolute name of package e.g "github.com/abiiranathan/todos/models"
	ModelPkgName string     // Name of package e.g "models"
	Model        string     // The struct name e.g "{{.Model}}"
	ModelObj     StructMeta // The model metadata object
	WritePKGDecl bool
	Preloads     []string // Stores fields to preload
	SkipService  bool     // Whether to skip creating this service
	SvcPkgName   string   // Name of package e.g "services"
}

// Data for import metadata
type HeaderData struct {
	HandlerPkgName string // Package name for handlers generated
	ModelPkg       string // Absolute name of package e.g "github.com/abiiranathan/todos/models"
	SvcPkg         string // Absolute name of package e.g "github.com/abiiranathan/todos/services"
	SvcPkgName     string // Services names
}

/*
Generate fiber/v2 route handlers for all services, skipping models in skip.

inputs: Parsed Struct metadata

w: Where to write the generate code for handlers

modelPkg: Package for the models

svcPkg: Package for the services

skip: The model names whose services where skipped.
*/
func generateHandlers(
	inputs []StructMeta,
	modelPkg string,
	svcPkg string,
	handlerPkgName string,
	skip ...string) (handlerBytes []byte, validation []byte, err error) {

	parts := strings.Split(modelPkg, "/")
	modelPkgName := parts[len(parts)-1]

	svcParts := strings.Split(svcPkg, "/")
	svcPkgName := svcParts[len(svcParts)-1]

	buf := new(bytes.Buffer)

	index := 0
	data := make([]handlerTmplData, 0, len(inputs))

	for _, st := range inputs {
		if sliceContains(skip, st.Name) {
			continue
		}

		data = append(data, handlerTmplData{
			ModelPkg:     modelPkg,
			ModelPkgName: modelPkgName,
			SvcPkgName:   svcPkgName,
			ModelObj:     st,
			Model:        st.Name,
			WritePKGDecl: index == 0,
			SkipService:  sliceContains(skip, st.Name),
		})
		index++
	}

	// Parse header
	tmpl1, err := template.New("handlersTmplHeader").Funcs(template.FuncMap{
		"ToLower": strings.ToLower,
		"ToCamelCase": func(s string) string {
			return strcase.ToCamel(enCaser.String(s))
		},
	}).Parse(headerTmpl)

	if err != nil {
		return nil, nil, fmt.Errorf("error parsing header template: %v", err)
	}

	err = tmpl1.Execute(buf, HeaderData{
		HandlerPkgName: handlerPkgName,
		ModelPkg:       modelPkg,
		SvcPkg:         svcPkg,
		SvcPkgName:     svcPkgName,
	})

	if err != nil {
		return nil, nil, err
	}

	// Write validation logic

	tmpl, err := template.New("handlersTmpl").Funcs(template.FuncMap{
		"ToLower": strings.ToLower,
		"ToCamelCase": func(s string) string {
			return strcase.ToCamel(enCaser.String(s))
		},
	}).Parse(handlerTmpl)

	if err != nil {
		return nil, nil, fmt.Errorf("error parsing template: %w", err)
	}

	if err := tmpl.Execute(buf, data); err != nil {
		return nil, nil, err
	}

	// Format source
	b, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, nil, fmt.Errorf("error format source file: %w, source: %s", err, b)
	}
	validation = []byte(fmt.Sprintf(ValidationText, handlerPkgName))
	formattedValidation, err := format.Source(validation)
	if err != nil {
		return nil, nil, err
	}
	return b, formattedValidation, nil
}

var headerTmpl = `package {{.HandlerPkgName}}


import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"{{.SvcPkg}}"
	"{{.ModelPkg}}"
)

type Handlers struct {
	db *gorm.DB
	svc *{{.SvcPkgName}}.Service
}

// Create a new handler instance for routing.
// All requests are validated using the go validator/v10 pkg.
// The default tag is validate. Call handlers.DefaultValidator.SetTagName(tagName)
// to change the validation tag for your structs.
func New(db *gorm.DB, svc *{{.SvcPkgName}}.Service) *Handlers{
	return &Handlers{db: db, svc:svc}
}
`

var handlerTmpl = `
{{range .}}
{{$ident := .Model | ToLower}}
{{$pkType := .ModelObj.PKType}}

func (h *Handlers) Create{{.Model}}(options ...{{.SvcPkgName}}.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var {{$ident}} {{.ModelPkgName}}.{{.Model}}
		if err := BodyParser(c, &{{$ident}}); err != nil{
			return err
		}
		if err := h.svc.{{.Model}}Service.Create(&{{$ident}}, options...); err != nil{
			return err
		}
		return c.Status(fiber.StatusCreated).JSON({{$ident}}) 
	}
}


func (h *Handlers) Get{{.Model}}(options ...{{.SvcPkgName}}.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		{{$ident}}Id := {{$pkType}}(GetParam(c, "id"))
		{{$ident}}, err := h.svc.{{.Model}}Service.Get({{$ident}}Id, options...)
		if err != nil{
			return err
		}
		return c.JSON({{$ident}}) 
	}
}

func (h *Handlers) GetAll{{.Model}}s(options ...{{.SvcPkgName}}.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		{{$ident}}s, err := h.svc.{{.Model}}Service.GetAll(options...)
		if err != nil{
			return err
		}
		return c.JSON({{$ident}}s) 
	}
}

func (h *Handlers) GetPaginated{{.Model}}s(options ...{{.SvcPkgName}}.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		page := GetQuery(c, "page")
		pageSize := GetQuery(c, "limit")

		if pageSize <= 0 {
			pageSize = 25
		}

		{{$ident}}s, err := h.svc.{{.Model}}Service.GetPaginated(page, pageSize, options...)
		if err != nil{
			return err
		}
		return c.JSON({{$ident}}s) 
	}
}

func (h *Handlers) FindMany{{.Model}}s(options ...{{.SvcPkgName}}.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		{{$ident}}s, err := h.svc.{{.Model}}Service.FindMany(options...)
		if err != nil{
			return err
		}
		return c.JSON({{$ident}}s) 
	}
}

func (h *Handlers) Update{{.Model}}(options ...{{.SvcPkgName}}.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var {{$ident}} {{.ModelPkgName}}.{{.Model}}
		if err := BodyParser(c, &{{$ident}}); err != nil{
			return err
		}

		// Since we expect a full update, {{$ident}} must have an ID
		if {{$ident}}.ID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Record missing id field.")
		}

		updated{{.Model}}, err := h.svc.{{.Model}}Service.Update({{$ident}}.ID, &{{$ident}}, options...)
		if err != nil{
			return err
		}
		return c.JSON(updated{{.Model}}) 
	}
}

func (h *Handlers) Partial{{.Model}}Update(options ...{{.SvcPkgName}}.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		{{$ident}}Id := {{$pkType}}(GetParam(c, "id"))

		var {{$ident}} {{.ModelPkgName}}.{{.Model}}
		if err := BodyParser(c, &{{$ident}}); err != nil{
			return err
		}

		// Specify options for Where, Order, Omit or Select, etc.

		updated{{.Model}}, err := h.svc.{{.Model}}Service.PartialUpdate({{$ident}}Id, {{$ident}}, options...)
		if err != nil{
			return err
		}
		return c.JSON(updated{{.Model}}) 
	}
}

func (h *Handlers) Delete{{.Model}}(options ...{{.SvcPkgName}}.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		{{$ident}}Id := {{$pkType}}(GetParam(c, "id"))
		err := h.svc.{{.Model}}Service.Delete({{$ident}}Id)
		if err != nil{
			return err
		}
		return c.JSON("record deleted successfully") 
	}
}

func (h *Handlers) Delete{{.Model}}sWhere(options ...{{.SvcPkgName}}.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Specify conditions for delete with arguments
		condition := ""
		args := []any{}

		err := h.svc.{{.Model}}Service.DeleteWhere(condition, args...)
		if err != nil{
			return err
		}
		return c.JSON("records deleted successfully") 
	}
}
{{end}}


`

type routeTmplData struct {
	HandlerPkgName string
	HandlerImport  string
	Models         []string
}

var routesTemplate = `package {{.HandlerPkgName}}

import(
	"github.com/gofiber/fiber/v2"
)

// Set up all routes for registered models
func SetupRoutes(api fiber.Router, h *Handlers){
	
{{range .Models}}
{{$ident := . | ToLower | Pluralize}}

// API prefix for {{$ident}}
{{$ident}}Prefix := api.Group("/{{$ident}}")

{{$ident}}Prefix.Get("/", h.GetAll{{.}}s())
{{$ident}}Prefix.Get("/:id<int>", h.Get{{.}}())
{{$ident}}Prefix.Get("/paginated", h.GetPaginated{{.}}s())
{{$ident}}Prefix.Post("", h.Create{{.}}())
{{$ident}}Prefix.Put("/:id<int>", h.Update{{.}}())
{{$ident}}Prefix.Patch("/:id<int>", h.Partial{{.}}Update())
{{$ident}}Prefix.Delete("/:id<int>", h.Delete{{.}}())
{{end}}

}
`
