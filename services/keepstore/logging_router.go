package main

// LoggingRESTRouter
// LoggingResponseWriter

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
)

type LoggingResponseWriter struct {
	Status int
	Length int
	http.ResponseWriter
	Response string
}

func (loggingWriter *LoggingResponseWriter) WriteHeader(code int) {
	loggingWriter.Status = code
	loggingWriter.ResponseWriter.WriteHeader(code)
}

func (loggingWriter *LoggingResponseWriter) Write(data []byte) (int, error) {
	loggingWriter.Length += len(data)
	if loggingWriter.Status >= 400 {
		loggingWriter.Response += string(data)
	}
	return loggingWriter.ResponseWriter.Write(data)
}

type LoggingRESTRouter struct {
	router *mux.Router
}

func MakeLoggingRESTRouter() *LoggingRESTRouter {
	router := MakeRESTRouter()
	return (&LoggingRESTRouter{router})
}

func (loggingRouter *LoggingRESTRouter) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	loggingWriter := LoggingResponseWriter{200, 0, resp, ""}
	loggingRouter.router.ServeHTTP(&loggingWriter, req)
	if loggingWriter.Status >= 400 {
		log.Printf("[%s] %s %s %d %d '%s'", req.RemoteAddr, req.Method, req.URL.Path[1:], loggingWriter.Status, loggingWriter.Length, strings.TrimSpace(loggingWriter.Response))
	} else {
		log.Printf("[%s] %s %s %d %d", req.RemoteAddr, req.Method, req.URL.Path[1:], loggingWriter.Status, loggingWriter.Length)
	}
}
