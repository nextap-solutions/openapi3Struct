package openapi3Struct

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// parseTypeSpec parses a single Go source file and returns the TypeSpec and
// declaration map for use with resolveSchema.
func parseTypeSpec(t *testing.T, src string) (*ast.TypeSpec, map[string]*ast.TypeSpec) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	declMap := map[string]*ast.TypeSpec{}
	var target *ast.TypeSpec
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			declMap[ts.Name.Name] = ts
			// Use the last type as the target (the one we want to test).
			target = ts
		}
	}
	if target == nil {
		t.Fatal("no type declaration found in source")
	}
	return target, declMap
}

func TestResolveSchema_StructType(t *testing.T) {
	t.Parallel()

	src := `package test
type Foo struct {
	Name string ` + "`json:\"name\"`" + `
	Age  int    ` + "`json:\"age\"`" + `
}
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	if name == nil || *name != "Foo" {
		t.Fatalf("expected name 'Foo', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "object" {
		t.Fatalf("expected type 'object', got %v", schema.Type)
	}
	if _, ok := schema.Properties["name"]; !ok {
		t.Error("expected property 'name'")
	}
	if _, ok := schema.Properties["age"]; !ok {
		t.Error("expected property 'age'")
	}
}

func TestResolveSchema_ArrayOfPrimitive(t *testing.T) {
	t.Parallel()

	src := `package test
type StringList []string
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	if name == nil || *name != "StringList" {
		t.Fatalf("expected name 'StringList', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "array" {
		t.Fatalf("expected type 'array', got %v", schema.Type)
	}
	if schema.Items == nil {
		t.Fatal("expected items schema")
	}
	if schema.Items.Value == nil || (*schema.Items.Value.Type)[0] != "string" {
		t.Fatalf("expected items type 'string', got %v", schema.Items.Value.Type)
	}
}

func TestResolveSchema_ArrayOfStruct(t *testing.T) {
	t.Parallel()

	src := `package test
type Item struct {
	ID string ` + "`json:\"id\"`" + `
}
type ItemList []Item
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	declMap := map[string]*ast.TypeSpec{}
	var target *ast.TypeSpec
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			declMap[ts.Name.Name] = ts
			if ts.Name.Name == "ItemList" {
				target = ts
			}
		}
	}
	if target == nil {
		t.Fatal("ItemList type not found")
	}

	schemas := openapi3.Schemas{}
	name, schema := resolveSchema(schemas, target, "", declMap)

	if name == nil || *name != "ItemList" {
		t.Fatalf("expected name 'ItemList', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "array" {
		t.Fatalf("expected type 'array', got %v", schema.Type)
	}
	if schema.Items == nil {
		t.Fatal("expected items schema")
	}
	// Item is in declarationMap, so it should be a $ref.
	if schema.Items.Ref != "#/components/schemas/Item" {
		t.Fatalf("expected items ref '#/components/schemas/Item', got %q", schema.Items.Ref)
	}
}

func TestResolveSchema_ArrayOfPointerToStruct(t *testing.T) {
	t.Parallel()

	src := `package test
type Item struct {
	ID string ` + "`json:\"id\"`" + `
}
type ItemPtrList []*Item
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	declMap := map[string]*ast.TypeSpec{}
	var target *ast.TypeSpec
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			declMap[ts.Name.Name] = ts
			if ts.Name.Name == "ItemPtrList" {
				target = ts
			}
		}
	}
	if target == nil {
		t.Fatal("ItemPtrList type not found")
	}

	schemas := openapi3.Schemas{}
	name, schema := resolveSchema(schemas, target, "", declMap)

	if name == nil || *name != "ItemPtrList" {
		t.Fatalf("expected name 'ItemPtrList', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "array" {
		t.Fatalf("expected type 'array', got %v", schema.Type)
	}
	if schema.Items == nil {
		t.Fatal("expected items schema")
	}
	if schema.Items.Ref != "#/components/schemas/Item" {
		t.Fatalf("expected items ref '#/components/schemas/Item', got %q", schema.Items.Ref)
	}
}

func TestResolveSchema_ArrayOfInt(t *testing.T) {
	t.Parallel()

	src := `package test
type IntList []int
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	if name == nil || *name != "IntList" {
		t.Fatalf("expected name 'IntList', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "array" {
		t.Fatalf("expected type 'array', got %v", schema.Type)
	}
	if schema.Items == nil {
		t.Fatal("expected items schema")
	}
	if schema.Items.Value == nil || (*schema.Items.Value.Type)[0] != "integer" {
		t.Fatalf("expected items type 'integer', got %v", schema.Items.Value.Type)
	}
}

func TestResolveSchema_FuncType_ReturnsEmptySchema(t *testing.T) {
	t.Parallel()

	src := `package test
type Handler func(s string) error
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	// FuncType is skipped — returns nil name and empty schema.
	if name != nil {
		t.Fatalf("expected nil name for func type, got %v", *name)
	}
	if schema.Type != nil {
		t.Fatalf("expected nil type for func type, got %v", schema.Type)
	}
}

func TestResolveSchema_TypeDefinition_Primitive(t *testing.T) {
	t.Parallel()

	src := `package test
type MyString string
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	if name == nil || *name != "MyString" {
		t.Fatalf("expected name 'MyString', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "string" {
		t.Fatalf("expected type 'string', got %v", schema.Type)
	}
}

func TestResolveSchema_TypeAlias_Primitive(t *testing.T) {
	t.Parallel()

	src := `package test
type stringType = string
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	// Type aliases should not produce a named schema.
	if name != nil {
		t.Fatalf("expected nil name for type alias, got %v", *name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "string" {
		t.Fatalf("expected type 'string', got %v", schema.Type)
	}
}

func TestResolveSchema_TypeDefinition_Int(t *testing.T) {
	t.Parallel()

	src := `package test
type MyInt int64
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	if name == nil || *name != "MyInt" {
		t.Fatalf("expected name 'MyInt', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "integer" {
		t.Fatalf("expected type 'integer', got %v", schema.Type)
	}
}

func TestResolveSchema_MapType(t *testing.T) {
	t.Parallel()

	src := `package test
type MyMap map[string]string
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	if name == nil || *name != "MyMap" {
		t.Fatalf("expected name 'MyMap', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "object" {
		t.Fatalf("expected type 'object', got %v", schema.Type)
	}
}

func TestResolveSchema_InterfaceType(t *testing.T) {
	t.Parallel()

	src := `package test
type MyInterface interface{}
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	if name == nil || *name != "MyInterface" {
		t.Fatalf("expected name 'MyInterface', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "object" {
		t.Fatalf("expected type 'object', got %v", schema.Type)
	}
}

func TestResolveSchema_ArrayOfSelectorExpr(t *testing.T) {
	t.Parallel()

	src := `package test
import "time"
type TimeList []time.Time
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	declMap := map[string]*ast.TypeSpec{}
	var target *ast.TypeSpec
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			declMap[ts.Name.Name] = ts
			target = ts
		}
	}
	if target == nil {
		t.Fatal("no type declaration found")
	}

	schemas := openapi3.Schemas{}
	name, schema := resolveSchema(schemas, target, "", declMap)

	if name == nil || *name != "TimeList" {
		t.Fatalf("expected name 'TimeList', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "array" {
		t.Fatalf("expected type 'array', got %v", schema.Type)
	}
	if schema.Items == nil {
		t.Fatal("expected items schema")
	}
	if schema.Items.Value == nil || (*schema.Items.Value.Type)[0] != "object" {
		t.Fatalf("expected items type 'object' for external type, got %v", schema.Items.Value.Type)
	}
}

func TestResolveSchema_ArrayOfPointerSelectorExpr(t *testing.T) {
	t.Parallel()

	src := `package test
import "time"
type TimePtrList []*time.Time
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	declMap := map[string]*ast.TypeSpec{}
	var target *ast.TypeSpec
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			declMap[ts.Name.Name] = ts
			target = ts
		}
	}
	if target == nil {
		t.Fatal("no type declaration found")
	}

	schemas := openapi3.Schemas{}
	name, schema := resolveSchema(schemas, target, "", declMap)

	if name == nil || *name != "TimePtrList" {
		t.Fatalf("expected name 'TimePtrList', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "array" {
		t.Fatalf("expected type 'array', got %v", schema.Type)
	}
	if schema.Items == nil {
		t.Fatal("expected items schema")
	}
	if schema.Items.Value == nil || (*schema.Items.Value.Type)[0] != "object" {
		t.Fatalf("expected items type 'object' for external pointer type, got %v", schema.Items.Value.Type)
	}
}

func TestResolveSchema_ArrayOfMap(t *testing.T) {
	t.Parallel()

	src := `package test
type MapList []map[string]string
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	if name == nil || *name != "MapList" {
		t.Fatalf("expected name 'MapList', got %v", name)
	}
	if schema.Type == nil || (*schema.Type)[0] != "array" {
		t.Fatalf("expected type 'array', got %v", schema.Type)
	}
	if schema.Items == nil {
		t.Fatal("expected items schema")
	}
	if schema.Items.Value == nil || (*schema.Items.Value.Type)[0] != "object" {
		t.Fatalf("expected items type 'object' for map element, got %v", schema.Items.Value.Type)
	}
}

// TestResolveField_CrossPackagePointerField tests that a field typed *pkg.Type
// (StarExpr wrapping SelectorExpr) does not panic and resolves as an optional object.
// Regression test for the StarExpr→SelectorExpr fall-through panic.
func TestResolveField_CrossPackagePointerField_DoesNotPanic(t *testing.T) {
	t.Parallel()

	src := `package test
import "time"
type Outer struct {
	Timestamp *time.Time ` + "`json:\"timestamp,omitempty\"`" + `
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	declMap := map[string]*ast.TypeSpec{}
	var target *ast.TypeSpec
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			declMap[ts.Name.Name] = ts
			target = ts
		}
	}
	if target == nil {
		t.Fatal("no type declaration found")
	}

	schemas := openapi3.Schemas{}
	name, schema := resolveSchema(schemas, target, "", declMap)

	if name == nil || *name != "Outer" {
		t.Fatalf("expected name 'Outer', got %v", name)
	}
	prop, ok := schema.Properties["timestamp"]
	if !ok {
		t.Fatal("expected property 'timestamp'")
	}
	if prop.Value == nil || (*prop.Value.Type)[0] != "object" {
		t.Fatalf("expected timestamp type 'object', got %v", prop.Value.Type)
	}
	// Pointer field must not be required.
	for _, req := range schema.Required {
		if req == "timestamp" {
			t.Fatal("expected 'timestamp' to be optional (pointer type), but it was required")
		}
	}
}

// TestResolveField_CrossPackageDirectField tests that a field typed pkg.Type
// (direct SelectorExpr, non-pointer) resolves as an object without panicking.
func TestResolveField_CrossPackageDirectField_ReturnsObject(t *testing.T) {
	t.Parallel()

	src := `package test
import "time"
type Outer struct {
	Timestamp time.Time ` + "`json:\"timestamp\"`" + `
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	declMap := map[string]*ast.TypeSpec{}
	var target *ast.TypeSpec
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			declMap[ts.Name.Name] = ts
			target = ts
		}
	}
	if target == nil {
		t.Fatal("no type declaration found")
	}

	schemas := openapi3.Schemas{}
	name, schema := resolveSchema(schemas, target, "", declMap)

	if name == nil || *name != "Outer" {
		t.Fatalf("expected name 'Outer', got %v", name)
	}
	prop, ok := schema.Properties["timestamp"]
	if !ok {
		t.Fatal("expected property 'timestamp'")
	}
	if prop.Value == nil || (*prop.Value.Type)[0] != "object" {
		t.Fatalf("expected timestamp type 'object', got %v", prop.Value.Type)
	}
}

// TestResolveSchema_NestedStructAutoRegistered tests that when a struct field
// references another local struct, that sub-struct is auto-registered in the
// schemas map so $ref pointers in the output are valid.
func TestResolveSchema_NestedStructAutoRegistered(t *testing.T) {
	t.Parallel()

	src := `package test
type Inner struct {
	Value string ` + "`json:\"value\"`" + `
}
type Outer struct {
	Sub Inner ` + "`json:\"sub\"`" + `
}
`
	// parseTypeSpec returns the last type (Outer) as target with both in declMap.
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	if name == nil || *name != "Outer" {
		t.Fatalf("expected name 'Outer', got %v", name)
	}
	sub, ok := schema.Properties["sub"]
	if !ok {
		t.Fatal("expected property 'sub'")
	}
	if sub.Ref != "#/components/schemas/Inner" {
		t.Fatalf("expected sub to be $ref to Inner, got %q", sub.Ref)
	}
	// Inner must be auto-registered so the $ref is resolvable.
	if _, ok := schemas["Inner"]; !ok {
		t.Fatal("expected Inner to be auto-registered in schemas map")
	}
}

// TestResolveSchema_NestedStructPointerAutoRegistered tests auto-registration
// with a pointer field (*Inner).
func TestResolveSchema_NestedStructPointerAutoRegistered(t *testing.T) {
	t.Parallel()

	src := `package test
type Inner struct {
	Value string ` + "`json:\"value\"`" + `
}
type Outer struct {
	Sub *Inner ` + "`json:\"sub,omitempty\"`" + `
}
`
	ts, declMap := parseTypeSpec(t, src)
	schemas := openapi3.Schemas{}

	name, schema := resolveSchema(schemas, ts, "", declMap)

	if name == nil || *name != "Outer" {
		t.Fatalf("expected name 'Outer', got %v", name)
	}
	sub, ok := schema.Properties["sub"]
	if !ok {
		t.Fatal("expected property 'sub'")
	}
	if sub.Ref != "#/components/schemas/Inner" {
		t.Fatalf("expected sub to be $ref to Inner, got %q", sub.Ref)
	}
	if _, ok := schemas["Inner"]; !ok {
		t.Fatal("expected Inner to be auto-registered in schemas map")
	}
	// Pointer field must be optional.
	for _, req := range schema.Required {
		if req == "sub" {
			t.Fatal("expected 'sub' to be optional (pointer type), but it was required")
		}
	}
}
