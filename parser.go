package openapi3Struct

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/itchyny/json2yaml"
	"golang.org/x/tools/go/packages"
)

const (
	openapiSchemaDecoration = "oapi:schema"
	swaggerSchemaDecoration = "swagger:model"
)

type Path struct {
	Path string
	Item openapi3.PathItem
}

type Parser struct {
	T           openapi3.T
	packagePath []string
	paths       []Path
}

type Option func(p Parser) Parser

func NewParser(t openapi3.T, options ...Option) *Parser {
	p := Parser{
		T: t,
	}
	for _, option := range options {
		p = option(p)
	}

	return &p
}

func WithPackagePaths(paths []string) Option {
	return func(p Parser) Parser {
		p.packagePath = paths
		return p
	}
}

func (p *Parser) AddPath(path Path) {
	if p.T.Paths == nil {
		p.T.Paths = &openapi3.Paths{}
	}
	existingPath := p.T.Paths.Find(path.Path)
	// TODO improve this to add checks for alscl kinds of optional fields
	if existingPath == nil {
		p.T.Paths.Set(path.Path, &path.Item)
		return
	}
	if path.Item.Delete != nil {
		existingPath.Delete = path.Item.Delete
	}
	if path.Item.Head != nil {
		existingPath.Head = path.Item.Head
	}
	if path.Item.Post != nil {
		existingPath.Post = path.Item.Post
	}
	if path.Item.Get != nil {
		existingPath.Get = path.Item.Get
	}
	if path.Item.Put != nil {
		existingPath.Put = path.Item.Put
	}

	p.T.Paths.Set(path.Path, existingPath)
}

func (p *Parser) SaveYamlToFile(path string) error {
	json, err := p.T.MarshalJSON()
	if err != nil {
		return err
	}
	result := bytes.NewBuffer([]byte{})
	err = json2yaml.Convert(result, bytes.NewBuffer(json))
	if err != nil {
		return err
	}

	return os.WriteFile(path, result.Bytes(), 0644)
}

func (p *Parser) SaveJsonToFile(path string) error {
	json, err := p.T.MarshalJSON()
	if err != nil {
		return err
	}

	return os.WriteFile(path, json, 0644)
}

// Validate resolves refs and validates schema
func (p *Parser) Validate(ctx context.Context) error {
	loader := openapi3.NewLoader()
	err := loader.ResolveRefsIn(&p.T, nil)
	if err != nil {
		return err
	}

	err = p.T.Validate(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Parser) ParseSchemasFromStructs() error {
	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes}
	pkgs, err := packages.Load(cfg, p.packagePath...)
	if err != nil {
		return err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return err
	}
	if p.T.Components.Schemas == nil {
		p.T.Components.Schemas = openapi3.Schemas{}
	}

	schemas := walkPackageAndResolveSchemas(pkgs)
	for name, schema := range schemas {
		if _, ok := p.T.Components.Schemas[name]; ok {
			return fmt.Errorf("Generated schema conflict Name=%s", name)
		}

		p.T.Components.Schemas[name] = schema
	}

	return nil
}

func walkPackageAndResolveSchemas(pkgs []*packages.Package) openapi3.Schemas {
	schemas := openapi3.Schemas{}
	declarationMap := map[string]*ast.TypeSpec{}
	//this loop is to collect all the type declarations, this way when we parse star expressions we can resolve them as if they were the actual type
	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			for _, v := range f.Decls {
				switch decl := v.(type) {
				case *ast.GenDecl:
					if !strings.Contains(decl.Doc.Text(), openapiSchemaDecoration) && !strings.Contains(decl.Doc.Text(), swaggerSchemaDecoration) {
						continue
					}
					for _, s := range decl.Specs {
						switch spec := s.(type) {
						case *ast.TypeSpec:
							declarationMap[spec.Name.Name] = spec
						}
					}
				}
			}
		}
	}
	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			for _, v := range f.Decls {
				switch decl := v.(type) {
				case *ast.FuncDecl:
					break
				case *ast.GenDecl:
					if !strings.Contains(decl.Doc.Text(), openapiSchemaDecoration) && !strings.Contains(decl.Doc.Text(), swaggerSchemaDecoration) {
						continue
					}
					for _, s := range decl.Specs {
						doc := ""
						if decl.Doc != nil {
							doc = decl.Doc.Text()
						}
						// TODO: add schema renaming
						name, schema := resolveSchema(schemas, s, doc, declarationMap)
						if name != nil {
							schemas[*name] = openapi3.NewSchemaRef("", &schema)
						}
					}
					break
				case *ast.BadDecl:
					break
				default:
					break
				}
			}
		}
	}
	return schemas
}
