package main

import "io"

var _ FileWriter = new(WebsocketRequestFileWriter)

type WebsocketRequestFileWriter struct {
	AppModuleName          string
	HandlerModuleName      string
	RequestName            string
	WebsocketAppModuleName string

	RequestFile      io.Writer
	WebsocketAppFile io.Writer
}

func (w *WebsocketRequestFileWriter) Write() error {
	err := WebsocketRequestFileTemplate.ExecuteTemplate(w.RequestFile, TEMPLATE_NAME_REQUEST_FILE, w)
	if err != nil {
		return err
	}
	err = WebsocketRequestFileTemplate.ExecuteTemplate(w.WebsocketAppFile, TEMPLATE_NAME_WEBSOCKET_APP_FILE, w)
	return err
}
