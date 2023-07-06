package main

import (
	"path"
	"strings"
)

var (
	DIR_CONF = ".conf"

	FILE_ENV          = ".env"
	FILE_ENV_TEMPLATE = `Environment=local
JaegerTraceUrl=
`

	FILE_ENV_SAMPLE          = ".env.sample"
	FILE_ENV_SAMPLE_TEMPLATE = `Environment=local
JaegerTraceUrl=http://localhost:14268/api/traces
`

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

	FILE_SERVICE_NAME          = ".SERVICE_NAME"
	FILE_SERVICE_NAME_TEMPLATE = `{{.AppModuleName}}
`

	FILE_LOAD_ENV_SH          = "loadenv.sh"
	FILE_LOAD_ENV_SH_TEMPLATE = `#!/usr/bin/env bash

ENV_FILE=.env

if [ ! -f "$ENV_FILE" ]; then
	echo "can not find $ENV_FILE file"
	exit 1
fi
# load .env file
while IFS='' read -r setting || [[ -n "$setting" ]]; do
	if [ "${setting:0:1}" != "#" ]; then
		export ${setting}
	fi
done < $ENV_FILE
`

	FILE_LOAD_ENV_BAT          = "loadenv.bat"
	FILE_LOAD_ENV_BAT_TEMPLATE = `@ECHO OFF

SET ENV_FILE=.env

IF NOT EXIST %ENV_FILE% (
	ECHO "can not find %ENV_FILE% file"
	EXIT /B 1
)
REM load .env file
FOR /F "tokens=*" %%i in ('type %ENV_FILE%') DO (
	SET LINE=%%i
	IF [!LINE!] NEQ [] (
		IF "!LINE:~0,1!" NEQ "#" (
			SET %%i
		)
	)
)
`

	FILE_DOCKERFILE          = "Dockerfile"
	FILE_DOCKERFILE_TEMPLATE = `
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
RUN go build -tags musl -o {{.AppExeName}} $GOPATH/src/app/app.go

RUN apk del git && \
	rm -rf $GOPATH/src/github.com/

CMD ./{{.AppExeName}}
`

	FILE_CONFIG_LOCAL_YAML          = "config.local.yaml"
	FILE_CONFIG_LOCAL_YAML_TEMPLATE = `ListenAddress: ":10074"
`

	FILE_CONFIG_YAML          = "config.yaml"
	FILE_CONFIG_YAML_TEMPLATE = `
ListenAddress: ":80"
ServerName: {{.AppExeName}}
UseCompress: true
`

	FILE_INTERNAL_DEF_GO          = path.Join("internal", "def.go")
	FILE_INTERNAL_DEF_GO_TEMPLATE = strings.ReplaceAll(`package internal

import (
	"log"

	fasthttp "github.com/Bofry/host-fasthttp"
)

var (
	defaultLogger *log.Logger = log.New(log.Writer(), {{printf "%q" (print "[" .AppModuleName "] ")}}, log.LstdFlags|log.Lmsgprefix|log.LUTC)
)

type (
	Host fasthttp.Host

	Config struct {
		Environment string ”env:"Environment"”

		// app
		Version     string ”resource:".VERSION"”
		Signature   string ”resource:".SIGNATURE"”
		ServiceName string ”resource:".SERVICE_NAME"”

		// host-fasthttp server
		ListenAddress  string ”yaml:"ListenAddress"  arg:"listen-address;the combination of IP address and listen port"”
		EnableCompress bool   ”yaml:"UseCompress"    arg:"use-compress;indicates the response enable compress or not"”
		ServerName     string ”yaml:"ServerName"”

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
		Logger:                        defaultLogger,
	}
	h.ListenAddress = conf.ListenAddress
	h.EnableCompress = conf.EnableCompress
	h.Version = conf.Version
}

func (h *Host) OnError(err error) (disposed bool) {
	return false
}
`, "”", "`")

	FILE_INTERNAL_SERVICE_PROVIDER_GO          = path.Join("internal", "serviceProvider.go")
	FILE_INTERNAL_SERVICE_PROVIDER_GO_TEMPLATE = `package internal

import (
	"log"

	"github.com/Bofry/trace"
	"go.opentelemetry.io/otel/propagation"
)

type ServiceProvider struct {}

func (p *ServiceProvider) Init(conf *Config) {
	// initialize service provider components
}

func (p *ServiceProvider) TracerProvider() *trace.SeverityTracerProvider {
	return trace.GetTracerProvider()
}

func (p *ServiceProvider) TextMapPropagator() propagation.TextMapPropagator {
	return trace.GetTextMapPropagator()
}

func (p *ServiceProvider) Logger() *log.Logger {
	return defaultLogger
}

func (p *ServiceProvider) ConfigureLogger(l *log.Logger) {
	l.SetOutput(p.Logger().Writer())
	l.SetPrefix(p.Logger().Prefix())
	l.SetFlags(p.Logger().Flags())
}
`

	FILE_INTERNAL_APP_GO          = path.Join("internal", "app.go")
	FILE_INTERNAL_APP_GO_TEMPLATE = `package internal

import (
	"context"
	"log"

	"github.com/Bofry/host"
	"github.com/Bofry/trace"
	"go.opentelemetry.io/otel/propagation"
)

var (
	_ host.App                    = new(App)
	_ host.AppStaterConfigurator  = new(App)
	_ host.AppTracingConfigurator = new(App)
)

type App struct {
	Host            *Host
	Config          *Config
	ServiceProvider *ServiceProvider
}

func (app *App) Init() {
	// initialize daemon components
}

func (app *App) OnInit() {
}

func (app *App) OnInitComplete() {
}

func (app *App) OnStart(ctx context.Context) {
}

func (app *App) OnStop(ctx context.Context) {
	{
		defaultLogger.Printf("stoping TracerProvider")
		tp := trace.GetTracerProvider()
		err := tp.Shutdown(ctx)
		if err != nil {
			defaultLogger.Printf("stoping TracerProvider error: %+v", err)
		}
	}
}

func (app *App) ConfigureLogger(l *log.Logger) {
	l.SetFlags(defaultLogger.Flags())
	l.SetOutput(defaultLogger.Writer())
}

func (app *App) Logger() *log.Logger {
	return defaultLogger
}

func (app *App) ConfigureTracerProvider() {
	if len(app.Config.JaegerTraceUrl) == 0 {
		tp, _ := trace.NoopProvider()
		trace.SetTracerProvider(tp)
		return
	}

	tp, err := trace.JaegerProvider(app.Config.JaegerTraceUrl,
		trace.ServiceName(app.Config.ServiceName),
		trace.Signature(app.Config.Signature),
		trace.Version(app.Config.Version),
		trace.Environment(app.Config.Environment),
		trace.OS(),
		trace.Pid(),
	)
	if err != nil {
		defaultLogger.Fatal(err)
	}

	trace.SetTracerProvider(tp)
}

func (app *App) TracerProvider() *trace.SeverityTracerProvider {
	return trace.GetTracerProvider()
}

func (app *App) ConfigureTextMapPropagator() {
	trace.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}

func (app *App) TextMapPropagator() propagation.TextMapPropagator {
	return trace.GetTextMapPropagator()
}
`

	FILE_INTERNAL_EVENT_LOG_GO          = path.Join("internal", "eventLog.go")
	FILE_INTERNAL_EVENT_LOG_GO_TEMPLATE = `package internal

import (
	"log"

	fasthttp "github.com/Bofry/host-fasthttp"
	"github.com/Bofry/host-fasthttp/response"
)

var _ fasthttp.EventLog = EventLog{}

type EventLog struct {
	logger   *log.Logger
	evidence fasthttp.EventEvidence
}

// Flush implements middleware.EventLog.
func (l EventLog) Flush() {
}

// OnError implements middleware.EventLog.
func (l EventLog) OnError(ctx *fasthttp.RequestCtx, err interface{}, stackTrace []byte) {
}

// OnProcessRequest implements middleware.EventLog.
func (l EventLog) OnProcessRequest(ctx *fasthttp.RequestCtx) {
}

// OnProcessRequestComplete implements middleware.EventLog.
func (l EventLog) OnProcessRequestComplete(ctx *fasthttp.RequestCtx, flag response.ResponseFlag) {
}
`

	FILE_INTERNAL_LOGGING_SERVICE_GO          = path.Join("internal", "loggingService.go")
	FILE_INTERNAL_LOGGING_SERVICE_GO_TEMPLATE = `package internal

import (
	"log"

	fasthttp "github.com/Bofry/host-fasthttp"
)

var _ fasthttp.LoggingService = new(LoggingService)

type LoggingService struct {
	logger *log.Logger
}

// ConfigureLogger implements middleware.LoggingService.
func (s *LoggingService) ConfigureLogger(l *log.Logger) {
	s.logger = l
}

// CreateEventLog implements middleware.LoggingService.
func (s *LoggingService) CreateEventLog(ev fasthttp.EventEvidence) fasthttp.EventLog {
	return EventLog{
		logger:   s.logger,
		evidence: ev,
	}
}
`

	FILE_APP_GO          = "app.go"
	FILE_APP_GO_TEMPLATE = strings.ReplaceAll(`package main

import (	
	. "{{.AppModuleName}}/internal"

	"github.com/Bofry/config"
	fasthttp "github.com/Bofry/host-fasthttp"
	"github.com/Bofry/host-fasthttp/handlers"
	"github.com/Bofry/host-fasthttp/response"
	"github.com/Bofry/host-fasthttp/response/failure"
)

//go:generate gen-host-fasthttp-request
type RequestManager struct {
	/* put your request handler below */
	// *RootRequest ”url:"/"”
	*handlers.HealthCheckRequest ”url:"/healthcheck"”
}

func main() {
	app := App{}
	fasthttp.Startup(&app).
		Middlewares(
			fasthttp.UseRequestManager(&RequestManager{}),
			fasthttp.UseXHttpMethodHeader(),
			fasthttp.UseLogging(&LoggingService{}),
			fasthttp.UseTracing(true),
			fasthttp.UseErrorHandler(func(ctx *fasthttp.RequestCtx, err interface{}) {
				fail, ok := err.(*failure.Failure)
				if ok {
					response.Json.Failure(ctx, fail, fasthttp.StatusBadRequest)
				}
			}),
			fasthttp.UseUnhandledRequestHandler(func(ctx *fasthttp.RequestCtx) {
				ctx.NotFound()
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

type AppMetadata struct {
	AppExeName    string
	AppModuleName string
}
