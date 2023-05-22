package main

import (
	"fmt"
	"io"
	"text/template"
)

const (
	ARGV_ASSERTOR_FILE_TEMPLATE = `package {{.PackageName}}

import (
	arg "github.com/Bofry/arg"
{{- block "imports" .Imports}}{{range .}}
	{{if .Name}}{{printf "%s %q" .Name .Path}}{{else}}{{printf "%q" .Path}}{{end}}{{end}}
{{- end}}
)
`

	ARGV_ASSERTOR_TYPE_TEMPLATE = `
type {{.Name}} struct {
	argv  *{{.SourceTypeName}}
}

func (argv *{{.SourceTypeName}}) Assertor() *{{.Name}} {
	return &{{.Name}}{
		argv: argv,
	}
}
`

	ARGV_NONE_ASSERTION_TEMPLATE = ""

	ARGV_STRING_ASSERTION_TEMPLATE = `
func (assertor *{{.TypeName}}) {{.Name}}(validators ...arg.StringValidator) error {
	return arg.Strings.Assert({{.ArgvFieldTypeStar}}assertor.argv.{{.Name}}, {{printf "%q" .Tag}},
		validators...,
	)
}
`

	ARGV_NUMBER_ASSERTION_TEMPLATE = `
func (assertor *{{.TypeName}}) {{.Name}}(validators ...arg.NumberValidator) error {
	return arg.Numbers.Assert({{.ArgvFieldTypeStar}}assertor.argv.{{.Name}}, {{printf "%q" .Tag}},
		validators...,
	)
}
`

	ARGV_FLOAT_ASSERTION_TEMPLATE = `
func (assertor *{{.TypeName}}) {{.Name}}(validators ...arg.FloatValidator) error {
	return arg.Floats.Assert({{.ArgvFieldTypeStar}}assertor.argv.{{.Name}}, {{printf "%q" .Tag}},
		validators...,
	)
}
`

	ARGV_INT_ASSERTION_TEMPLATE = `
func (assertor *{{.TypeName}}) {{.Name}}(validators ...arg.IntValidator) error {
	return arg.Ints.Assert(int64({{.ArgvFieldTypeStar}}assertor.argv.{{.Name}}), {{printf "%q" .Tag}},
		validators...,
	)
}
`

	ARGV_VALUE_ASSERTION_TEMPLATE = `
func (assertor *{{.TypeName}}) {{.Name}}(validators ...arg.ValueValidator) error {
	return arg.Values.Assert({{.ArgvFieldTypeStar}}assertor.argv.{{.Name}}, {{printf "%q" .Tag}},
		validators...,
	)
}
`

	ARGV_IP_ASSERTION_TEMPLATE = `
func (assertor *{{.TypeName}}) {{.Name}}(validators ...arg.IPValidator) error {
	return arg.IPs.Assert({{.ArgvFieldTypeStar}}assertor.argv.{{.Name}}, {{printf "%q" .Tag}},
		validators...,
	)
}
`
)

var (
	FieldAssertionMap map[string]string = map[string]string{
		BOOL_ASSERTION_TYPE:   ARGV_NONE_ASSERTION_TEMPLATE,
		STRING_ASSERTION_TYPE: ARGV_STRING_ASSERTION_TEMPLATE,
		INT_ASSERTION_TYPE:    ARGV_INT_ASSERTION_TEMPLATE,
		FLOAT_ASSERTION_TYPE:  ARGV_FLOAT_ASSERTION_TEMPLATE,
		NUMBER_ASSERTION_TYPE: ARGV_NUMBER_ASSERTION_TEMPLATE,
		VALUE_ASSERTION_TYPE:  ARGV_VALUE_ASSERTION_TEMPLATE,
		IP_ASSERTION_TYPE:     ARGV_IP_ASSERTION_TEMPLATE,
	}

	AssertorTypeTemplate   *template.Template
	ValueAssertionTemplate *template.Template
	FileDirectiveTemplage  *template.Template
)

func init() {
	{
		tmpl, err := template.New("AssertorFile").Parse(ARGV_ASSERTOR_FILE_TEMPLATE)
		if err != nil {
			panic(err)
		}
		FileDirectiveTemplage = tmpl
	}

	{
		tmpl, err := template.New("AssertorType").Parse(ARGV_ASSERTOR_TYPE_TEMPLATE)
		if err != nil {
			panic(err)
		}
		AssertorTypeTemplate = tmpl
	}

	{
		var (
			tmpl *template.Template
			err  error
		)
		tmpl = template.New("ValueAssertion")
		for key, content := range FieldAssertionMap {
			_, err = tmpl.New(key).Parse(content)
			if err != nil {
				panic(err)
			}
		}
		ValueAssertionTemplate = tmpl
	}
}

type AssertorFileWriter struct {
	fileDirectiveTemplage  *template.Template
	assertorTypeTemplate   *template.Template
	valueAssertionTemplate *template.Template
}

func NewAssertorFileWriter() *AssertorFileWriter {
	return &AssertorFileWriter{
		fileDirectiveTemplage:  FileDirectiveTemplage,
		assertorTypeTemplate:   AssertorTypeTemplate,
		valueAssertionTemplate: ValueAssertionTemplate,
	}
}

func (w *AssertorFileWriter) Write(writer io.Writer, file *AssertorFile) error {
	// write package name and imports
	err := w.fileDirectiveTemplage.Execute(writer, file)
	if err != nil {
		panic(err)
	}

	// write types
	for _, t := range file.Types {
		if t != nil {
			err = w.WriteType(writer, t)
			if err != nil {
				panic(err)
			}
		}
	}
	return nil
}

func (w *AssertorFileWriter) WriteType(writer io.Writer, t *AssertorType) error {
	// write type definition and get asserter method
	err := w.assertorTypeTemplate.Execute(writer, t)
	if err != nil {
		panic(err)
	}

	// write assertions
	for _, field := range t.Assertions {
		if field != nil {
			err := w.WriteValueAssertion(writer, field)
			if err != nil {
				panic(err)
			}
		}
	}
	return nil
}

func (w *AssertorFileWriter) WriteValueAssertion(writer io.Writer, f *AssertorValueAssertion) error {
	tmpl := w.valueAssertionTemplate.Lookup(f.Type)
	if tmpl == nil {
		panic(fmt.Errorf("unknown templage %q", f.Type))
	}
	return tmpl.Execute(writer, f)
}