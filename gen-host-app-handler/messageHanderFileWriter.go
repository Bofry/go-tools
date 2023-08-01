package main

import "io"

var _ FileWriter = new(MessageHanderFileWriter)

type MessageHanderFileWriter struct {
	PackageName string
	MessageName string
}

func (w *MessageHanderFileWriter) Write(writer io.Writer) error {
	return MessageHandlerFileTemplate.Execute(writer, w)
}
