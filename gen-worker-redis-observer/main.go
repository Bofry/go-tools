package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/mod/modfile"
)

const (
	GO_MOD_FILE           string = "go.mod"
	MAX_GO_MOD_FILE_DEPTH int    = 3

	MESSAGE_OBSERVER_MANAGER_TYPE_NAME string = "MessageObserverManager"
	MESSAGE_OBSERVER_TYPE_SUFFIX       string = "MessageObserver"
)

var (
	osExit        func(int) = os.Exit
	gofile        string
	workdir       string
	appModuleName string
	moduleName    string
)

func init() {
	flag.StringVar(&gofile, "file", "", "input file")
}

func main() {
	var (
		err error
	)
	flag.Parse()

	if dir, file := path.Split(gofile); dir != "." {
		workdir, err = os.Getwd()
		if err != nil {
			throw("Cannot get work directory.")
			exit(1)
		}
		os.Chdir(dir)
		gofile = file
	}

	if gofile == "" {
		gofile = os.Getenv("GOFILE")
		if gofile == "" {
			throw("No file to parse.")
			exit(1)
		}
	}

	// get module name
	appModuleName, err = getAppModuleName()
	if err != nil {
		throw(err.Error())
		exit(1)
	}

	// parse app.go to AST
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, gofile, nil, parser.ParseComments)
	if err != nil {
		throw(err.Error())
		exit(1)
	}

	moduleName = f.Name.Name

	// resolve AST
	for _, node := range f.Decls {
		switch node.(type) {

		case *ast.GenDecl:
			genDecl := node.(*ast.GenDecl)

			for _, spec := range genDecl.Specs {
				switch realSpec := spec.(type) {
				case *ast.ValueSpec:
					structExpr := lookupMessageObserverManager(realSpec)
					if structExpr != nil {
						_, err := generateMessageObserverFile(structExpr)
						if err != nil {
							throw(err.Error())
							exit(1)
						}
					}

				}
			}
		}
	}

	if err := execCmd("gofmt", "-w", gofile); err != nil {
		throw(err.Error())
		exit(1)
	}
	exit(0)
}

func throw(err string) {
	fmt.Fprintln(os.Stderr, err)
}

func exit(code int) {
	if len(workdir) > 0 {
		os.Chdir(workdir)
	}
	osExit(code)
}

func getAppModuleName() (string, error) {
	var (
		path         = filepath.Join(GO_MOD_FILE)
		attempts int = 0
	)

retry:
	goModBytes, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if attempts < MAX_GO_MOD_FILE_DEPTH {
				path = filepath.Join("../", path)
				attempts++
				goto retry
			}
		}
		return "", err
	}

	modName := modfile.ModulePath(goModBytes)

	return modName, nil
}

func execCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin

	var (
		stdout io.ReadCloser
		stderr io.ReadCloser

		err error
	)

	if stdout, err = cmd.StdoutPipe(); err != nil {
		return err
	}
	if stderr, err = cmd.StderrPipe(); err != nil {
		return err
	}
	reader := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(reader)
	go func() {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	if err = cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

func lookupMessageObserverManager(spec *ast.ValueSpec) *ast.StructType {
	// find variant name from var list
	var index int = -1
	for i, ident := range spec.Names {
		if ident.Name == MESSAGE_OBSERVER_MANAGER_TYPE_NAME {
			index = i
			break
		}
	}

	// not found
	if index < 0 || len(spec.Values) < index {
		return nil
	}

	// get value
	expr := spec.Values[index]
	definition, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil
	}

	structExpr, ok := definition.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	return structExpr
}

func generateMessageObserverFile(structType *ast.StructType) (n int, err error) {
	var (
		fileTypeNameMap = make(map[string]string, len(structType.Fields.List))
	)

	for _, field := range structType.Fields.List {
		switch field.Type.(type) {
		case *ast.StarExpr:
			star := field.Type.(*ast.StarExpr)
			ident, ok := star.X.(*ast.Ident)
			if ok {
				typename := ident.Name

				filename := resolveMessageObserverFileName(typename)
				if len(filename) > 0 {
					if existedTypeName, ok := fileTypeNameMap[filename]; ok {
						// NOTE: it have not to be happen.
						throw(fmt.Sprintf("output file '%s' is ambiguous on MessageObserver type name '%s' and '%s'",
							filename,
							existedTypeName,
							typename))
						exit(1)
					}
					fileTypeNameMap[filename] = typename
				}
			}
		}
	}

	var count int = 0
	if len(fileTypeNameMap) > 0 {
		for RequestFilename, RequestTypename := range fileTypeNameMap {
			fmt.Printf("generating '%s' ...", RequestFilename)

			file, err := createRequestFile(RequestFilename, RequestTypename)
			if err != nil {
				if os.IsExist(err) {
					fmt.Println("skipped")
					continue
				} else {
					return count, err
				}
			}
			defer file.Close()

			metadata := FileMetadata{
				AppModuleName:      appModuleName,
				ObserverModuleName: moduleName,
				ObserverName:       RequestTypename,
			}

			tmpl, err := template.New("").Parse(MESSAGE_OBSERVER_FILE_TEMPLATE)
			if err != nil {
				return count, err
			}

			err = tmpl.Execute(file, metadata)
			if err != nil {
				fmt.Println("failed")
			} else {
				fmt.Println("ok")
				count++
			}
		}
	}
	return count, nil
}

func resolveMessageObserverFileName(typename string) string {
	if strings.HasSuffix(typename, MESSAGE_OBSERVER_TYPE_SUFFIX) {
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

func createRequestFile(filename, typename string) (*os.File, error) {
	path := filepath.Join(filename + ".go")

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.Create(path)
		}
		return nil, err
	}
	return nil, os.ErrExist
}
