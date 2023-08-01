package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	MODULE_TYPE_NAME string = "Module"

	BOFRY_HOST_APP_PACKAGE_NAME string = `"github.com/Bofry/host/app"`
	EVENT_HANDLER_TYPE_NAME     string = "EventHandler"
	MESSAGE_HANDLER_TYPE_NAME   string = "MessageHandler"
)

var (
	osExit     func(int) = os.Exit
	gofile     string
	workdir    string
	moduleName string
	moduleDir  string

	BOFRY_HOST_APP_IDENT string = "app"
)

func init() {
	flag.StringVar(&gofile, "file", "", "input file")
}

func main() {
	var (
		err error
	)
	flag.Parse()

	if len(gofile) > 0 {
		if fullpath, err := filepath.Abs(gofile); true {
			if err != nil {
				throw("Cannot get work directory.")
				exit(1)
			}

			if dir, file := filepath.Split(fullpath); dir != "." {
				workdir, err = os.Getwd()
				if err != nil {
					throw("Cannot get work directory.")
					exit(1)
				}
				os.Chdir(dir)
				gofile = file
			}
		}
	}

	if gofile == "" {
		gofile = os.Getenv("GOFILE")
		if gofile == "" {
			throw("No file to parse.")
			exit(1)
		}
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
		switch realDecl := node.(type) {
		case *ast.GenDecl:
			for _, spec := range realDecl.Specs {
				switch realSpec := spec.(type) {
				case *ast.ImportSpec:
					if realSpec.Path.Value == BOFRY_HOST_APP_PACKAGE_NAME {
						if realSpec.Name != nil {
							BOFRY_HOST_APP_IDENT = realSpec.Name.Name
						}
					}

				case *ast.ValueSpec:
					structExpr := lookupModule(realSpec)
					if structExpr != nil {
						_, err := generateModuleFiles(structExpr)
						if err != nil {
							throw(err.Error())
							exit(1)
						}
					}
				}
			}
		}
	}

	if err := execCmd("go", "mod", "tidy"); err != nil {
		throw(err.Error())
		exit(1)
	}
	exit(0)

}

func throw(err string) {
	fmt.Fprintln(os.Stderr, err)
}

func exit(code int) {
	osExit(code)
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

func lookupModule(spec *ast.ValueSpec) *ast.StructType {
	// find variant name from var list
	var index int = -1
	for i, ident := range spec.Names {
		if ident.Name == MODULE_TYPE_NAME {
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

func generateModuleFiles(structExpr *ast.StructType) (n int, err error) {
	var (
		handlerFileMap = make(map[string]string)
	)

	for _, field := range structExpr.Fields.List {
		switch realIdent := field.Type.(type) {
		case *ast.SelectorExpr:
			ident, ok := realIdent.X.(*ast.Ident)
			if ok {
				if ident.Name == BOFRY_HOST_APP_IDENT {
					switch realIdent.Sel.Name {
					case EVENT_HANDLER_TYPE_NAME:
						if len(field.Names) > 0 {
							handlerName := field.Names[0].Name
							handlerFileMap[handlerName] = EVENT_HANDLER_TYPE_NAME
						}
					case MESSAGE_HANDLER_TYPE_NAME:
						if len(field.Names) > 0 {
							handlerName := field.Names[0].Name
							handlerFileMap[handlerName] = MESSAGE_HANDLER_TYPE_NAME
						}
					}
				}
			}
		}
	}

	var count int = 0
	for handlerName, typeName := range handlerFileMap {
		var (
			filename = getHandlerFileName(handlerName)
		)

		if len(filename) > 0 {
			var (
				writer FileWriter = nil
			)

			fmt.Printf("generating %s: '%s' ...", typeName, handlerName)

			file, err := createFile(filename)
			if err != nil {
				if os.IsExist(err) {
					fmt.Println("skipped")
					continue
				}
				return count, err
			}
			defer file.Close()

			switch typeName {
			case EVENT_HANDLER_TYPE_NAME:
				writer = &EventHandlerFileWriter{
					PackageName: moduleName,
					EventName:   handlerName,
				}
			case MESSAGE_HANDLER_TYPE_NAME:
				writer = &MessageHanderFileWriter{
					PackageName: moduleName,
					MessageName: handlerName,
				}
			}

			if writer == nil {
				fmt.Println("skipped")
				continue
			}

			err = writer.Write(file)
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

func getHandlerFileName(handlerName string) string {
	part := normalizeHandlerFileName(handlerName)
	if part == "" {
		return ""
	}
	return fmt.Sprintf("app.%s", part)
}

// Normalize the handler name to file name.
// e.g: EchoHandler to echoHandle, XMLHandler to xmlHandler.
func normalizeHandlerFileName(name string) string {
	var (
		runes  = []rune(name)
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
	return ""
}

func createFile(filename string) (*os.File, error) {
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
