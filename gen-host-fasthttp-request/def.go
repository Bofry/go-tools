package main

const (
	REQUEST_FILE_TEMPLATE string = `package {{.HandlerModuleName}}

import (
	. "{{.AppModuleName}}/internal"
	"log"

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
	sp.Argv(nil)

	response.Text.Success(ctx, "OK")
}
`
)

type FileMetadata struct {
	AppModuleName     string
	HandlerModuleName string
	RequestName       string
}
