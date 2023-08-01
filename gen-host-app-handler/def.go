package main

import (
	"html/template"
	"io"
)

const (
	MESSAGE_HANDLER_TEMPLATE = `package {{.PackageName}}

import (
	"github.com/Bofry/arg"
	"github.com/Bofry/host/app"
)

var _ app.MessageContent = new({{.MessageName}}ReqMessage)

func (app *App) {{.MessageName}}(ctx *app.Context, message *app.Message) {
	content := {{.MessageName}}ReqMessage{}
	err := message.DecodeContent(&content)
	if err != nil {
		panic(err)
	}
}

type {{.MessageName}}ReqMessage struct {
}

// Decode implements app.MessageContent.
func (m *{{.MessageName}}ReqMessage) Decode(format app.MessageFormat, body []byte) error {
	panic("unimplemented")
}

// Encode implements app.MessageContent.
func (m *{{.MessageName}}ReqMessage) Encode() (app.MessageFormat, []byte) {
	panic("unimplemented")
}

// Validate implements app.MessageContent.
func (m *{{.MessageName}}ReqMessage) Validate() error {
	err := arg.Assert(
		// add your validation code here
	)
	return err
}
`

	EVENT_HANDLER_TEMPLATE = `package {{.PackageName}}

import "github.com/Bofry/host/app"

func (app *App) {{.EventName}}(ctx *app.Context, event *app.Event) error {
	return nil
}
`
)

var (
	MessageHandlerFileTemplate *template.Template
	EventHandlerFileTemplate   *template.Template
)

type (
	FileWriter interface {
		Write(io.Writer) error
	}
)

func init() {
	{
		tmpl, err := template.New("").Parse(MESSAGE_HANDLER_TEMPLATE)
		if err != nil {
			panic(err)
		}
		MessageHandlerFileTemplate = tmpl
	}

	{
		tmpl, err := template.New("").Parse(EVENT_HANDLER_TEMPLATE)
		if err != nil {
			panic(err)
		}
		EventHandlerFileTemplate = tmpl
	}

}
