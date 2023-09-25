package main

const (
	MESSAGE_OBSERVER_FILE_TEMPLATE string = `package {{.ObserverModuleName}}

import (
	"log"
	"reflect"

	. "{{.AppModuleName}}/internal"

	nsq "github.com/Bofry/worker-nsq"
	"github.com/Bofry/worker-nsq/tracing"
)

var _ nsq.MessageObserver = new({{.ObserverName}})

type {{.ObserverName}} struct {
	ServiceProvider *ServiceProvider
}

func (obs *{{.ObserverName}}) Init() {
	obs.ServiceProvider.ConfigureLogger(log.Default())
}

// OnFinish implements nsq.MessageObserver.
func (obs *{{.ObserverName}}) OnFinish(ctx *nsq.Context, message *nsq.Message) {
	tr := tracing.GetTracer(obs)
	sp := tr.Start(ctx, "OnFinish()")
	defer sp.End()

}

// OnRequeue implements nsq.MessageObserver.
func (obs *{{.ObserverName}}) OnRequeue(ctx *nsq.Context, message *nsq.Message) {
	tr := tracing.GetTracer(obs)
	sp := tr.Start(ctx, "OnRequeue()")
	defer sp.End()

}

// OnTouch implements nsq.MessageObserver.
func (obs *{{.ObserverName}}) OnTouch(ctx *nsq.Context, message *nsq.Message) {
	tr := tracing.GetTracer(obs)
	sp := tr.Start(ctx, "OnTouch()")
	defer sp.End()

}

// Type implements nsq.MessageObserver.
func (obs *{{.ObserverName}}) Type() reflect.Type {
	return reflect.TypeOf(obs)
}
`
)

type FileMetadata struct {
	AppModuleName      string
	ObserverModuleName string
	ObserverName       string
}
