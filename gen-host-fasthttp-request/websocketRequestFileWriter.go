package main

import "io"

var _ FileWriter = new(WebsocketRequestFileWriter)

type WebsocketRequestFileWriter struct {
	AppModuleName          string
	RequestPackageName     string
	RequestName            string
	RequestPrefix          string
	IsSubRequestPackage    bool
	WebsocketAppModuleName string

	RequestFile        io.Writer
	WebsocketAppFile   io.Writer
	RequestGetArgvFile io.Writer
}

func (w *WebsocketRequestFileWriter) Write() error {
	var err error

	err = WebsocketRequestFileTemplate.ExecuteTemplate(w.RequestFile, TEMPLATE_NAME_REQUEST_FILE, w)
	if err != nil {
		return err
	}

	if w.RequestGetArgvFile != nil {
		err = WebsocketRequestFileTemplate.ExecuteTemplate(w.RequestGetArgvFile, TEMPLATE_NAME_REQUEST_GET_ARGV_FILE, w)
		if err != nil {
			return err
		}
	}

	err = WebsocketRequestFileTemplate.ExecuteTemplate(w.WebsocketAppFile, TEMPLATE_NAME_WEBSOCKET_APP_FILE, w)
	return err
}
