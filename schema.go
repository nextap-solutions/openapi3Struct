package openapi3Struct

import (
	"fmt"
	"go/ast"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// See https://regex101.com/r/8SGj7m/1
var tagReqexp = regexp.MustCompile(`([^  \x60\n][a-zA-z0-9_-]+):"? ?([ a-zA-z0-9{},_-]+)"? ?`)

func resolveSchema(schemas openapi3.Schemas, s ast.Spec, doc string) (*string, openapi3.Schema) {
	schema := openapi3.Schema{
		Required: []string{},
	}
	// fmt.Printf("resolveSchema %T\n", s)
	switch s := s.(type) {
	case *ast.TypeSpec:
		switch st := s.Type.(type) {
		case *ast.FuncType:
			// fmt.Printf("FuncType\n")
		case *ast.StructType:
			// fmt.Printf("StructType\n")
			schema := openapi3.Schema{
				Type: "object",
			}
			fiels := openapi3.Schemas{}
			for _, f := range st.Fields.List {
				// fmt.Printf("",f.Type)
				name := f.Names[0].Name
				fieldSchema, required := resolveField(schemas, f, f.Type)

				if f.Tag != nil {
					// fmt.Printf("Field %v %s\n", f.Names, f.Tag.Value)
					matches := tagReqexp.FindAllStringSubmatch(f.Tag.Value, -1)
					// fmt.Printf("matches %v\n", matches)
					for _, match := range matches {
						if len(match) != 3 {
							continue
						}
						if match[1] == "json" {
							name = match[2]
						}

						// Handle oapi tag
						if strings.HasPrefix(match[1], "oapi") {
							requiredAttr := updateSchemAttribute(fieldSchema, match[0])
							if requiredAttr {
								required = true
							}
						}
					}
				}

				if f.Doc != nil {
					for _, line := range strings.Split(f.Doc.Text(), "\n") {
						if strings.HasPrefix(line, "oapi") {
							requiredAttr := updateSchemAttribute(fieldSchema, line)
							if requiredAttr {
								required = true
							}
						}
					}
				}

				fiels[name] = fieldSchema
				if required {
					schema.Required = append(schema.Required, name)
				}
			}

			schema.Properties = fiels
			return &s.Name.Name, schema
		default:
			// fmt.Printf("default %s %s %s\n", s.Name.Name, s.Doc.Text(), s.Comment.Text())
			schema := openapi3.Schema{
				Type: fmt.Sprintf("%v", s.Type.(*ast.Ident).Name),
			}
			return nil, schema
		}
	}
	return nil, schema
}

func updateSchemAttribute(fieldSchema *openapi3.SchemaRef, keyValue string) bool {
	fullMatch := tagReqexp.FindAllStringSubmatch(keyValue, -1)
	if len(fullMatch) != 1 {
		// TODO handle error
	}
	match := fullMatch[0]
	attrs := strings.Split(match[1], "_")
	// TODO handle error
	if len(attrs) != 2 {
	}
	attrName := attrs[1]
	attrName = strings.ToUpper(string(attrs[1][0])) + string(attrName[1:])
	// fmt.Printf("Setitng %s to %s\n", attrName, match[2])
	if attrName == "Required" {
		if match[2] == "true" {
			return true
		}

		return false
	}

	rfValue := reflect.ValueOf(fieldSchema.Value).Elem()
	fv := rfValue.FieldByName(attrName)
	// fmt.Printf("Type %v\n", fv.Type())
	fvType := fv.Type().String()
	pointer := false
	if strings.HasPrefix(fvType, "*") {
		pointer = true
		fvType = string(fvType[1:])
	}
	switch fvType {
	case "bool":
		bool, err := strconv.ParseBool(match[2])
		if err != nil {
			// TODO handle error
			// return nil, err
		}
		if pointer {
			fv.Set(reflect.ValueOf(&bool))
		} else {
			fv.Set(reflect.ValueOf(bool))
		}
	case "float", "float32", "float64":
		float, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			// TODO handle error
			// return nil, err
		}
		if pointer {
			fv.Set(reflect.ValueOf(&float))
		} else {
			fv.Set(reflect.ValueOf(float))
		}
	case "uint64":
		uint, err := strconv.ParseUint(match[2], 10, 64)
		if err != nil {
			// TODO handle error
			// return nil, err
		}
		if pointer {
			fv.Set(reflect.ValueOf(&uint))
		} else {
			fv.Set(reflect.ValueOf(uint))
		}
	case "[]interface {}":
		newValue := []any{}
		currentValue, ok := fv.Interface().([]any)
		if ok {
			newValue = append(newValue, currentValue...)
		}
		values := strings.Split(match[2], ",")
		for _, v := range values {
			newValue = append(newValue, strings.TrimSpace(v))
		}
		fv.Set(reflect.ValueOf(newValue))
	default:
		if pointer {
			fv.Set(reflect.ValueOf(&match[2]))
		} else {
			fv.Set(reflect.ValueOf(match[2]))
		}

	}
	updatedSchema := rfValue.Interface().(openapi3.Schema)
	fieldSchema.Value = &updatedSchema

	return false
}

func resolveField(schemas openapi3.Schemas, f *ast.Field, typ ast.Expr) (*openapi3.SchemaRef, bool) {
	// TODO add option to parse pointers as non optional
	required := true
	var fieldSchema *openapi3.SchemaRef
	switch ft := typ.(type) {
	case *ast.MapType:
		// TODO is this default required correct ?
		return openapi3.NewSchemaRef("", &openapi3.Schema{
			Type: "object",
		}), false
	// TODO improve, we cannot handle array of arrays now
	case *ast.ArrayType:
		el := ft.Elt
		switch at := ft.Elt.(type) {
		case *ast.StarExpr:
			el = at.X
		}
		arraySchema, _ := resolveField(schemas, f, el)
		// TODO is this default required correct ?
		return openapi3.NewSchemaRef("", &openapi3.Schema{
			Items: arraySchema,
			Type:  "array",
		}), false
	// TODO add option to parse pointers as non optional
	case *ast.StarExpr:
		required = false
		typ = ft.X
	}
	ident := typ.(*ast.Ident)

	if ident.Obj != nil {
		doc := ""
		if f.Doc != nil {
			doc = f.Doc.Text()
		}
		name, subSchema := resolveSchema(schemas, ident.Obj.Decl.(*ast.TypeSpec), doc)
		if name != nil {
			fieldSchema = openapi3.NewSchemaRef(createRef(*name), nil)
		} else {
			fieldSchema = openapi3.NewSchemaRef("", &subSchema)
		}
	} else {
		fieldSchema = openapi3.NewSchemaRef("", &openapi3.Schema{
			Type: resolvePrimitiveType(ident.Name),
		})
	}

	return fieldSchema, required
}

func resolvePrimitive(f *ast.Field) (string, *openapi3.Schema) {
	typ := f.Type.(*ast.Ident)
	schema := openapi3.Schema{
		Type: resolvePrimitiveType(typ.Name),
	}

	if f.Tag != nil {
	}

	return f.Names[0].Name, &schema
}

func resolvePrimitiveType(typ string) string {
	switch typ {
	case "int64", "int32", "int":
		return "integer"
	case "float64", "float32", "float":
		return "number"
	case "string", "byte":
		return "string"
	case "bool":
		return "boolean"
	default:
		return typ
	}
}

func createRef(typeName string) string {
	return fmt.Sprintf("#/components/schemas/%s", typeName)
}
