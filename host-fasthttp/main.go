package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"golang.org/x/mod/modfile"
)

var (
	osExit func(int) = os.Exit
)

var (
	__FILE_TEMPLATES = map[string]string{
		FILE_APP_GO:                       FILE_APP_GO_TEMPLATE,
		FILE_INTERNAL_DEF_GO:              FILE_INTERNAL_DEF_GO_TEMPLATE,
		FILE_INTERNAL_APP_GO:              FILE_INTERNAL_APP_GO_TEMPLATE,
		FILE_INTERNAL_SERVICE_PROVIDER_GO: FILE_INTERNAL_SERVICE_PROVIDER_GO_TEMPLATE,
		FILE_INTERNAL_EVENT_LOG_GO:        FILE_INTERNAL_EVENT_LOG_GO_TEMPLATE,
		FILE_INTERNAL_LOGGING_SERVICE_GO:  FILE_INTERNAL_LOGGING_SERVICE_GO_TEMPLATE,
		FILE_CONFIG_YAML:                  FILE_CONFIG_YAML_TEMPLATE,
		FILE_CONFIG_LOCAL_YAML:            FILE_CONFIG_LOCAL_YAML_TEMPLATE,
		FILE_GITIGNORE:                    FILE_GITIGNORE_TEMPLATE,
		FILE_SERVICE_NAME:                 FILE_SERVICE_NAME_TEMPLATE,
		FILE_ENV:                          FILE_ENV_TEMPLATE,
		FILE_ENV_SAMPLE:                   FILE_ENV_SAMPLE_TEMPLATE,
		FILE_LOAD_ENV_SH:                  FILE_LOAD_ENV_SH_TEMPLATE,
		FILE_LOAD_ENV_BAT:                 FILE_LOAD_ENV_BAT_TEMPLATE,
		FILE_DOCKERFILE:                   FILE_DOCKERFILE_TEMPLATE,
	}
)

func main() {
	if len(os.Args) < 2 {
		showUsage()
		exit(0)
	}

	argv := os.Args[1]
	switch argv {
	case "init":
		var (
			moduleName string

			err error
		)

		if len(os.Args) > 2 {
			var pos int = 2
			argv = os.Args[pos]

			if strings.HasPrefix(argv, "-") {
				moduleName, err = getModuleName()
				if err != nil {
					throw(err.Error())
					exit(1)
				}
			} else {
				moduleName = argv
				moduleName, err = initModule(moduleName)
				if err != nil {
					throw(err.Error())
					exit(1)
				}
				pos++
			}

			for len(os.Args) > pos {
				argv = os.Args[pos]
				pos++
				switch argv {
				case "-v":
					if len(os.Args) > pos {
						// run go get -v github.com/Bofry/host-fasthttp@<version>
						argv = os.Args[pos]
						pos++
						err = executeCommand("go", "get", "-v", "github.com/Bofry/host-fasthttp@"+argv)
						if err != nil {
							throw(err.Error())
							exit(1)
						}
					}
				default:
					throw(fmt.Sprintf("unknown flag '%s'\n", argv))
					exit(1)
				}
			}
		} else {
			moduleName, err = getModuleName()
			if err != nil {
				throw(err.Error())
				exit(1)
			}
		}

		runtimeVersion := getRuntimeVersion()
		if len(runtimeVersion) == 0 {
			throw("cannot get go version")
		}

		metadata := AppMetadata{
			RuntimeVersion: runtimeVersion,
			AppModuleName:  moduleName,
			AppExeName:     extractAppExeName(moduleName),
		}
		err = initProject(&metadata)
		if err != nil {
			throw(err.Error())
			exit(1)
		}
	case "help", "-h", "--help":
		showUsage()
		exit(0)
	default:
		throw(fmt.Sprintf("unknown command '%s'\n", argv))
		showUsage()
		exit(1)
	}
}

func do(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func throw(err string) {
	fmt.Fprintln(os.Stderr, err)
}

func exit(code int) {
	osExit(code)
}

func showUsage() {
	fmt.Print(`Usage: http-fasthttp COMMAND [ARGS...] [OPTIONS...]

COMMANDS:
  init        create new host-fasthttp project
  help        show this usage


init USAGE:
  http-fasthttp init [MODULE_NAME] [OPTIONS...]

init ARGS:
  MODULE_NAME   the go module name for the application.
                NOTE: The [MODULE_NAME] can use "." period symbol
				      to apply current working directory name.

init OPTIONS:
  -v VERSION    the host-fasthttp version.

`)
}

func executeCommand(name string, args ...string) error {
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

func initModule(name string) (moduleName string, err error) {
	moduleName = name

	// NOTE: if specified module name as ".", parse current module name from
	//  existed go.mod file first or else use the current working directory
	//  name instead.
	if name == "." {
		f, _ := os.Stat("go.mod")
		if f != nil {
			return getModuleName()
		}

		cwd, err := os.Getwd()
		if err != nil {
			return moduleName, err
		}

		moduleName = filepath.Base(cwd)
		moduleName = strings.ToLower(moduleName)
	}
	return moduleName, executeCommand("go", "mod", "init", moduleName)
}

func initProject(metadata *AppMetadata) error {
	return do(
		generateFiles(metadata),
		generateDir(DIR_CONF),
		executeCommand("go", "get", "go.opentelemetry.io/otel@v1.16.0"),
		executeCommand("go", "mod", "tidy"),
	)
}

func generateDir(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		os.Mkdir(dir, os.ModePerm)
		return nil
	}
	return err
}

func generateFiles(metadata *AppMetadata) error {
	for filename, template := range __FILE_TEMPLATES {
		if err := generateFile(filename, template, metadata); err != nil {
			return err
		}
	}
	return nil
}

func generateFile(filename string, pattern string, metadata *AppMetadata) error {
	fmt.Printf("generating '%s' ...", filename)

	dir, _ := path.Split(filename)
	if len(dir) > 0 {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.Mkdir(dir, os.ModePerm)
		}
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		file, err := os.Create(filename)
		defer file.Close()

		if err != nil {
			return err
		}

		tmpl, err := template.New(filename).Parse(pattern)
		if err != nil {
			return err
		}

		err = tmpl.Execute(file, metadata)
		if err == nil {
			fmt.Println("ok")
		} else {
			fmt.Println("failed")
		}
		return err
	} else {
		fmt.Println("skipped")
	}
	return nil
}

func getModuleName() (string, error) {
	goModBytes, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	modName := modfile.ModulePath(goModBytes)

	return modName, nil
}

func getRuntimeVersion() string {
	str := runtime.Version()
	pattern := regexp.MustCompile(`^go(\d+\.\d+)`)

	matches := pattern.FindSubmatch([]byte(str))

	if len(matches) == 2 {
		return string(matches[1])
	}
	return ""
}

func extractAppExeName(moduleName string) string {
	index := strings.LastIndex(moduleName, "/")
	if index != -1 {
		return moduleName[index+1:]
	}
	return moduleName
}
