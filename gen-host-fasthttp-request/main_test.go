package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
)

var (
	_FILE_GO_MOD = `module host-fasthttp-request-demo

go 1.19

require (
	github.com/Bofry/host-fasthttp v0.2.0-alpha.20230703153647.0.20230723153100-a95def19beda
	github.com/Bofry/trace v0.0.0-20230327070031-663464d25b86
	github.com/valyala/fasthttp v1.45.0
)
`
	_FILE_APP_GO = strings.ReplaceAll(`package main

//go:generate gen-host-fasthttp-request
type RequestManager struct {
	/* put your request handler here */
	*HealthCheckRequest ”url:"/healthcheck"”
	*ChatRequest        ”url:"/chat"         @hijack:"websocket"”
}

func main() {}
`, "”", "`")

	_FILE_INTERNAL_SERVICE_PROVIDER_GO = `package internal

type ServiceProvider struct{}
`

	_EXPECT_FILE_APP_GO = strings.ReplaceAll(`package main

import . "host-fasthttp-request-demo/handler"

//go:generate gen-host-fasthttp-request
type RequestManager struct {
	/* put your request handler here */
	*HealthCheckRequest ”url:"/healthcheck"”
	*ChatRequest        ”url:"/chat"         @hijack:"websocket"”
}

func main() {}
`, "”", "`")

	_EXPECT_FILE_HEALTHCHECK_REQUEST_GO = `package handler

import (
	"log"
	"host-fasthttp-request-demo/handler/args"
	. "host-fasthttp-request-demo/internal"

	"github.com/Bofry/host-fasthttp/response"
	"github.com/Bofry/host-fasthttp/tracing"
	"github.com/Bofry/httparg"
	"github.com/valyala/fasthttp"
)

type HealthCheckRequest struct {
	ServiceProvider *ServiceProvider
}

func (r *HealthCheckRequest) Init() {
	r.ServiceProvider.ConfigureLogger(log.Default())
}

func (r *HealthCheckRequest) Get(ctx *fasthttp.RequestCtx) {
	sp := tracing.SpanFromRequestCtx(ctx)
	_ = sp

	argv := args.HealthCheckGetArgv{}

	httparg.Args(&argv).
		ProcessQueryString(ctx.QueryArgs().String()).
		Validate()

	response.Text.Success(ctx, "OK")
}

func (r *HealthCheckRequest) Post(ctx *fasthttp.RequestCtx) {
	sp := tracing.SpanFromRequestCtx(ctx)
	_ = sp

	argv := args.HealthCheckPostArgv{}

	httparg.Args(&argv).
		ProcessQueryString(ctx.QueryArgs().String()).
		ProcessContent(ctx.PostBody(), string(ctx.Request.Header.ContentType())).
		Validate()

	response.Text.Success(ctx, "OK")
}
`

	_EXPECT_FILE_CHAT_REQUEST_GO = `package handler

import (
	"context"
	"log"
	"host-fasthttp-request-demo/handler/args"
	"host-fasthttp-request-demo/handler/websocket/chat"
	. "host-fasthttp-request-demo/internal"

	"github.com/Bofry/host-fasthttp/app/websocket"
	"github.com/Bofry/host-fasthttp/response"
	"github.com/Bofry/host-fasthttp/tracing"
	"github.com/Bofry/host/app"
	"github.com/Bofry/httparg"
	"github.com/valyala/fasthttp"
)

type ChatRequest struct {
	ServiceProvider *ServiceProvider
	Config          *Config

	WesocketApp *app.Application
}

func (r *ChatRequest) Init() {
	r.ServiceProvider.ConfigureLogger(log.Default())

	// setup WebsocketApp
	{
		r.WesocketApp = app.Init(&chat.Module,
			app.BindServiceProvider(r.ServiceProvider),
			app.BindConfig(r.Config),
			app.BindEventClient(app.MultiEventClient{
				// register channel and EventClient map herre
			}),
		)
		r.WesocketApp.Start(context.Background())
	}
}

func (r *ChatRequest) Get(ctx *fasthttp.RequestCtx) {
	sp := tracing.SpanFromRequestCtx(ctx)
	_ = sp

	argv := args.ChatGetArgv{}

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

	_EXPECT_FILE_CHAT_APP_GO = strings.ReplaceAll(`package chat

import (
	"bytes"
	"log"
	"host-fasthttp-request-demo/internal"

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

func Test(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(func() {
		os.RemoveAll(tmp)
		os.Clearenv()
	})

	assert(t,
		createTempFiles(tmp, _FILE_GO_MOD, "go.mod", ""),
		createTempFiles(tmp, _FILE_APP_GO, "app.go", ""),
		createTempFiles(tmp, _FILE_INTERNAL_SERVICE_PROVIDER_GO, "serviceProvider.go", "internal"),
	)

	os.Args = []string{
		"gen-host-fasthttp-request",
		"-file",
		path.Join(tmp, "app.go"),
	}
	// NOTE: avoid painc when call os.Exit() under testing
	osExit = func(i int) {
		if i != 0 {
			t.Fatalf("got exit code %d", i)
		}
	}
	main()

	{
		content, err := readFile(tmp, "app.go", "")
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_APP_GO
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		content, err := readFile(tmp, "healthCheckRequest.go", "handler")
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_HEALTHCHECK_REQUEST_GO
		if expectedContent != string(content) {
			t.Errorf("healthCheckRequest.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		content, err := readFile(tmp, "chatRequest.go", "handler")
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_CHAT_REQUEST_GO
		if expectedContent != string(content) {
			t.Errorf("chatRequest.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
	{
		content, err := readFile(tmp, "app.go", "handler/websocket/chat")
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_CHAT_APP_GO
		if expectedContent != string(content) {
			t.Errorf("chatRequest.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}
}

func assert(t *testing.T, err ...error) {
	for _, e := range err {
		if e != nil {
			t.Fatal(e)
		}
	}
}

func createTempFiles(tmpPath string, content string, filename string, subDir string) error {
	if len(subDir) > 0 {
		tmpPath = path.Join(tmpPath, subDir)

		_, err := os.Stat(tmpPath)
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot create dir '%s' cause %v", subDir, err)
		}
		err = os.Mkdir(tmpPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("cannot create dir '%s' cause %v", subDir, err)
		}
	}
	fs, err := os.Create(path.Join(tmpPath, filename))
	defer fs.Close()
	if err != nil {
		return fmt.Errorf("cannot create file '%s' cause %v", filename, err)
	}
	_, err = fmt.Fprint(fs, content)
	return err
}

func readFile(tmpPath string, filename string, subDir string) ([]byte, error) {
	if len(subDir) > 0 {
		tmpPath = path.Join(tmpPath, subDir)
	}

	filepath := path.Join(tmpPath, filename)
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open dir '%s' cause %v", filename, err)
	}
	return content, nil
}
