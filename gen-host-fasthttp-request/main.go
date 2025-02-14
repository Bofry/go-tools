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
	HANDLER_ARGS_DIR_PATH     string = "args"
	WEBSOCKET_APP_DIR_PATH    string = "websocket"
	WEBSOCKET_APP_FILE_NAME   string = "app"

	TAG_SKIP_OPT_NAME   string = "@skip"
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
							count       int
							err         error
							subPackages []string
						)

						switch typeSpec.Type.(type) {
						case *ast.StructType:
							structType := typeSpec.Type.(*ast.StructType)
							count, subPackages, err = generateRequestFiles(structType, HANDLER_MODULE_NAME)
							if err != nil {
								throw(err.Error())
								exit(1)
							}
						}

						if count > 0 {
							// import handler module path
							err := importHandlerModulePath(fset, f, subPackages)
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

func generateRequestFiles(structType *ast.StructType, handlerDir string) (n int, subPackages []string, err error) {
	type fileOpt struct {
		packageName string
		hijackType  string
		requestName string
	}

	var (
		handlerFileMap    = make(map[string]fileOpt, len(structType.Fields.List))
		handlerPackageMap = make(map[string]bool)
	)

	for _, field := range structType.Fields.List {
		var (
			opt fileOpt
		)

		// resolve tag
		if field.Tag != nil && field.Tag.Kind == token.STRING {
			tagLiteral, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				tagLiteral = field.Tag.Value
			}
			tag := reflect.StructTag(tagLiteral)

			// has @skip
			{
				val, ok := tag.Lookup(TAG_SKIP_OPT_NAME)
				if ok {
					if len(val) == 0 || val == "on" {
						continue
					}
				}
			}
			// has @hijack?
			{
				val, ok := tag.Lookup(TAG_HIJACK_OPT_NAME)
				if ok {
					opt.hijackType = val
				}
			}
		}

		switch field.Type.(type) {
		case *ast.StarExpr:
			star := field.Type.(*ast.StarExpr)

			switch star.X.(type) {
			case *ast.SelectorExpr:
				sel := star.X.(*ast.SelectorExpr)

				opt.requestName = sel.Sel.Name
				// parse package name
				{
					ident, ok := sel.X.(*ast.Ident)
					if !ok {
						// NOTE: it have not to be happen.
						throw(fmt.Sprintf("cannot resolve request package name at '%d'", star.X.Pos()))
						exit(1)
					}
					opt.packageName = ident.Name
				}

				// generate requestTypename
				requestTypename := opt.packageName + "." + opt.requestName
				// duplicated request name?
				if _, ok := handlerFileMap[requestTypename]; ok {
					continue
				}
				handlerFileMap[requestTypename] = opt
			case *ast.Ident:
				ident := star.X.(*ast.Ident)

				opt.requestName = ident.Name

				// generate requestTypename
				requestTypename := ident.Name
				// duplicated request name?
				if _, ok := handlerFileMap[requestTypename]; ok {
					continue
				}
				handlerFileMap[requestTypename] = opt
			}
		}
	}

	var count int = 0
	if len(handlerFileMap) > 0 {
		if _, err := os.Stat(handlerDir); os.IsNotExist(err) {
			os.Mkdir(handlerDir, os.ModePerm)
		}

		for name, opt := range handlerFileMap {
			var (
				packageDir   = handlerDir
				packageName  = handlerDir
				filename     = getHandlerFileName(opt.requestName)
				isSubPackage = false
			)

			if len(opt.packageName) > 0 {
				packageDir = path.Join(handlerDir, opt.packageName)
				if _, err := os.Stat(packageDir); os.IsNotExist(err) {
					os.Mkdir(packageDir, os.ModePerm)
				}
				packageName = opt.packageName
				isSubPackage = true

				// collect sub package name as wall
				handlerPackageMap[opt.packageName] = true
			}

			if len(filename) > 0 {
				var (
					writer FileWriter = nil
				)

				fmt.Printf("generating '%s' ...", name)

				file, err := createFile(filename, packageDir)
				if err != nil {
					if os.IsExist(err) {
						fmt.Println("skipped")
						continue
					} else {
						return 0, nil, err
					}
				}
				defer file.Close()

				requestPrefix := strings.TrimSuffix(opt.requestName, REQUEST_TYPE_SUFFIX)
				requestArgFilenamePrefix := normalizeFileName(requestPrefix)

				switch opt.hijackType {
				case HIJACK_NONE:
					requestGetArgvFile, err := createRequestArgvFile(packageDir, requestArgFilenamePrefix+"GetArgv")
					if err != nil {
						if os.IsExist(err) {
							fmt.Println("skipped")
						} else {
							return 0, nil, err
						}
					}
					defer requestGetArgvFile.Close()

					requestPostArgvFile, err := createRequestArgvFile(packageDir, requestArgFilenamePrefix+"PostArgv")
					if err != nil {
						if os.IsExist(err) {
							fmt.Println("skipped")
						} else {
							return 0, nil, err
						}
					}
					defer requestPostArgvFile.Close()

					writer = &HttpRequestFileWriter{
						AppModuleName:       appModuleName,
						RequestPackageName:  packageName,
						RequestName:         opt.requestName,
						RequestPrefix:       requestPrefix,
						IsSubRequestPackage: isSubPackage,
						RequestFile:         file,
						RequestGetArgvFile:  requestGetArgvFile,
						RequestPostArgvFile: requestPostArgvFile,
					}

				case HIJACK_WEBSOCKET:
					requestGetArgvFile, err := createRequestArgvFile(packageDir, requestArgFilenamePrefix+"GetArgv")
					if err != nil {
						if os.IsExist(err) {
							fmt.Println("skipped")
						} else {
							return 0, nil, err
						}
					}
					defer requestGetArgvFile.Close()

					websocketAppModuleName := getWebsocketAppModuleName(opt.requestName)
					websocketAppFile, err := createWebsocketAppFile(packageDir, websocketAppModuleName)
					if err != nil {
						if os.IsExist(err) {
							fmt.Println("skipped")
							continue
						} else {
							return 0, nil, err
						}
					}
					defer websocketAppFile.Close()

					writer = &WebsocketRequestFileWriter{
						AppModuleName:          appModuleName,
						RequestPackageName:     packageName,
						RequestName:            opt.requestName,
						RequestPrefix:          requestPrefix,
						IsSubRequestPackage:    isSubPackage,
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

	// export subPackages
	if len(handlerPackageMap) > 0 {
		subPackages = make([]string, 0, len(handlerPackageMap))
		for k, _ := range handlerPackageMap {
			subPackages = append(subPackages, k)
		}
	}
	return count, subPackages, nil
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

func createRequestArgvFile(dir, requestArgvName string) (*os.File, error) {
	argvDir := path.Join(dir, HANDLER_ARGS_DIR_PATH)
	if err := os.MkdirAll(argvDir, os.ModePerm); err != nil {
		return nil, err
	}

	return createFile(requestArgvName, argvDir)
}

func createWebsocketAppFile(dir, websocketAppModuleName string) (*os.File, error) {
	websocketAppDir := path.Join(dir, WEBSOCKET_APP_DIR_PATH, websocketAppModuleName)
	if err := os.MkdirAll(websocketAppDir, os.ModePerm); err != nil {
		return nil, err
	}

	return createFile(WEBSOCKET_APP_FILE_NAME, websocketAppDir)
}

func importHandlerModulePath(fset *token.FileSet, f *ast.File, subPackages []string) error {
	var (
		shouldUpdateImpoty bool
	)

	{
		handlerModulePath := appModuleName + "/" + HANDLER_MODULE_NAME
		ok := astutil.AddNamedImport(fset, f, ".", handlerModulePath)
		if ok {
			shouldUpdateImpoty = ok
		}
	}
	for _, name := range subPackages {
		handlerModulePath := appModuleName + "/" + HANDLER_MODULE_NAME + "/" + name
		ok := astutil.AddImport(fset, f, handlerModulePath)
		if ok {
			shouldUpdateImpoty = ok
		}
	}
	if shouldUpdateImpoty {
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
