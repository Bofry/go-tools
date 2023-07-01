package main

const (
	REQUEST_FILE_TEMPLATE string = `package {{.HandlerModuleName}}

import (
	"reflect"

	. "{{.AppModuleName}}/internal"

	"github.com/Bofry/trace"
	redis "github.com/Bofry/worker-redis"
)

var (
	_ redis.MessageHandler        = new({{.HandlerName}})
	_ redis.MessageObserverAffair = new({{.HandlerName}})
)

type {{.HandlerName}} struct {
	ServiceProvider *ServiceProvider
}

func (h *{{.HandlerName}}) Init() {
}

// ProcessMessage implements MessageHandler.
func (h *{{.HandlerName}}) ProcessMessage(ctx *redis.Context, message *redis.Message) {
	sp := trace.SpanFromContext(ctx)
	_ = sp

	message.Ack()
	message.Del()
}

// MessageObserverTypes implements MessageObserverAffair.
func (*{{.HandlerName}}) MessageObserverTypes() []reflect.Type {
	return []reflect.Type{
		// put your observer type here
	}
}
`
)

type FileMetadata struct {
	AppModuleName     string
	HandlerModuleName string
	HandlerName       string
}
