package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"golang.org/x/mod/modfile"
)

const (
	DIR_CONF = ".conf"

	FILE_ENV          = ".env"
	FILE_ENV_TEMPLATE = `Environment=local`

	FILE_ENV_SAMPLE          = ".env.sample"
	FILE_ENV_SAMPLE_TEMPLATE = `Environment=local`

	FILE_GITIGNORE          = ".gitignore"
	FILE_GITIGNORE_TEMPLATE = `.vscode
.env

.VERSION

# local environment shell script
env.bat
env.sh
env.*.bat
env.*.sh
`

	FILE_CONFIG_LOCAL_YAML          = "config.local.yaml"
	FILE_CONFIG_LOCAL_YAML_TEMPLATE = `address: ":10074"
`

	FILE_CONFIG_YAML          = "config.yaml"
	FILE_CONFIG_YAML_TEMPLATE = `address: ":80"
serverName: WebAPI
useCompress: true
`

	FILE_APP_CONTEXT_GO          = "internal/appContext.go"
	FILE_APP_CONTEXT_GO_TEMPLATE = `package internal

import fasthttp "github.com/Bofry/host-fasthttp"

type (
	AppContext struct {
		Host            *Host
		Config          *Config
		ServiceProvider *ServiceProvider
	}

	Host fasthttp.Host

	Config struct {
		// host-fasthttp server configuration
		ListenAddress  string ”yaml:"address"        arg:"address;the combination of IP address and listen port"”
		EnableCompress bool   ”yaml:"useCompress"    arg:"use-compress;indicates the response enable compress or not"”
		ServerName     string ”yaml:"serverName"”
		Version        string ”resource:".VERSION"”

		// put your configuration below
	}

	ServiceProvider struct {}
)

func (h *Host) Init(conf *Config) {
	h.Server = &fasthttp.Server{
		Name:                          conf.ServerName,
		DisableKeepalive:              true,
		DisableHeaderNamesNormalizing: true,
	}
	h.ListenAddress = conf.ListenAddress
	h.EnableCompress = conf.EnableCompress
	h.Version = conf.Version
}


func (p *ServiceProvider) Init(conf *Config) {
	// initialize service provider components
}
`

	FILE_APP_GO          = "app.go"
	FILE_APP_GO_TEMPLATE = `package main

import (
	. "%[1]s/internal"

	"github.com/Bofry/config"
	fasthttp "github.com/Bofry/host-fasthttp"
)

//go:generate gen-host-fasthttp-resource
type ResourceManager struct {}

func main() {
	ctx := AppContext{}
	fasthttp.Startup(&ctx).
		Middlewares(
			fasthttp.UseResourceManager(&ResourceManager{}),
			fasthttp.UseXHttpMethodHeader(),
		).
		ConfigureConfiguration(func(service *config.ConfigurationService) {
			service.
				LoadYamlFile("config.yaml").
				LoadYamlFile("config.${Environment}.yaml").
				LoadEnvironmentVariables("").
				LoadResource(".").
				LoadResource(".conf/${Environment}").
				LoadCommandArguments()
		}).
		Run()
}
`
)

var (
	modulePath string = ""
)

func main() {
	if len(os.Args) != 2 {
		showUsage()
		os.Exit(0)
	}

	arg := os.Args[1]
	switch arg {
	case "init":
		initProject()
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

func showUsage() {
	fmt.Print(`Usage: http-fasthttp [command]

init        create new host-fasthttp project
help        show this usage
`)
}

func initProject() {
	modulePath, err := getModulePath()
	if err != nil {
		throw(err.Error())
		os.Exit(1)
	}

	err = do(
		generateFile(FILE_APP_GO, FILE_APP_GO_TEMPLATE, modulePath),
		generateFile(FILE_APP_CONTEXT_GO, FILE_APP_CONTEXT_GO_TEMPLATE),
		generateFile(FILE_CONFIG_YAML, FILE_CONFIG_YAML_TEMPLATE),
		generateFile(FILE_CONFIG_LOCAL_YAML, FILE_CONFIG_LOCAL_YAML_TEMPLATE),
		generateFile(FILE_GITIGNORE, FILE_GITIGNORE_TEMPLATE),
		generateFile(FILE_ENV, FILE_ENV_TEMPLATE),
		generateFile(FILE_ENV_SAMPLE, FILE_ENV_SAMPLE_TEMPLATE),
		generateDir(DIR_CONF),
	)
	if err != nil {
		throw(err.Error())
		os.Exit(1)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Run(); err != nil {
		throw(err.Error())
		os.Exit(1)
	}
}

func generateDir(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		os.Mkdir(dir, os.ModePerm)
		return nil
	}
	return err
}

func generateFile(filename string, template string, a ...interface{}) error {
	fmt.Printf("generating '%s' ...", filename)

	dir, _ := path.Split(filename)
	if len(dir) > 0 {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.Mkdir(dir, os.ModePerm)
		}
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		file, err := os.Create(filename)
		if err != nil {
			return err
		}

		template = strings.ReplaceAll(template, "”", "`")
		_, err = fmt.Fprintf(file, template, a...)
		if err == nil {
			fmt.Println("ok")
		} else {
			fmt.Println("failed")
		}
		return err
	} else {
		fmt.Println("skip")
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
