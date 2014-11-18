package main

// LoggingRESTRouter
// LoggingResponseWriter

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type LoggingResponseWriter struct {
	Status int
	Length int
	http.ResponseWriter
}

func (loggingWriter *LoggingResponseWriter) WriteHeader(code int) {
	loggingWriter.Status = code
	loggingWriter.ResponseWriter.WriteHeader(code)
}

func (loggingWriter *LoggingResponseWriter) Write(data []byte) (int, error) {
	loggingWriter.Length += len(data)
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
	loggingWriter := LoggingResponseWriter{200, 0, resp}
	loggingRouter.router.ServeHTTP(&loggingWriter, req)
	log.Printf("[%s] %s %s %d %d", req.RemoteAddr, req.Method, req.URL.Path[1:], loggingWriter.Status, loggingWriter.Length)
}
