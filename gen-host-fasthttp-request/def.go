package main

import (
	"html/template"
	"strings"
)

var (
	TEMPLATE_NAME_REQUEST_FILE       string = "RequestFile"
	TEMPLATE_NAME_WEBSOCKET_APP_FILE string = "WebsocketAppFile"

	HTTP_REQUEST_FILE_TEMPLATE string = `package {{.HandlerModuleName}}

import (
	"log"
	. "{{.AppModuleName}}/internal"

	"github.com/Bofry/host-fasthttp/response"
	"github.com/Bofry/host-fasthttp/tracing"
	"github.com/valyala/fasthttp"
)

type {{.RequestName}} struct {
	ServiceProvider *ServiceProvider
}

func (r *{{.RequestName}}) Init() {
	r.ServiceProvider.ConfigureLogger(log.Default())
}

func (r *{{.RequestName}}) Get(ctx *fasthttp.RequestCtx) {
	sp := tracing.SpanFromRequestCtx(ctx)
	_ = sp

	response.Text.Success(ctx, "OK")
}
`

	WEBSOCKET_REQUEST_FILE_TEMPLATE string = `package {{.HandlerModuleName}}

import (
	"context"
	"log"
	"{{.AppModuleName}}/handler/websocket/{{.WebsocketAppModuleName}}"
	. "{{.AppModuleName}}/internal"

	"github.com/Bofry/host-fasthttp/app/websocket"
	"github.com/Bofry/host-fasthttp/response"
	"github.com/Bofry/host-fasthttp/tracing"
	"github.com/Bofry/host/app"
	"github.com/valyala/fasthttp"
)

type {{.RequestName}} struct {
	ServiceProvider *ServiceProvider
	Config          *Config

	WesocketApp *app.Application
}

func (r *{{.RequestName}}) Init() {
	r.ServiceProvider.ConfigureLogger(log.Default())

	// setup WebsocketApp
	{
		r.WesocketApp = app.Init(&{{.WebsocketAppModuleName}}.Module,
			app.BindServiceProvider(r.ServiceProvider),
			app.BindConfig(r.Config),
			app.BindEventClient(app.MultiEventClient{
				// register channel and EventClient map herre
			}),
		)
		r.WesocketApp.Start(context.Background())
	}
}

func (r *{{.RequestName}}) Get(ctx *fasthttp.RequestCtx) {
	sp := tracing.SpanFromRequestCtx(ctx)
	_ = sp

	client := websocket.NewMessageClient(ctx)
	client.RegisterCloseHandler(func(mc app.MessageClient) {
		// register close processing here
	})
	r.WesocketApp.MessageClientManager().Join(client)

	response.Text.Success(ctx, "OK")
}
`

	WEBSOCKET_APP_FILE_TEMPLATE string = strings.ReplaceAll(`package {{.WebsocketAppModuleName}}

import (
	"{{.AppModuleName}}/internal"

	"github.com/Bofry/host/app"
)

//go:generate gen-host-app-handler
var Module = struct {
	/* put your MessageHandler below */
	Echo app.MessageHandler ”protocol:"????"”

	/* put your EventHandler below */
	EchoEvent app.EventHandler ”channel:"????"”

	*App
	app.ModuleOptionCollection
}{
	ModuleOptionCollection: app.ModuleOptions(
		app.WithProtocolResolver(func(format app.MessageFormat, payload []byte) string {
			/* write your protocol resolving below */
			return ""
		}),
	),
}

type App struct {
	ServiceProvider *internal.ServiceProvider
	Config          *internal.Config
}

func (ap *App) Init() {
}

func (ap *App) DefaultMessageHandler(ctx *app.Context, message *app.Message) {

}

func (ap *App) DefaultEventHandler(ctx *app.Context, event *app.Event) error {
	return nil
}
`, "”", "`")
)

var (
	HttpRequestFileTemplate      *template.Template
	WebsocketRequestFileTemplate *template.Template
)

type (
	FileWriter interface {
		Write() error
	}
)

func init() {
	{
		tmpl, err := template.New(TEMPLATE_NAME_REQUEST_FILE).Parse(HTTP_REQUEST_FILE_TEMPLATE)
		if err != nil {
			panic(err)
		}
		HttpRequestFileTemplate = tmpl
	}

	{
		tmpl, err := template.New(TEMPLATE_NAME_REQUEST_FILE).Parse(WEBSOCKET_REQUEST_FILE_TEMPLATE)
		if err != nil {
			panic(err)
		}
		tmpl, err = tmpl.New(TEMPLATE_NAME_WEBSOCKET_APP_FILE).Parse(WEBSOCKET_APP_FILE_TEMPLATE)
		WebsocketRequestFileTemplate = tmpl
	}
}
