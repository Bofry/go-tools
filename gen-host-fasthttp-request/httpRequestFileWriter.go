package main

import (
	"io"
)

var _ FileWriter = new(HttpRequestFileWriter)

type HttpRequestFileWriter struct {
	AppModuleName       string
	RequestPackageName  string
	IsSubRequestPackage bool
	RequestName         string
	RequestPrefix       string

	RequestFile         io.Writer
	RequestGetArgvFile  io.Writer
	RequestPostArgvFile io.Writer
}

func (w *HttpRequestFileWriter) Write() error {
	var err error

	err = HttpRequestFileTemplate.ExecuteTemplate(w.RequestFile, TEMPLATE_NAME_REQUEST_FILE, w)
	if err != nil {
		return err
	}

	if w.RequestGetArgvFile != nil {
		err = HttpRequestFileTemplate.ExecuteTemplate(w.RequestGetArgvFile, TEMPLATE_NAME_REQUEST_GET_ARGV_FILE, w)
		if err != nil {
			return err
		}
	}

	if w.RequestPostArgvFile != nil {
		err = HttpRequestFileTemplate.ExecuteTemplate(w.RequestPostArgvFile, TEMPLATE_NAME_REQUEST_POST_ARGV_FILE, w)
		if err != nil {
			return err
		}
	}

	return nil
}
