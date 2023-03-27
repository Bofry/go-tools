package main

const (
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
	FILE_CONFIG_LOCAL_YAML_TEMPLATE = `address: ":10074"
`

	FILE_CONFIG_YAML          = "config.yaml"
	FILE_CONFIG_YAML_TEMPLATE = `address: ":80"
serverName: WebAPI
useCompress: true
`

	FILE_INTERNAL_DEF_GO          = "internal/def.go"
	FILE_INTERNAL_DEF_GO_TEMPLATE = `package internal

import (
	"log"

	fasthttp "github.com/Bofry/host-fasthttp"
)

var (
	logger *log.Logger = log.New(log.Writer(), {{printf "%q" (print "[" .ModuleName "] ")}}, log.LstdFlags|log.Lmsgprefix|log.LUTC)
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
`

	FILE_INTERNAL_SERVICE_PROVIDER_GO          = "internal/serviceProvider.go"
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
	return logger
}
`

	FILE_INTERNAL_APP_GO          = "internal/app.go"
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
}

func (app *App) ConfigureLogger(l *log.Logger) {
	l.SetFlags(logger.Flags())
	l.SetOutput(logger.Writer())
}

func (app *App) Logger() *log.Logger {
	return logger
}

func (app *App) ConfigureTracerProvider() {
	if len(app.Config.JaegerTraceUrl) == 0 {
		return
	}

	tp, err := trace.JaegerProvider(app.Config.JaegerTraceUrl,
		trace.ServiceName(app.Config.ServerName),
		trace.Signature(app.Config.Signature),
		trace.Version(app.Config.Version),
		trace.Environment(app.Config.Environment),
		trace.OS(),
		trace.Pid(),
	)
	if err != nil {
		logger.Fatal(err)
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

	FILE_APP_GO          = "app.go"
	FILE_APP_GO_TEMPLATE = `package main

import (	
	. "{{.ModuleName}}/internal"

	"github.com/Bofry/config"
	fasthttp "github.com/Bofry/host-fasthttp"
	"github.com/Bofry/host-fasthttp/response"
	"github.com/Bofry/host-fasthttp/response/failure"
)

//go:generate gen-host-fasthttp-handler
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
`
)

type AppMetadata struct {
	AppExeName string
	ModuleName string
}
