package main

import (
	"io"
)

var _ FileWriter = new(HttpRequestFileWriter)

type HttpRequestFileWriter struct {
	AppModuleName     string
	HandlerModuleName string
	RequestName       string

	RequestFile io.Writer
}

func (w *HttpRequestFileWriter) Write() error {
	return HttpRequestFileTemplate.ExecuteTemplate(w.RequestFile, TEMPLATE_NAME_REQUEST_FILE, w)
}
