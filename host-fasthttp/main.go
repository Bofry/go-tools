package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
		FILE_CONFIG_YAML:                  FILE_CONFIG_YAML_TEMPLATE,
		FILE_CONFIG_LOCAL_YAML:            FILE_CONFIG_LOCAL_YAML_TEMPLATE,
		FILE_GITIGNORE:                    FILE_GITIGNORE_TEMPLATE,
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
			moduleName = os.Args[2]
			moduleName, err = initModule(moduleName)
			if err != nil {
				throw(err.Error())
				exit(1)
			}
			if len(os.Args) > 3 {
				v := os.Args[3]
				switch v {
				case "-v":
					if len(os.Args) > 4 {
						// run go get -u -v github.com/Bofry/host-fasthttp@<version>
						v = os.Args[4]
						err = executeCommand("go", "get", "-u", "-v", "github.com/Bofry/host-fasthttp@"+v)
						if err != nil {
							throw(err.Error())
							exit(1)
						}
					}
				}
			}
		} else {
			moduleName, err = getModuleName()
			if err != nil {
				throw(err.Error())
				exit(1)
			}
		}

		metadata := AppMetadata{
			ModuleName: moduleName,
			AppExeName: extractAppExeName(moduleName),
		}
		err = initProject(&metadata)
		if err != nil {
			throw(err.Error())
			exit(1)
		}
	case "help":
		showUsage()
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

	// NOTE: if module name is ".", use the current working
	//  directory instead.
	if name == "." {
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
	goModBytes, err := ioutil.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	modName := modfile.ModulePath(goModBytes)

	return modName, nil
}

func extractAppExeName(moduleName string) string {
	index := strings.LastIndex(moduleName, "/")
	if index != -1 {
		return moduleName[index+1:]
	}
	return moduleName
}
