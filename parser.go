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
		p.T.Paths = openapi3.Paths{}
	}
	// TODO improve this to add checks for all kinds of optional fields
	if p.T.Paths[path.Path] == nil {
		p.T.Paths[path.Path] = &path.Item
		return
	}
	if path.Item.Delete != nil {
		p.T.Paths[path.Path].Delete = path.Item.Delete
	}
	if path.Item.Post != nil {
		p.T.Paths[path.Path].Post = path.Item.Post
	}
	if path.Item.Get != nil {
		p.T.Paths[path.Path].Get = path.Item.Get
	}
	if path.Item.Put != nil {
		p.T.Paths[path.Path].Put = path.Item.Put
	}
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
	openapi3.NewLoader().ResolveRefsIn(&p.T, nil)

	err := p.T.Validate(ctx)
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
	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			// fmt.Printf("File %s\n", f.Name)
			for _, v := range f.Decls {
				// fmt.Printf("V %T\n", v)
				switch decl := v.(type) {
				case *ast.FuncDecl:
					// fmt.Printf("FuncDecl %s %v\n", decl.Name.Name, decl.Doc)
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
						name, schema := resolveSchema(schemas, s, doc)
						if name != nil {
							schemas[*name] = openapi3.NewSchemaRef("", &schema)
						}
					}

					break
				case *ast.BadDecl:
					// fmt.Printf("BadDeclypeSpec\n")
					break
				default:
					// fmt.Printf("Unknown %T\n", decl)
					break
				}
			}
		}
	}
	return schemas
}
