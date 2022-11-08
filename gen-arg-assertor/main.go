package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

const (
	ARGV_TYPE_SUFFIX          string = "Argv"
	ARGV_FIELD_TAG_DIRECTIVE  string = "^"
	ARGV_STRUCT_TAG_DIRECTIVE string = "tag"
	ARGV_ASSERTOR_TYPE_SUFFIX string = "Assertor"
)

var (
	gofile string
)

type (
	StructField struct {
		Name          string
		Tag           string
		Type          string
		TypeStar      string
		AssertionType string
	}
)

func main() {
	var err error

	if gofile == "" {
		gofile = os.Getenv("GOFILE")
		if gofile == "" {
			throw("No file to parse.")
			os.Exit(1)
		}
	}

	// Type-check the package.
	// We create an empty map for each kind of input
	// we're interested in, and Check populates them.
	info := types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	// get *ast.File
	f, err := parseAst(gofile, &info)
	if err != nil {
		throw(err.Error())
		os.Exit(1)
	}

	file := new(AssertorFile)
	if err = fillAssertorFile(file, f, &info); err != nil {
		throw(err.Error())
		os.Exit(1)
	}

	writer := NewAssertorFileWriter()
	outputFile := filepath.Join(extractfilename(gofile) + ARGV_ASSERTOR_TYPE_SUFFIX + "_gen.go")

	if err = writeFile(outputFile, writer, file); err != nil {
		throw(err.Error())
		os.Exit(1)
	}

	if err = execCmd("gofmt", "-w", outputFile); err != nil {
		throw(err.Error())
		os.Exit(1)
	}
}

func writeFile(filename string, writer *AssertorFileWriter, file *AssertorFile) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		f, err := os.Create(filename)
		if err != nil {
			return err
		}

		return writer.Write(f, file)
	}
	return nil
}

func extractfilename(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

func parseAst(gofile string, refInfo *types.Info) (*ast.File, error) {
	// FIXME: check exist and dir/file
	pkgdir := filepath.Dir(gofile)
	pkgfile := filepath.Clean(gofile)

	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, pkgdir,
		func(fi fs.FileInfo) bool {
			return true
		},
		parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var f *ast.File
	var files []*ast.File
	for _, pkg := range pkgs {
		if file, ok := pkg.Files[pkgfile]; ok {
			f = file
		}
		for _, file := range pkg.Files {
			files = append(files, file)
		}
	}

	if refInfo != nil {
		var conf types.Config = types.Config{
			DisableUnusedImportCheck: true,
			IgnoreFuncBodies:         true,
			Importer:                 importer.ForCompiler(fset, "source", nil),
		}
		_, err = conf.Check(f.Name.Name, fset, files, refInfo)
		if err != nil {
			return nil, err
		}
	}

	return f, nil
}

func parseStructTagNamesAnnotation(key string, structType *ast.StructType, comments []*ast.CommentGroup) []string {
	var (
		attrPos token.Pos = structType.Pos()
		attrEnd token.Pos = structType.Fields.Opening
	)

	for _, comment := range comments {
		if attrPos <= comment.Pos() && attrEnd >= comment.End() {

			var tag string = comment.Text()

			for tag != "" {
				// Skip leading space.
				i := 0
				for i < len(tag) && tag[i] == ' ' {
					i++
				}
				tag = tag[i:]
				if tag == "" {
					break
				}

				i = 0
				for i < len(tag) && tag[i] > ' ' && tag[i] != '=' {
					i++
				}

				if i == 0 || i+1 >= len(tag) || tag[i] != '=' {
					var nlchar byte
					for i < len(tag) {
						if tag[i] == '\r' || tag[i] == '\n' {
							nlchar = tag[i]
							i++
							continue
						}
						if nlchar != 0 {
							break
						}
						i++
					}
					if nlchar != 0 {
						tag = tag[i:]
						continue
					}
				}
				name := string(tag[:i])
				tag = tag[i+1:]

				for i < len(tag) && tag[i] != '\n' && tag[i] != ' ' {
					i++
				}
				if i >= len(tag) {
					break
				}
				text := string(tag[:i])

				if key == name {
					return strings.Split(text, ",")
				}
			}

			break
		}
	}
	return nil
}

func findAssertionType(ident *ast.Ident, info *types.Info) string {
	var (
		signatures []string = []string{
			ident.Name,
		}
	)

	// append other candicated type signatures
	if obj := info.Uses[ident]; obj != nil {
		signatures = append(signatures,
			obj.Type().String(),
			obj.Type().Underlying().String())
	}

	// find suitable AssertionType
	for _, signature := range signatures {
		switch signature {
		case "encoding/json.Number":
			return NUMBER_ASSERTION_TYPE
		case "net.IP":
			return IP_ASSERTION_TYPE
		case "string":
			return STRING_ASSERTION_TYPE
		case "int", "int8", "int16", "int32", "int64":
			return INT_ASSERTION_TYPE
		case "float32", "float64":
			return FLOAT_ASSERTION_TYPE
		case "bool":
			return BOOL_ASSERTION_TYPE
		}

	}
	return VALUE_ASSERTION_TYPE
}

func execCmd(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func throw(err string) {
	fmt.Fprintln(os.Stderr, err)
}

func fillAssertorFile(ref *AssertorFile, f *ast.File, info *types.Info) error {
	var assertorTypes []*AssertorType

	for _, node := range f.Decls {
		switch node.(type) {
		case *ast.GenDecl:
			genDecl := node.(*ast.GenDecl)
			for _, spec := range genDecl.Specs {
				switch specExpr := spec.(type) {
				case *ast.TypeSpec:
					switch typeExpr := specExpr.Type.(type) {
					case *ast.StructType:
						var (
							structName = specExpr.Name.Name
						)

						if strings.HasSuffix(structName, ARGV_TYPE_SUFFIX) {
							assertorType := &AssertorType{
								SourceTypeName: structName,
								Name:           structName + ARGV_ASSERTOR_TYPE_SUFFIX,
							}

							err := fillAssertorType(assertorType, typeExpr, info, f.Comments)
							if err != nil {
								return err
							}

							assertorTypes = append(assertorTypes, assertorType)
						}
					default:
						// ignore
					}
				}
			}
		}
	}

	ref.PackageName = f.Name.Name
	ref.Types = assertorTypes

	return nil
}

func fillAssertorType(ref *AssertorType, structType *ast.StructType, info *types.Info, comments []*ast.CommentGroup) error {
	var (
		assertions []*AssertorValueAssertion
		tagnames   = parseStructTagNamesAnnotation(ARGV_STRUCT_TAG_DIRECTIVE, structType, comments)
	)

	// traverse all fields
	for _, field := range structType.Fields.List {
		assertion := &AssertorValueAssertion{
			TypeName: ref.Name,
		}

		err := fillAssertorValueAssertion(assertion, field, info, tagnames)
		if err != nil {
			return err
		}

		// append assertion
		assertions = append(assertions, assertion)
	}

	// exports
	ref.Assertions = assertions

	return nil
}

func fillAssertorValueAssertion(ref *AssertorValueAssertion, field *ast.Field, info *types.Info, tagnames []string) error {
	var (
		fieldName     string = field.Names[0].Name
		fieldTag      string
		fieldType     string
		fieldTypeStar string
		assertionType string
	)

	// resolve type
	{
		var (
			expr = field.Type
			stop = false
		)

		for !stop {
			switch typedExpr := expr.(type) {
			case *ast.Ident:
				fieldType = typedExpr.Name

				assertionType = findAssertionType(typedExpr, info)
				stop = true
			case *ast.StarExpr:
				expr = typedExpr.X

				fieldTypeStar += "*"
			case *ast.SelectorExpr:
				{
					identExpr := typedExpr.X.(*ast.Ident)
					fieldType = identExpr.Name + "." + typedExpr.Sel.Name
				}

				assertionType = findAssertionType(typedExpr.Sel, info)
				stop = true
			default:
				return fmt.Errorf("unsupported field type %T", expr)
			}
		}
	}

	// resolve tag
	{
		if field.Tag.Kind == token.STRING {
			content, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				return fmt.Errorf("bad struct tag %q at %d", field.Tag.Value, field.Tag.ValuePos)
			}
			tag := reflect.StructTag(content)

			// use field specifiec tag name
			if tagname, ok := tag.Lookup(ARGV_FIELD_TAG_DIRECTIVE); ok {
				fieldTag = tag.Get(tagname)
			} else {
				// use default struct tag name
				for _, tagname := range tagnames {
					// found !
					if v, ok := tag.Lookup(tagname); ok {
						fieldTag = v
						break
					}
				}
			}

		}
	}

	// exports
	ref.Name = fieldName
	ref.Tag = fieldTag
	ref.Type = assertionType
	ref.ArgvFieldType = fieldType
	ref.ArgvFieldTypeStar = fieldTypeStar

	return nil
}
