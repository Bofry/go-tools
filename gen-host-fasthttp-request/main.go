package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/ast/astutil"
)

const (
	REQUEST_MANAGER_TYPE_NAME string = "RequestManager"
	REQUEST_TYPE_SUFFIX       string = "Request"
	HANDLER_MODULE_NAME       string = "handler"
	HANDLER_ARGS_DIR_PATH     string = "handler/args"
	WEBSOCKET_APP_DIR_PATH    string = "handler/websocket"
	WEBSOCKET_APP_FILE_NAME   string = "app"

	TAG_HIJACK_OPT_NAME string = "@hijack"

	HIJACK_NONE      string = ""
	HIJACK_WEBSOCKET string = "websocket"
)

var (
	osExit        func(int) = os.Exit
	gofile        string
	workdir       string
	appModuleName string
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

	// resolve AST
	for _, node := range f.Decls {
		switch realDecl := node.(type) {
		case *ast.GenDecl:
			for _, spec := range realDecl.Specs {
				switch spec.(type) {
				case *ast.TypeSpec:
					var (
						typeSpec       = spec.(*ast.TypeSpec)
						structTypeName = typeSpec.Name.Name
					)

					// find RequestManager type
					if structTypeName == REQUEST_MANAGER_TYPE_NAME {
						var (
							count int
							err   error
						)

						switch typeSpec.Type.(type) {
						case *ast.StructType:
							structType := typeSpec.Type.(*ast.StructType)
							count, err = generateRequestFiles(structType, HANDLER_MODULE_NAME)
							if err != nil {
								throw(err.Error())
								exit(1)
							}
						}

						if count > 0 {
							// import handler module path
							err := importHandlerModulePath(fset, f)
							if err != nil {
								throw(err.Error())
								exit(1)
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
		exit(1)
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

func generateRequestFiles(structType *ast.StructType, handlerDir string) (n int, err error) {
	var (
		handlerFileMap = make(map[string]string, len(structType.Fields.List))
	)

	for _, field := range structType.Fields.List {
		var (
			hijackType string = ""
		)

		// resolve tag
		if field.Tag != nil && field.Tag.Kind == token.STRING {
			tagLiteral, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				tagLiteral = field.Tag.Value
			}
			tag := reflect.StructTag(tagLiteral)
			val, ok := tag.Lookup(TAG_HIJACK_OPT_NAME)
			if ok {
				hijackType = val
			}
		}

		switch field.Type.(type) {
		case *ast.StarExpr:
			star := field.Type.(*ast.StarExpr)
			ident, ok := star.X.(*ast.Ident)
			if ok {
				requestTypename := ident.Name

				if existedTypeName, ok := handlerFileMap[requestTypename]; ok {
					// NOTE: it have not to be happen.
					throw(fmt.Sprintf("output file '%s' is ambiguous on request type name '%s' and '%s'",
						requestTypename,
						existedTypeName,
						requestTypename))
					exit(1)
				}
				handlerFileMap[requestTypename] = hijackType
			}
		}
	}

	var count int = 0
	if len(handlerFileMap) > 0 {
		if _, err := os.Stat(handlerDir); os.IsNotExist(err) {
			os.Mkdir(handlerDir, os.ModePerm)
		}

		for handlerName, hijackType := range handlerFileMap {
			var (
				filename = getHandlerFileName(handlerName)
			)
			if len(filename) > 0 {
				var (
					writer FileWriter = nil
				)

				fmt.Printf("generating '%s' ...", handlerName)

				file, err := createFile(filename, handlerDir)
				if err != nil {
					if os.IsExist(err) {
						fmt.Println("skipped")
						continue
					} else {
						return count, err
					}
				}
				defer file.Close()

				requestPrefix := strings.TrimSuffix(handlerName, REQUEST_TYPE_SUFFIX)

				switch hijackType {
				case HIJACK_NONE:
					requestGetArgvFile, err := createRequestArgvFile(requestPrefix + "GetArgv")
					if err != nil {
						if os.IsExist(err) {
							fmt.Println("skipped")
						} else {
							return count, err
						}
					}
					defer requestGetArgvFile.Close()

					requestPostArgvFile, err := createRequestArgvFile(requestPrefix + "PostArgv")
					if err != nil {
						if os.IsExist(err) {
							fmt.Println("skipped")
						} else {
							return count, err
						}
					}
					defer requestPostArgvFile.Close()

					writer = &HttpRequestFileWriter{
						AppModuleName:       appModuleName,
						HandlerModuleName:   HANDLER_MODULE_NAME,
						RequestName:         handlerName,
						RequestPrefix:       requestPrefix,
						RequestFile:         file,
						RequestGetArgvFile:  requestGetArgvFile,
						RequestPostArgvFile: requestPostArgvFile,
					}

				case HIJACK_WEBSOCKET:
					requestGetArgvFile, err := createRequestArgvFile(requestPrefix + "GetArgv")
					if err != nil {
						if os.IsExist(err) {
							fmt.Println("skipped")
						} else {
							return count, err
						}
					}
					defer requestGetArgvFile.Close()

					websocketAppModuleName := getWebsocketAppModuleName(handlerName)
					websocketAppFile, err := createWebsocketAppFile(websocketAppModuleName)
					if err != nil {
						if os.IsExist(err) {
							fmt.Println("skipped")
							continue
						} else {
							return count, err
						}
					}
					defer websocketAppFile.Close()

					writer = &WebsocketRequestFileWriter{
						AppModuleName:          appModuleName,
						HandlerModuleName:      HANDLER_MODULE_NAME,
						RequestName:            handlerName,
						RequestPrefix:          requestPrefix,
						WebsocketAppModuleName: websocketAppModuleName,
						RequestFile:            file,
						WebsocketAppFile:       websocketAppFile,
						RequestGetArgvFile:     requestGetArgvFile,
					}
				}

				if writer == nil {
					fmt.Println("skipped")
					continue
				}

				err = writer.Write()
				if err != nil {
					fmt.Println("failed")
				} else {
					fmt.Println("ok")
					count++
				}
			}
		}
	}
	return count, nil
}

// Normalize the handler type name to file name.
// e.g: EchoHandler to echoHandle, XMLHandler to xmlHandler.
func normalizeFileName(typename string) string {
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
	return ""
}

func getHandlerFileName(handlerName string) string {
	if strings.HasSuffix(handlerName, REQUEST_TYPE_SUFFIX) && len(handlerName) > len(REQUEST_TYPE_SUFFIX) {
		return normalizeFileName(handlerName)
	}
	return ""
}

func getWebsocketAppModuleName(handlerName string) string {
	name := handlerName[:len(handlerName)-len(REQUEST_TYPE_SUFFIX)]
	return strings.ToLower(name)
}

func createFile(filename string, handlerDir string) (*os.File, error) {
	path := filepath.Join(handlerDir, filename+".go")

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.Create(path)
		}
		return nil, err
	}
	return nil, os.ErrExist
}

func createRequestArgvFile(requestArgvName string) (*os.File, error) {
	argvDir := HANDLER_ARGS_DIR_PATH
	if err := os.MkdirAll(argvDir, os.ModePerm); err != nil {
		return nil, err
	}

	return createFile(requestArgvName, argvDir)
}

func createWebsocketAppFile(websocketAppModuleName string) (*os.File, error) {
	websocketAppDir := path.Join(WEBSOCKET_APP_DIR_PATH, websocketAppModuleName)
	if err := os.MkdirAll(websocketAppDir, os.ModePerm); err != nil {
		return nil, err
	}

	return createFile(WEBSOCKET_APP_FILE_NAME, websocketAppDir)
}

func importHandlerModulePath(fset *token.FileSet, f *ast.File) error {

	handlerModulePath := appModuleName + "/" + HANDLER_MODULE_NAME

	ok := astutil.AddNamedImport(fset, f, ".", handlerModulePath)
	if ok {
		stream, err := os.OpenFile(gofile, os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			return err
		}
		defer stream.Close()

		err = printer.Fprint(stream, fset, f)
		if err != nil {
			return err
		}
	}
	return nil
}

func getAppModuleName() (string, error) {
	goModBytes, err := os.ReadFile("go.mod")
	if err != nil {
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
