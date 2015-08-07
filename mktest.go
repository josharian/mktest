package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"

	"code.google.com/p/go.tools/imports"
)

type Param struct {
	Name string
	Type string
}

type Test struct {
	Name   string
	Params []Param
}

type TestFile struct {
	Pkg   string
	Tests []Test
}

func fmtexpr(fset *token.FileSet, e ast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, e)
	return buf.String()
}

func paramname(field *ast.Field) string {
	if len(field.Names) > 0 {
		return field.Names[0].Name
	}
	return "_"
}

func param(fset *token.FileSet, field *ast.Field) Param {
	return Param{
		Name: paramname(field),
		Type: fmtexpr(fset, field.Type),
	}
}

func Parse(path string) (*TestFile, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	tf := TestFile{Pkg: f.Name.Name}

	for _, decl := range f.Decls {
		fun, ok := decl.(*ast.FuncDecl)
		if !ok || !ast.IsExported(fun.Name.Name) {
			continue
		}

		test := Test{Name: fun.Name.Name}

		// Receiver (for methods)
		if fun.Recv != nil {
			test.Params = append(test.Params, param(fset, fun.Recv.List[0]))
		}

		// Parameters
		for _, field := range fun.Type.Params.List {
			test.Params = append(test.Params, param(fset, field))
		}

		// Return values
		for _, field := range fun.Type.Results.List {
			test.Params = append(test.Params, param(fset, field))
		}

		tf.Tests = append(tf.Tests, test)
	}

	return &tf, nil
}

const skeleton = `
package {{.Pkg}}

import "testing"

{{range .Tests}}
func Test{{.Name}}(t *testing.T) { {{if .Params}}
	cases := []struct{
		{{range .Params}}{{.Name}} {{.Type}}
		{{end}}
	}{
		{
			{{range .Params}}// {{.Name}}: ,
			{{end}} },
	}
	{{end}}

	{{if .Params}}
	for _, tt := range cases {
		_ = tt
	}{{end}}
}
{{end}}
`

var tmpl = template.Must(template.New("test").Parse(skeleton))

func main() {
	file := os.Args[1]
	if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
		log.Fatal("arg must be go file, not test")
	}

	in, err := Parse(file)
	if err != nil {
		log.Fatal(err)
	}
	var buf bytes.Buffer
	tmpl.Execute(&buf, in)
	src, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		log.Fatalf("%s\nfailed to gofmt generated code: %v", src, err)
	}
	file = strings.TrimSuffix(file, ".go") + "_test.go"
	out, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	if _, err := out.Write(src); err != nil {
		log.Fatal(err)
	}
}
