package main

const (
	MESSAGE_OBSERVER_FILE_TEMPLATE string = `package {{.ObserverModuleName}}

import (
	"log"
	"reflect"

	. "{{.AppModuleName}}/internal"

	redis "github.com/Bofry/worker-redis"
	"github.com/Bofry/worker-redis/tracing"
)

var _ redis.MessageObserver = new({{.ObserverName}})

type {{.ObserverName}} struct {
	ServiceProvider *ServiceProvider
}

func (obs *{{.ObserverName}}) Init() {
	obs.ServiceProvider.ConfigureLogger(log.Default())
}

// OnAck implements redis.MessageObserver.
func (obs *{{.ObserverName}}) OnAck(ctx *redis.Context, message *redis.Message) {
	tr := tracing.GetTracer(obs)
	sp := tr.Start(ctx, "OnAck()")
	defer sp.End()

}

// OnDel implements redis.MessageObserver.
func (obs *{{.ObserverName}}) OnDel(ctx *redis.Context, message *redis.Message) {
	tr := tracing.GetTracer(obs)
	sp := tr.Start(ctx, "OnDel()")
	defer sp.End()

}

// Type implements redis.MessageObserver.
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
