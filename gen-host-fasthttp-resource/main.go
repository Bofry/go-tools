package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/ast/astutil"
)

const (
	RESOURCE_MANAGER_TYPE_NAME string = "ResourceManager"
	RESOURCE_SUBMODULE_NAME    string = "resource"
	RESOURCE_TYPE_SUFFIX       string = "Resource"

	RESOURCE_FILE_TEMPLATE string = `package resource

import (
	"github.com/valyala/fasthttp"
	"github.com/Bofry/host-fasthttp/response"
)

type %[1]s struct {}

func (r *%[1]s) Ping(ctx *fasthttp.RequestCtx) {
	response.Success(ctx, "text/plain", []byte("PONG"))
}
`
)

var (
	gofile string
)

func main() {
	if gofile == "" {
		gofile = os.Getenv("GOFILE")
		if gofile == "" {
			throw("No file to parse.")
			os.Exit(1)
		}
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, gofile, nil, parser.ParseComments)
	if err != nil {
		throw(err.Error())
		os.Exit(1)
	}

	for _, node := range f.Decls {
		switch node.(type) {

		case *ast.GenDecl:
			genDecl := node.(*ast.GenDecl)
			for _, spec := range genDecl.Specs {
				switch spec.(type) {
				case *ast.TypeSpec:
					typeSpec := spec.(*ast.TypeSpec)

					structTypeName := typeSpec.Name.Name
					if structTypeName == RESOURCE_MANAGER_TYPE_NAME {
						var (
							count int
							err   error
						)

						switch typeSpec.Type.(type) {
						case *ast.StructType:
							structType := typeSpec.Type.(*ast.StructType)
							count, err = generateResourceFiles(structType, RESOURCE_SUBMODULE_NAME)
							if err != nil {
								throw(err.Error())
								os.Exit(1)
							}
						}

						if count > 0 {
							// import resource module path
							err := importResourceModulePath(fset, f)
							if err != nil {
								throw(err.Error())
								os.Exit(1)
							}
						}
						break
					}
				}
			}
		}
	}

	if err := execCmd("go", "mod", "tidy"); err != nil {
		throw(err.Error())
		os.Exit(1)
	}

	if err := execCmd("gofmt", "-w", gofile); err != nil {
		throw(err.Error())
		os.Exit(1)
	}
}

func throw(err string) {
	fmt.Fprintln(os.Stderr, err)
}

func generateResourceFiles(structType *ast.StructType, resourceDir string) (n int, err error) {
	var (
		resourceFileNames = make(map[string]string, len(structType.Fields.List))
	)

	for _, field := range structType.Fields.List {
		switch field.Type.(type) {
		case *ast.StarExpr:
			star := field.Type.(*ast.StarExpr)
			i, ok := star.X.(*ast.Ident)
			if ok {
				typename := i.Name

				filename := resolveResourceFileName(typename)
				if len(filename) > 0 {
					if resourceType, ok := resourceFileNames[filename]; ok {
						throw(fmt.Sprintf("output file '%s' is ambiguous on resource type name '%s' and '%s'",
							filename,
							resourceType,
							typename))
						os.Exit(1)
					}
					resourceFileNames[filename] = typename
				}
			}
		}
	}

	var count int = 0
	if len(resourceFileNames) > 0 {
		if _, err := os.Stat(resourceDir); os.IsNotExist(err) {
			os.Mkdir(resourceDir, os.ModePerm)
		}

		for filename, typename := range resourceFileNames {

			ok, err := writeResourceFile(filename, typename, resourceDir)
			if err != nil {
				return count, err
			}
			if ok {
				count++
			}
		}
	}
	return count, nil
}

// Resolve the resource type name to file name.
// e.g: EchoResource to echoResource, XMLResource to xmlResource.
func resolveResourceFileName(typename string) string {
	if strings.HasSuffix(typename, RESOURCE_TYPE_SUFFIX) {
		var (
			runes  = []rune(typename)
			length = len(runes)
		)

		if ch := runes[0]; unicode.IsUpper(rune(ch)) && unicode.IsLetter(ch) {
			var pos int = 0
			for i := 0; i < length; i++ {
				if unicode.IsUpper(runes[i]) && unicode.IsLower(runes[i+1]) {
					pos = i
					break
				}
			}
			if pos == 0 {
				pos++
			}
			return strings.ToLower(string(runes[:pos])) + string(runes[pos:])
		}
	}
	return ""
}

func writeResourceFile(filename, typename string, resourceDir string) (bool, error) {
	path := filepath.Join(resourceDir, filename+".go")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return false, err
		}
		err = writeResouceContent(f, typename)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func writeResouceContent(file *os.File, typename string) error {
	_, err := fmt.Fprintf(file,
		RESOURCE_FILE_TEMPLATE, typename)

	return err
}

func importResourceModulePath(fset *token.FileSet, f *ast.File) error {
	modulePath, err := getModulePath()
	if err != nil {
		throw(err.Error())
		os.Exit(1)
	}
	resourceModulePath := modulePath + "/" + RESOURCE_SUBMODULE_NAME
	ok := astutil.AddNamedImport(fset, f, ".", resourceModulePath)
	if ok {
		stream, err := os.OpenFile(gofile, os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return err
		}
		err = printer.Fprint(stream, fset, f)
		if err != nil {
			return err
		}
	}
	return nil
}

func getModulePath() (string, error) {
	goModBytes, err := ioutil.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	modName := modfile.ModulePath(goModBytes)

	return modName, nil
}

func execCmd(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
