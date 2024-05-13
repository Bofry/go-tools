package main

import (
	"html/template"
	"strings"
)

var (
	TEMPLATE_NAME_REQUEST_FILE           string = "RequestFile"
	TEMPLATE_NAME_REQUEST_GET_ARGV_FILE  string = "RequestGetArgvFile"
	TEMPLATE_NAME_REQUEST_POST_ARGV_FILE string = "RequestPostArgvFile"
	TEMPLATE_NAME_WEBSOCKET_APP_FILE     string = "WebsocketAppFile"

	HTTP_REQUEST_GET_ARGV_FILE_TEMPLATE string = strings.ReplaceAll(`package args

import (
	"github.com/Bofry/arg"
	"github.com/Bofry/httparg"
)

var (
	_ httparg.Validatable = new({{.RequestPrefix}}GetArgv)
)

//go:generate gen-bofry-arg-assertor
type {{.RequestPrefix}}GetArgv struct /* tag=query */ {
	Nonce *string ”query:"nonce"”
}

// Validate implements httparg.Validatable.
func (argv *{{.RequestPrefix}}GetArgv) Validate() error {
	v := argv.Assertor()

	err := arg.Assert(
		v.Nonce(arg.StringPtr.NonEmpty),
	)
	return err
}
`, "”", "`")

	HTTP_REQUEST_POST_ARGV_FILE_TEMPLATE string = strings.ReplaceAll(`package args

import (
	"github.com/Bofry/arg"
	"github.com/Bofry/httparg"
)

var (
	_ httparg.Validatable = new({{.RequestPrefix}}PostArgv)
)

//go:generate gen-bofry-arg-assertor
type {{.RequestPrefix}}PostArgv struct /* tag=json */ {
	Nonce *string ”query:"nonce"   ^:"query"”
	Text  string  ”json:"*text"”
}

// Validate implements httparg.Validatable.
func (argv *{{.RequestPrefix}}PostArgv) Validate() error {
	v := argv.Assertor()

	err := arg.Assert(
		v.Nonce(arg.StringPtr.NonEmpty),
		v.Text(arg.Strings.NonEmpty),
	)
	return err
}
`, "”", "`")

	HTTP_REQUEST_FILE_TEMPLATE string = `package {{.HandlerModuleName}}

import (
	"log"
	"{{.AppModuleName}}/handler/args"
	. "{{.AppModuleName}}/internal"

	"github.com/Bofry/host-fasthttp/response"
	"github.com/Bofry/host-fasthttp/tracing"
	"github.com/Bofry/httparg"
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

	argv := args.{{.RequestPrefix}}GetArgv{}

	httparg.Args(&argv).
		ProcessQueryString(ctx.QueryArgs().String()).
		Validate()

	response.Text.Success(ctx, "OK")
}

func (r *{{.RequestName}}) Post(ctx *fasthttp.RequestCtx) {
	sp := tracing.SpanFromRequestCtx(ctx)
	_ = sp

	argv := args.{{.RequestPrefix}}PostArgv{}

	httparg.Args(&argv).
		ProcessQueryString(ctx.QueryArgs().String()).
		ProcessContent(ctx.PostBody(), string(ctx.Request.Header.ContentType())).
		Validate()

	response.Text.Success(ctx, "OK")
}
`

	WEBSOCKET_REQUEST_FILE_TEMPLATE string = `package {{.HandlerModuleName}}

import (
	"context"
	"log"
	"{{.AppModuleName}}/handler/args"
	"{{.AppModuleName}}/handler/websocket/{{.WebsocketAppModuleName}}"
	. "{{.AppModuleName}}/internal"

	"github.com/Bofry/host-fasthttp/app/websocket"
	"github.com/Bofry/host-fasthttp/response"
	"github.com/Bofry/host-fasthttp/tracing"
	"github.com/Bofry/host/app"
	"github.com/Bofry/httparg"
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

	argv := args.{{.RequestPrefix}}GetArgv{}

	httparg.Args(&argv).
		ProcessQueryString(ctx.QueryArgs().String()).
		Validate()

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
	"bytes"
	"log"
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
		app.WithProtocolResolver(func(format app.MessageFormat, payload []byte) (string, []byte) {
			/* processing your protocol resolving below */
			if len(payload) > 5 && payload[4] == '\n' {
				return string(payload[:4]), payload[5:]
			}
			return "", payload
		}),
		app.WithProtocolEmitter(func(format app.MessageFormat, protocol string, body []byte) []byte {
			/* processing your protocol emitting below */
			if len(protocol) == 0 {
				return body
			}
			return bytes.Join(
				[][]byte{
					[]byte(protocol + "\n"),
					body,
				}, nil)
		}),
	),
}

type App struct {
	ServiceProvider *internal.ServiceProvider
	Config          *internal.Config
}

func (ap *App) Init() {
	ap.ServiceProvider.ConfigureLogger(log.Default())
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
		tmpl, err = tmpl.New(TEMPLATE_NAME_REQUEST_GET_ARGV_FILE).Parse(HTTP_REQUEST_GET_ARGV_FILE_TEMPLATE)
		if err != nil {
			panic(err)
		}
		tmpl, err = tmpl.New(TEMPLATE_NAME_REQUEST_POST_ARGV_FILE).Parse(HTTP_REQUEST_POST_ARGV_FILE_TEMPLATE)
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
		tmpl, err = tmpl.New(TEMPLATE_NAME_REQUEST_GET_ARGV_FILE).Parse(HTTP_REQUEST_GET_ARGV_FILE_TEMPLATE)
		if err != nil {
			panic(err)
		}
		tmpl, err = tmpl.New(TEMPLATE_NAME_WEBSOCKET_APP_FILE).Parse(WEBSOCKET_APP_FILE_TEMPLATE)
		if err != nil {
			panic(err)
		}
		WebsocketRequestFileTemplate = tmpl
	}
}
