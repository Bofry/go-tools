package main

import "io"

var _ FileWriter = new(EventHandlerFileWriter)

type EventHandlerFileWriter struct {
	PackageName string
	EventName   string
}

func (w *EventHandlerFileWriter) Write(writer io.Writer) error {
	return EventHandlerFileTemplate.Execute(writer, w)
}
