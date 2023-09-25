package main

const (
	REQUEST_FILE_TEMPLATE string = `package {{.HandlerModuleName}}

import (
	"log"
	"reflect"

	. "{{.AppModuleName}}/internal"

	"github.com/Bofry/trace"
	nsq "github.com/Bofry/worker-nsq"
)

var (
	_ nsq.MessageHandler        = new({{.HandlerName}})
	_ nsq.MessageObserverAffair = new({{.HandlerName}})
)

type {{.HandlerName}} struct {
	ServiceProvider *ServiceProvider
}

func (h *{{.HandlerName}}) Init() {
	h.ServiceProvider.ConfigureLogger(log.Default())
}

// ProcessMessage implements MessageHandler.
func (h *{{.HandlerName}}) ProcessMessage(ctx *nsq.Context, message *nsq.Message) error {
	sp := trace.SpanFromContext(ctx)
	_ = sp

	return nil
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
