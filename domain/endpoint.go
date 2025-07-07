package domain

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/nextap-solutions/openapi3Struct"
)

func GetTypeName(v any) string {
	if v == nil {
		return ""
	}
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ.Name()
}

func ToPointer[T any](v T) *T {
	return &v
}

type OperationBuilder struct {
	op        *openapi3.Operation
	responses map[int]*openapi3.ResponseRef
}

func NewOperationBuilder() *OperationBuilder {
	return &OperationBuilder{
		op:        &openapi3.Operation{},
		responses: make(map[int]*openapi3.ResponseRef),
	}
}

func (ob *OperationBuilder) WithTags(tags ...string) *OperationBuilder {
	ob.op.Tags = tags
	return ob
}

func (ob *OperationBuilder) WithDescription(desc string) *OperationBuilder {
	ob.op.Description = desc
	return ob
}

func (ob *OperationBuilder) WithRequestBodyType(bodyType any, description string, required bool) *OperationBuilder {
	if bodyType == nil {
		fmt.Println("Warning: bodyType is nil for request body.")
		ob.op.RequestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Required:    required,
				Description: description,
				Content: map[string]*openapi3.MediaType{
					"application/json": {
						Schema: openapi3.NewSchemaRef("", nil),
					},
				},
			},
		}
		return ob
	}

	typ := reflect.TypeOf(bodyType)
	isSlice := typ.Kind() == reflect.Slice
	var itemTypeName string
	var schemaRef *openapi3.SchemaRef

	if isSlice {
		elemType := typ.Elem()
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		itemTypeName = elemType.Name()
		if itemTypeName == "" {
			fmt.Printf("Warning: Could not determine element type name for request body array: %T\n", bodyType)
			return ob
		}
		schemaRef = &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: openapi3.NewArraySchema().Type,
				Items: &openapi3.SchemaRef{
					Ref: fmt.Sprintf("#/components/schemas/%s", itemTypeName),
				},
			},
		}
	} else {
		itemTypeName = GetTypeName(bodyType)
		if itemTypeName == "" {
			fmt.Printf("Warning: Could not determine type name for request body: %T\n", bodyType)
			return ob
		}
		schemaRef = &openapi3.SchemaRef{
			Ref: fmt.Sprintf("#/components/schemas/%s", itemTypeName),
		}
	}

	ob.op.RequestBody = &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Required:    required,
			Description: description,
			Content: map[string]*openapi3.MediaType{
				"application/json": {
					Schema: schemaRef,
				},
			},
		},
	}
	return ob
}

func (ob *OperationBuilder) WithResponse(statusCode int, description string, responseType any) *OperationBuilder {
	var schemaRef *openapi3.SchemaRef
	var content map[string]*openapi3.MediaType

	if responseType != nil {
		typeName := GetTypeName(responseType)
		if typeName == "" {
			fmt.Printf("Warning: Could not determine type name for response type: %T\n", responseType)
			schemaRef = openapi3.NewSchemaRef("", nil) // Empty schema
		} else {
			schemaRef = openapi3.NewSchemaRef(fmt.Sprintf("#/components/schemas/%s", typeName), nil)
		}
		content = map[string]*openapi3.MediaType{
			"application/json": {
				Schema: schemaRef,
			},
		}
	}

	ob.responses[statusCode] = &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: ToPointer(description),
			Content:     content,
		},
	}
	return ob
}

func (ob *OperationBuilder) WithParameter(name, in, description string, required bool, schemaType any) *OperationBuilder {
	param := &openapi3.Parameter{
		Name:        name,
		In:          in,
		Description: description,
		Required:    required,
	}

	// WIP - it'll be improved once i start working on hermes
	if schemaType != nil {
		typeName := GetTypeName(schemaType)
		var paramSchema *openapi3.Schema
		if typeName == "" {
			switch reflect.TypeOf(schemaType).Kind() {
			case reflect.String:
				paramSchema = openapi3.NewStringSchema()
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				paramSchema = openapi3.NewInt64Schema()
			case reflect.Float32, reflect.Float64:
				paramSchema = openapi3.NewFloat64Schema()
			case reflect.Bool:
				paramSchema = openapi3.NewBoolSchema()
			default:
				fmt.Printf("Warning: Could not determine primitive OpenAPI schema for parameter '%s' type: %T. Schema will be empty.\n", name, schemaType)
				paramSchema = openapi3.NewSchema()
			}
			param.Schema = openapi3.NewSchemaRef("", paramSchema)
		} else {
			param.Schema = openapi3.NewSchemaRef(fmt.Sprintf("#/components/schemas/%s", typeName), nil)
		}
	} else {
		param.Schema = openapi3.NewSchemaRef("", nil)
	}

	ob.op.Parameters = append(ob.op.Parameters, &openapi3.ParameterRef{
		Value: param,
	})
	return ob
}

func (ob *OperationBuilder) WithParameterRef(ref string) *OperationBuilder {
	ob.op.Parameters = append(ob.op.Parameters, &openapi3.ParameterRef{
		Ref: ref,
	})
	return ob
}

func (ob *OperationBuilder) Build() *openapi3.Operation {
	responses := []openapi3.NewResponsesOption{}
	for status, res := range ob.responses {
		responses = append(responses, openapi3.WithStatus(status, res))
	}
	ob.op.Responses = openapi3.NewResponses(responses...)
	return ob.op
}

type HandlerProvider interface{}

type EndpointDoc struct {
	Path     string
	Version  int
	Method   string
	PathItem *OperationBuilder
}

func (ep *EndpointDoc) GetPath() string {
	path := ""
	if ep.Version > 0 {
		path = fmt.Sprintf("/v%d/%s", ep.Version, ep.Path)
	} else {
		path = fmt.Sprintf("/%s", ep.Path)
	}

	return path
}

func (ep *EndpointDoc) BuildOpenAPiStruct() openapi3Struct.Path {
	item := openapi3.PathItem{}
	op := ep.PathItem.Build()
	switch ep.Method {
	case http.MethodPost:
		item.Post = op
	case http.MethodGet:
		item.Get = op
	case http.MethodPut:
		item.Put = op
	case http.MethodDelete:
		item.Delete = op
	case http.MethodOptions:
		item.Options = op
	default:
		panic(fmt.Sprintf("Unknown request method: %d", ep.Method))
	}
	return openapi3Struct.Path{
		Path: ep.GetPath(),
		Item: item,
	}
}
