package rawgen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/abiiranathan/apigen/parser"
)

func testMeta() []parser.StructMeta {
	return []parser.StructMeta{
		{
			Name:    "User",
			PKType:  "int",
			Package: "github.com/abiiranathan/apigen/models",
			Fields: []parser.Field{
				{Name: "ID", Type: "int", BaseType: "int", Parent: "User"},
				{Name: "Name", Type: "string", BaseType: "string", Parent: "User"},
				{Name: "Age", Type: "int", BaseType: "int", Parent: "User"},
				{Name: "Discount", Type: "float64", BaseType: "float64", Parent: "User"},
				{Name: "RoleID", Type: "int64", BaseType: "int64", Parent: "User"},
				{Name: "Role", Type: "Role", BaseType: "Role", Preload: true, Parent: "User"},
				{Name: "Tags", Type: "[]Tag", BaseType: "Tag", Preload: true, Parent: "User"},
			},
		},
		{
			Name:    "Role",
			PKType:  "int64",
			Package: "github.com/abiiranathan/apigen/models",
			Fields: []parser.Field{
				{Name: "ID", Type: "int64", BaseType: "int64", Parent: "Role"},
				{Name: "Name", Type: "string", BaseType: "string", Parent: "Role"},
			},
		},
		{
			Name:    "Issue",
			PKType:  "int64",
			Package: "github.com/abiiranathan/apigen/models",
			Fields: []parser.Field{
				{Name: "ID", Type: "int64", BaseType: "int64", Parent: "Issue"},
				{Name: "Name", Type: "string", BaseType: "string", Parent: "Issue"},
			},
		},
	}
}

func gen(t *testing.T, opts Options) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Generate(&buf, testMeta(), opts); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	return buf.String()
}

func has(t *testing.T, output, substr string) {
	t.Helper()
	if !strings.Contains(output, substr) {
		t.Errorf("output should contain %q\nGot:\n%s", substr, output)
	}
}

func hasNot(t *testing.T, output, substr string) {
	t.Helper()
	if strings.Contains(output, substr) {
		t.Errorf("output should NOT contain %q\nGot:\n%s", substr, output)
	}
}

func TestBasicGeneration(t *testing.T) {
	out := gen(t, Options{
		ModelName: "User",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
	})

	has(t, out, "package queries")
	has(t, out, `"database/sql"`)
	has(t, out, `"context"`)
	has(t, out, `"github.com/abiiranathan/apigen/models"`)
	hasNot(t, out, `"strings"`)

	has(t, out, "func InsertUser(ctx context.Context, db *sql.DB, u *models.User) error")
	has(t, out, "INSERT INTO users")
	has(t, out, "name, age, discount, role_id")
	has(t, out, "$1, $2, $3, $4")
	has(t, out, "RETURNING id")
	has(t, out, ".Scan(&u.ID)")

	has(t, out, "func GetUser(ctx context.Context, db *sql.DB, id int) (*models.User, error)")
	has(t, out, "SELECT id, name, age, discount, role_id FROM users WHERE id = $1")

	has(t, out, "func DeleteUser(ctx context.Context, db *sql.DB, id int) error")
	has(t, out, "DELETE FROM users WHERE id = $1")
	has(t, out, "sql.ErrNoRows")

	has(t, out, "func UpdateUser(ctx context.Context, db *sql.DB, u *models.User) error")
	has(t, out, "UPDATE users SET name = $1, age = $2, discount = $3, role_id = $4 WHERE id = $5")

	has(t, out, "func QueryUsers(ctx context.Context, db *sql.DB) ([]*models.User, error)")
	has(t, out, "SELECT id, name, age, discount, role_id FROM users")
	has(t, out, "rows.Close()")
	has(t, out, "rows.Err()")
}

func TestRelationFieldsExcluded(t *testing.T) {
	out := gen(t, Options{
		ModelName: "User",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
	})

	for _, line := range strings.Split(out, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "select") || strings.Contains(lower, "insert") || strings.Contains(lower, "update") {
			if strings.Contains(lower, ", role,") || strings.Contains(lower, ", tags") {
				t.Errorf("relation field leaked into SQL: %s", line)
			}
		}
	}
	has(t, out, "role_id")
}

func TestSelectFields(t *testing.T) {
	out := gen(t, Options{
		ModelName:    "User",
		ModelPkg:     "github.com/abiiranathan/apigen/models",
		SelectFields: []string{"ID", "Name", "Age"},
	})

	has(t, out, "SELECT id, name, age FROM users")
	has(t, out, "INSERT INTO users (name, age)")
	has(t, out, "$1, $2")
	hasNot(t, out, "discount")
	hasNot(t, out, "role_id")
	has(t, out, "&u.ID, &u.Name, &u.Age")
	hasNot(t, out, "&u.Discount")
	hasNot(t, out, "&u.RoleID")
}

func TestOmitFields(t *testing.T) {
	out := gen(t, Options{
		ModelName:  "User",
		ModelPkg:   "github.com/abiiranathan/apigen/models",
		OmitFields: []string{"Discount"},
	})

	has(t, out, "SELECT id, name, age, role_id FROM users")
	hasNot(t, out, "discount")
	has(t, out, "INSERT INTO users (name, age, role_id)")
	has(t, out, "$1, $2, $3")
	has(t, out, "UPDATE users SET name = $1, age = $2, role_id = $3 WHERE id = $4")
}

func TestOmitMultipleFields(t *testing.T) {
	out := gen(t, Options{
		ModelName:  "User",
		ModelPkg:   "github.com/abiiranathan/apigen/models",
		OmitFields: []string{"Discount", "Age"},
	})

	has(t, out, "SELECT id, name, role_id FROM users")
	hasNot(t, out, "discount")
	hasNot(t, out, ", age")
}

func TestCustomTableName(t *testing.T) {
	out := gen(t, Options{
		ModelName: "User",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
		TableName: "app_users",
	})

	has(t, out, "INSERT INTO app_users")
	has(t, out, "FROM app_users")
	has(t, out, "DELETE FROM app_users")
	has(t, out, "UPDATE app_users")
	hasNot(t, out, "FROM users")
}

func TestRequiredFilters(t *testing.T) {
	out := gen(t, Options{
		ModelName: "User",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
		Filters: []Filter{
			{Column: "Age", Op: ">", GoType: "int"},
			{Column: "Name", Op: "ILIKE", GoType: "string"},
		},
	})

	has(t, out, `"strings"`)
	has(t, out, `"fmt"`)
	has(t, out, "func QueryUsers(ctx context.Context, db *sql.DB, age int, name string)")
	has(t, out, `age > $%d`)
	has(t, out, `name ILIKE $%d`)
	has(t, out, `strings.Join(conditions, " AND ")`)
	has(t, out, "args...")
}

func TestSingleFilter(t *testing.T) {
	out := gen(t, Options{
		ModelName: "User",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
		Filters: []Filter{
			{Column: "Age", Op: "=", GoType: "int"},
		},
	})

	has(t, out, "func QueryUsers(ctx context.Context, db *sql.DB, age int)")
	has(t, out, `age = $%d`)
}

func TestNullableFilters(t *testing.T) {
	out := gen(t, Options{
		ModelName: "User",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
		Filters: []Filter{
			{Column: "Age", Op: ">", GoType: "int", Nullable: true},
		},
	})

	has(t, out, "age *int")
	has(t, out, "if age != nil")
	has(t, out, "*age")
}

func TestMixedFilters(t *testing.T) {
	out := gen(t, Options{
		ModelName: "User",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
		Filters: []Filter{
			{Column: "Name", Op: "ILIKE", GoType: "string"},
			{Column: "Age", Op: ">", GoType: "int", Nullable: true},
		},
	})

	has(t, out, "name string")
	has(t, out, "age *int")
	has(t, out, `name ILIKE $%d`)
	has(t, out, "if age != nil")
}

func TestModelNotFound(t *testing.T) {
	var buf bytes.Buffer
	err := Generate(&buf, testMeta(), Options{
		ModelName: "NonExistent",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
	})
	if err == nil {
		t.Fatal("expected error for non-existent model, got nil")
	}
	if !strings.Contains(err.Error(), "NonExistent") {
		t.Errorf("error should mention model name, got: %v", err)
	}
}

func TestModelWithoutPK(t *testing.T) {
	meta := []parser.StructMeta{
		{
			Name:    "Metric",
			PKType:  "",
			Package: "github.com/example/app/models",
			Fields: []parser.Field{
				{Name: "Value", Type: "float64", BaseType: "float64", Parent: "Metric"},
				{Name: "Label", Type: "string", BaseType: "string", Parent: "Metric"},
			},
		},
	}

	var buf bytes.Buffer
	err := Generate(&buf, meta, Options{
		ModelName: "Metric",
		ModelPkg:  "github.com/example/app/models",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	out := buf.String()

	has(t, out, "func InsertMetric")
	has(t, out, "ExecContext")
	hasNot(t, out, "RETURNING")
	hasNot(t, out, "func GetMetric")
	hasNot(t, out, "func DeleteMetric")
	hasNot(t, out, "func UpdateMetric")
	has(t, out, "func QueryMetrics")
}

func TestRoleModel(t *testing.T) {
	out := gen(t, Options{
		ModelName: "Role",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
	})

	has(t, out, "func InsertRole")
	has(t, out, "func GetRole")
	has(t, out, "func DeleteRole")
	has(t, out, "func UpdateRole")
	has(t, out, "func QueryRoles")
	has(t, out, "id int64")
	has(t, out, "FROM roles")
	has(t, out, "INSERT INTO roles")
	has(t, out, "SELECT id, name FROM roles")
}

func TestIssueModel(t *testing.T) {
	out := gen(t, Options{
		ModelName: "Issue",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
	})

	has(t, out, "func InsertIssue")
	has(t, out, "FROM issues")
}

func TestSelectAndOmitCombined(t *testing.T) {
	out := gen(t, Options{
		ModelName:    "User",
		ModelPkg:     "github.com/abiiranathan/apigen/models",
		SelectFields: []string{"ID", "Name"},
		OmitFields:   []string{"Name"},
	})

	hasNot(t, out, "name")
	has(t, out, "SELECT id FROM users")
}

func TestDefaultTableNameSnakeCase(t *testing.T) {
	meta := []parser.StructMeta{
		{
			Name:    "BlogPost",
			PKType:  "int",
			Package: "github.com/example/app/models",
			Fields: []parser.Field{
				{Name: "ID", Type: "int", BaseType: "int", Parent: "BlogPost"},
				{Name: "Title", Type: "string", BaseType: "string", Parent: "BlogPost"},
			},
		},
	}

	var buf bytes.Buffer
	err := Generate(&buf, meta, Options{
		ModelName: "BlogPost",
		ModelPkg:  "github.com/example/app/models",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	out := buf.String()

	has(t, out, "blog_posts")
}

func TestGeneratedHeader(t *testing.T) {
	out := gen(t, Options{
		ModelName: "User",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
	})

	has(t, out, "// Code generated by apigen rawgen; DO NOT EDIT.")
}

func TestScanOrderMatchesColumns(t *testing.T) {
	out := gen(t, Options{
		ModelName: "User",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
	})

	has(t, out, "&u.ID, &u.Name, &u.Age, &u.Discount, &u.RoleID")
}

func TestUpdateParameterOrder(t *testing.T) {
	out := gen(t, Options{
		ModelName: "User",
		ModelPkg:  "github.com/abiiranathan/apigen/models",
	})

	has(t, out, "name = $1, age = $2, discount = $3, role_id = $4 WHERE id = $5")

	idx := strings.Index(out, "func UpdateUser")
	updateFunc := out[idx:]
	nameIdx := strings.Index(updateFunc, "u.Name,")
	idIdx := strings.Index(updateFunc, "u.ID,")
	if nameIdx > idIdx {
		t.Error("Update args: non-PK columns should come before PK")
	}
}

func TestCustomPackageName(t *testing.T) {
	meta := []parser.StructMeta{
		{
			Name:    "Item",
			PKType:  "int",
			Package: "github.com/example/myapp/domain",
			Fields: []parser.Field{
				{Name: "ID", Type: "int", BaseType: "int", Parent: "Item"},
				{Name: "Label", Type: "string", BaseType: "string", Parent: "Item"},
			},
		},
	}

	var buf bytes.Buffer
	err := Generate(&buf, meta, Options{
		ModelName: "Item",
		ModelPkg:  "github.com/example/myapp/domain",
	})
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	out := buf.String()

	has(t, out, "domain.Item")
	has(t, out, `"github.com/example/myapp/domain"`)
}
