package main

const (
	REQUEST_FILE_TEMPLATE string = `package {{.HandlerModuleName}}

import (
	. "{{.AppModuleName}}/internal"

	"github.com/Bofry/trace"
	"github.com/valyala/fasthttp"
	"github.com/Bofry/host-fasthttp/response"
)

type {{.RequestName}} struct {
	ServiceProvider *ServiceProvider
}

func (r *{{.RequestName}}) Ping(ctx *fasthttp.RequestCtx) {
	// disable tracing
	trace.SpanFromContext(ctx).Disable(true)

	response.Text.Success(ctx, "PONG")
}
`
)

type FileMetadata struct {
	AppModuleName     string
	HandlerModuleName string
	RequestName       string
}
