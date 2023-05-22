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
	github.com/Bofry/host-fasthttp v0.1.2-0.20230328172331-74788714c83b
	github.com/Bofry/trace v0.0.0-20230327070031-663464d25b86
	github.com/valyala/fasthttp v1.45.0
)
`
	_FILE_APP_GO = strings.ReplaceAll(`package main

//go:generate gen-host-fasthttp-request
type RequestManager struct {
	/* put your request handler here */
	*HealthCheckRequest ”url:"/healthcheck"”
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
}

func main() {}
`, "”", "`")

	_EXPECT_FILE_HEALTHCHECK_REQUEST_GO = `package handler

import (
	. "host-fasthttp-request-demo/internal"

	"github.com/Bofry/trace"
	"github.com/valyala/fasthttp"
	"github.com/Bofry/host-fasthttp/response"
)

type HealthCheckRequest struct {
	ServiceProvider *ServiceProvider
}

func (r *HealthCheckRequest) Ping(ctx *fasthttp.RequestCtx) {
	// disable tracing
	trace.SpanFromContext(ctx).Disable(true)

	response.Success(ctx, "text/plain", []byte("PONG"))
}
`
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
	osExit = func(i int) {} // do nothing
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
