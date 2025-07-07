package domain

import (
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
)

type ParsedPath struct {
	Path string
	Item openapi3.PathItem
}

func (p *Path) Parse() ParsedPath {
	return ParsedPath{
		Path: p.Path,
		Item: ParsePathItem(p.Item),
	}
}

func ParsePathItem(item PathItem) openapi3.PathItem {

	oapiPathItem := openapi3.PathItem{}

	if item.Get != nil {
		oapiPathItem.Get = ParseOperation(item.Get)
	}
	if item.Post != nil {
		oapiPathItem.Post = ParseOperation(item.Post)
	}
	if item.Delete != nil {
		oapiPathItem.Delete = ParseOperation(item.Delete)
	}
	if item.Put != nil {
		oapiPathItem.Put = ParseOperation(item.Put)
	}

	// You could add similar logic for other HTTP methods like Head, Patch, Trace, Options
	// if your PathItem struct were extended to include them.

	return oapiPathItem
}

// ParseOperation converts your custom Operation struct to openapi3.Operation.
func ParseOperation(op *Operation) *openapi3.Operation {
	if op == nil {
		return nil
	}

	oapiOp := &openapi3.Operation{
		Tags:        op.Tags,
		OperationID: op.OperationID,
		Description: op.Description,
	}

	// Parse Parameters
	if len(op.Parameters) > 0 {
		oapiOp.Parameters = parseParameters(op.Parameters)
	}

	// Parse RequestBody
	if op.RequestBody != nil {
		oapiOp.RequestBody = parseRequestBodyRef(op.RequestBody)
	}

	// Parse Responses
	if len(op.Responses) > 0 {
		oapiOp.Responses = parseResponses(op.Responses)
	}

	return oapiOp
}

// parseParameters converts a slice of your custom ParameterRef to openapi3.Parameters.
func parseParameters(params Parameters) openapi3.Parameters {
	oapiParams := make(openapi3.Parameters, 0, len(params))
	for _, paramRef := range params {
		oapiParams = append(oapiParams, parseParameterRef(&paramRef)) // Pass address as it's a struct
	}
	return oapiParams
}

// parseParameterRef converts your custom ParameterRef to openapi3.ParameterRef.
func parseParameterRef(paramRef *ParameterRef) *openapi3.ParameterRef {
	if paramRef == nil {
		return nil
	}

	oapiParamRef := &openapi3.ParameterRef{}
	if paramRef.Ref != "" {
		oapiParamRef.Ref = paramRef.Ref
	} else if paramRef.Value != nil {
		oapiParamRef.Value = parseParameter(paramRef.Value)
	}
	return oapiParamRef
}

// parseParameter converts your custom Parameter to openapi3.Parameter.
func parseParameter(param *Parameter) *openapi3.Parameter {
	if param == nil {
		return nil
	}

	oapiParam := &openapi3.Parameter{
		Name:        param.Name,
		In:          param.In,
		Description: param.Description,
		Required:    param.Required,
		Example:     param.Example,
	}
	if param.Schema != nil {
		oapiParam.Schema = parseSchemaRef(param.Schema)
	}
	return oapiParam
}

// parseSchemaRef converts your custom SchemaRef to openapi3.SchemaRef.
func parseSchemaRef(schemaRef *SchemaRef) *openapi3.SchemaRef {
	if schemaRef == nil {
		return nil
	}

	oapiSchemaRef := &openapi3.SchemaRef{}
	if schemaRef.Ref != "" {
		oapiSchemaRef.Ref = schemaRef.Ref
	} else if schemaRef.Value != nil {
		oapiSchemaRef.Value = parseSchema(schemaRef.Value)
	}
	return oapiSchemaRef
}

// parseSchema converts your custom Schema to openapi3.Schema.
func parseSchema(schema *Schema) *openapi3.Schema {
	if schema == nil {
		return nil
	}

	var items *openapi3.SchemaRef = nil

	if schema.Items != nil {
		items = parseSchemaRef(schema.Items)
	}
	oapiSchema := &openapi3.Schema{
		Type:        &openapi3.Types{schema.Type},
		Enum:        schema.Enum,    // Direct assignment for []interface{}
		Default:     schema.Default, // Direct assignment for interface{}
		Example:     schema.Example,
		Items:       items,
		Description: schema.Description,
		Pattern:     schema.Pattern,
	}

	if len(schema.Properties) > 0 {
		oapiSchema.Properties = parseSchemas(schema.Properties)
	}

	return oapiSchema
}

// parseSchemas converts your custom Schemas map to openapi3.Schemas.
func parseSchemas(schemas Schemas) openapi3.Schemas {
	oapiSchemas := make(openapi3.Schemas, len(schemas))
	for k, v := range schemas {
		oapiSchemas[k] = parseSchemaRef(v)
	}
	return oapiSchemas
}

// parseRequestBodyRef converts your custom RequestBodyRef to openapi3.RequestBodyRef.
func parseRequestBodyRef(rbRef *RequestBodyRef) *openapi3.RequestBodyRef {
	if rbRef == nil {
		return nil
	}

	oapiRBRef := &openapi3.RequestBodyRef{}
	if rbRef.Ref != "" {
		oapiRBRef.Ref = rbRef.Ref
	} else if rbRef.Value != nil {
		oapiRBRef.Value = parseRequestBody(rbRef.Value)
	}
	return oapiRBRef
}

// parseRequestBody converts your custom RequestBody to openapi3.RequestBody.
func parseRequestBody(rb *RequestBody) *openapi3.RequestBody {
	if rb == nil {
		return nil
	}

	oapiRB := &openapi3.RequestBody{
		Description: rb.Description,
		Required:    rb.Required,
	}
	if len(rb.Content) > 0 {
		oapiRB.Content = parseContent(rb.Content)
	}
	return oapiRB
}

// parseContent converts your custom map[string]*MediaType to openapi3.Content.
func parseContent(content map[string]*MediaType) openapi3.Content {
	oapiContent := make(openapi3.Content, len(content))
	for k, v := range content {
		oapiContent[k] = parseMediaType(v)
	}
	return oapiContent
}

// parseMediaType converts your custom MediaType to openapi3.MediaType.
func parseMediaType(mt *MediaType) *openapi3.MediaType {
	if mt == nil {
		return nil
	}

	oapiMT := &openapi3.MediaType{}
	if mt.Schema != nil {
		oapiMT.Schema = parseSchemaRef(mt.Schema)
	}
	return oapiMT
}

// parseResponses converts your custom map[string]*ResponseRef to openapi3.Responses.
func parseResponses(responses map[string]*ResponseRef) *openapi3.Responses {
	responseOpts := []openapi3.NewResponsesOption{}
	for strStatus, responseRef := range responses {
		statusCode, err := strconv.Atoi(strStatus)
		if err != nil {
			panic(err)
		}
		responseOpts = append(responseOpts, openapi3.WithStatus(statusCode, parseResponseRef(responseRef)))
	}
	oapiResponses := openapi3.NewResponses(responseOpts...)
	return oapiResponses
}

// parseResponseRef converts your custom ResponseRef to openapi3.ResponseRef.
func parseResponseRef(resRef *ResponseRef) *openapi3.ResponseRef {
	if resRef == nil {
		return nil
	}

	oapiResRef := &openapi3.ResponseRef{}
	if resRef.Ref != "" {
		oapiResRef.Ref = resRef.Ref
	} else if resRef.Value != nil {
		oapiResRef.Value = parseResponse(resRef.Value)
	}
	return oapiResRef
}

// parseResponse converts your custom Response to openapi3.Response.
func parseResponse(res *Response) *openapi3.Response {
	if res == nil {
		return nil
	}

	oapiRes := &openapi3.Response{}
	if res.Description != nil {
		oapiRes.Description = res.Description // direct assignment of *string to *string
	}
	if len(res.Content) > 0 {
		oapiRes.Content = parseContent(res.Content)
	}
	return oapiRes
}

type Path struct {
	Path string
	Item PathItem
}

type PathItem struct {
	Get    *Operation
	Post   *Operation
	Delete *Operation
	Put    *Operation
}

type Operation struct {
	Tags        []string
	OperationID string
	Description string
	Deprecated  bool
	Parameters  Parameters
	RequestBody *RequestBodyRef
	Responses   map[string]*ResponseRef
}

type RequestBodyRef struct {
	Ref   string
	Value *RequestBody
}

type RequestBody struct {
	Description string
	Required    bool
	Content     map[string]*MediaType
}

type Parameters []ParameterRef

type ParameterRef struct {
	Ref   string
	Value *Parameter
}

type Parameter struct {
	Name        string
	In          string
	Description string
	Example     any
	Required    bool
	Schema      *SchemaRef
}

type SchemaRef struct {
	Ref   string
	Value *Schema
}

type Schema struct {
	Type        string
	Enum        []any
	Properties  Schemas
	Items       *SchemaRef
	Description string
	Pattern     string
	Default     any
	Example     any
}

type Schemas map[string]*SchemaRef

type ResponseRef struct {
	Ref   string
	Value *Response
}

type Response struct {
	Description *string
	Content     map[string]*MediaType
}

type MediaType struct {
	Schema *SchemaRef
}

func NewSchemaRef(ref string, value *Schema) *SchemaRef {
	return &SchemaRef{
		Ref:   ref,
		Value: value,
	}
}
