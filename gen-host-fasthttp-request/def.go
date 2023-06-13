package main

const (
	REQUEST_FILE_TEMPLATE string = `package {{.HandlerModuleName}}

import (
	. "{{.AppModuleName}}/internal"

	"github.com/Bofry/host-fasthttp/response"
	"github.com/Bofry/host-fasthttp/tracing"
	"github.com/valyala/fasthttp"
)

type {{.RequestName}} struct {
	ServiceProvider *ServiceProvider
}

func (r *{{.RequestName}}) Ping(ctx *fasthttp.RequestCtx) {
	// disable tracing
	tracing.SpanFromRequestCtx(ctx).Disable(true)

	response.Text.Success(ctx, "PONG")
}
`
)

type FileMetadata struct {
	AppModuleName     string
	HandlerModuleName string
	RequestName       string
}
