package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
)

var (
	_EXPECT_FILE_ENV          = FILE_ENV_TEMPLATE
	_EXPECT_FILE_ENV_SAMPLE   = FILE_ENV_SAMPLE_TEMPLATE
	_EXPECT_FILE_GITIGNORE    = FILE_GITIGNORE_TEMPLATE
	_EXPECT_FILE_LOAD_ENV_SH  = FILE_LOAD_ENV_SH_TEMPLATE
	_EXPECT_FILE_LOAD_ENV_BAT = FILE_LOAD_ENV_BAT_TEMPLATE
	_EXPECT_FILE_DOCKERFILE   = `
FROM golang:1.19-alpine

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

# Fix CVEs vulnerabilities
RUN sed -i 's/v3.15/v3.16/g' /etc/apk/repositories \
&& apk update \
&& apk upgrade

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && \
	chmod -R 777 "$GOPATH" && \
	apk add --no-cache gcc musl-dev && \
	apk add git && \
	apk update && \
	apk upgrade 

# RUN go version
ADD . $GOPATH/src/app

WORKDIR $GOPATH/src/app

# docker build
RUN go mod tidy
RUN go build -tags musl -o host-fasthttp-demo $GOPATH/src/app/app.go

RUN apk del git && \
	rm -rf $GOPATH/src/github.com/

CMD ./host-fasthttp-demo
`
	_EXPECT_FILE_CONFIG_LOCAL_YAML = FILE_CONFIG_LOCAL_YAML_TEMPLATE
	_EXPECT_FILE_CONFIG_YAML       = FILE_CONFIG_YAML_TEMPLATE
	_EXPECT_FILE_INTERNAL_DEF_GO   = strings.ReplaceAll(`package internal

import (
	"log"

	fasthttp "github.com/Bofry/host-fasthttp"
)

var (
	logger *log.Logger = log.New(log.Writer(), "[host-fasthttp-demo] ", log.LstdFlags|log.Lmsgprefix|log.LUTC)
)

type (
	Host fasthttp.Host

	Config struct {
		Environment string ”env:"Environment"”

		// app
		Version   string ”resource:".VERSION"”
		Signature string ”resource:".SIGNATURE"”

		// host-fasthttp server
		ListenAddress  string ”yaml:"address"        arg:"address;the combination of IP address and listen port"”
		EnableCompress bool   ”yaml:"useCompress"    arg:"use-compress;indicates the response enable compress or not"”
		ServerName     string ”yaml:"serverName"”

		// tracing
		JaegerTraceUrl string ”env:"JaegerTraceUrl"”

		// put your configuration here
	}
)

func (h *Host) Init(conf *Config) {
	h.Server = &fasthttp.Server{
		Name:                          conf.ServerName,
		DisableKeepalive:              true,
		DisableHeaderNamesNormalizing: true,
		Logger:                        logger,
	}
	h.ListenAddress = conf.ListenAddress
	h.EnableCompress = conf.EnableCompress
	h.Version = conf.Version
}
`, "”", "`")
	_EXPECT_FILE_INTERNAL_SERVICE_PROVIDER_GO = FILE_INTERNAL_SERVICE_PROVIDER_GO_TEMPLATE
	_EXPECT_FILE_INTERNAL_APP_GO              = FILE_INTERNAL_APP_GO_TEMPLATE
	_EXPECT_FILE_APP_GO                       = strings.ReplaceAll(`package main

import (	
	. "host-fasthttp-demo/internal"

	"github.com/Bofry/config"
	fasthttp "github.com/Bofry/host-fasthttp"
	"github.com/Bofry/host-fasthttp/response"
	"github.com/Bofry/host-fasthttp/response/failure"
)

//go:generate gen-host-fasthttp-request
type RequestManager struct {
	/* put your request handler here */
	// *HealthCheckRequest ”url:"/healthcheck"”
}

func main() {
	app := App{}
	fasthttp.Startup(&app).
		Middlewares(
			fasthttp.UseRequestManager(&RequestManager{}),
			fasthttp.UseXHttpMethodHeader(),
			fasthttp.UseTracing(true),
			fasthttp.UseErrorHandler(func(ctx *fasthttp.RequestCtx, err interface{}) {
				fail, ok := err.(*failure.Failure)
				if ok {
					response.Json.Failure(ctx, fail, fasthttp.StatusBadRequest)
				}
			}),
			fasthttp.UseUnhandledRequestHandler(func(ctx *fasthttp.RequestCtx) {
				ctx.SetStatusCode(fasthttp.StatusNotFound)
			}),
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
`, "”", "`")
)

func Test(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	os.Chdir(tmp)
	os.Args = []string{
		"host-fasthttp",
		"init",
		"host-fasthttp-demo",
		"-v",
		"bb3be52",
	}
	// NOTE: avoid painc when call os.Exit() under testing
	osExit = func(i int) {} // do nothing
	main()

	{
		// check .conf
		path := path.Join(tmp, DIR_CONF)
		fs, err := os.Stat(path)
		if err != nil {
			t.Errorf("should exist folder '%s', but got %v", path, err)
		}
		if fs == nil {
			t.Errorf("should exist folder '%s'", path)
		}
	}
	{
		// check .env
		content, err := readFile(tmp, FILE_ENV)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_ENV
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check .env.sample
		content, err := readFile(tmp, FILE_ENV_SAMPLE)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_ENV_SAMPLE
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check .gitignore
		content, err := readFile(tmp, FILE_GITIGNORE)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_GITIGNORE
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check loadenv.sh
		content, err := readFile(tmp, FILE_LOAD_ENV_SH)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_LOAD_ENV_SH
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check loadenv.bat
		content, err := readFile(tmp, FILE_LOAD_ENV_BAT)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_LOAD_ENV_BAT
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check Dockerfile
		content, err := readFile(tmp, FILE_DOCKERFILE)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_DOCKERFILE
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check config.local.yaml
		content, err := readFile(tmp, FILE_CONFIG_LOCAL_YAML)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_CONFIG_LOCAL_YAML
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check config.yaml
		content, err := readFile(tmp, FILE_CONFIG_YAML)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_CONFIG_YAML
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check internal/def.go
		content, err := readFile(tmp, FILE_INTERNAL_DEF_GO)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_INTERNAL_DEF_GO
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check serviceProvider/def.go
		content, err := readFile(tmp, FILE_INTERNAL_SERVICE_PROVIDER_GO)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_INTERNAL_SERVICE_PROVIDER_GO
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check internal/app.go
		content, err := readFile(tmp, FILE_INTERNAL_APP_GO)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_INTERNAL_APP_GO
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		// check app.go
		content, err := readFile(tmp, FILE_APP_GO)
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_APP_GO
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}

	// check go build
	err = executeCommand("go", "build")
	if err != nil {
		t.Fatal(err)
	}
	os.Chdir(workdir)
}

func readFile(tmpPath string, filename string) ([]byte, error) {
	filepath := path.Join(tmpPath, filename)
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file '%s' cause %v", filename, err)
	}
	return content, nil
}
