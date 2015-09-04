package main

// LoggingRESTRouter
// LoggingResponseWriter

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
	"time"
)

type LoggingResponseWriter struct {
	Status int
	Length int
	http.ResponseWriter
	ResponseBody string
}

func (loggingWriter *LoggingResponseWriter) WriteHeader(code int) {
	loggingWriter.Status = code
	loggingWriter.ResponseWriter.WriteHeader(code)
}

func (loggingWriter *LoggingResponseWriter) Write(data []byte) (int, error) {
	loggingWriter.Length += len(data)
	if loggingWriter.Status >= 400 {
		loggingWriter.ResponseBody += string(data)
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
	t0 := time.Now()
	loggingWriter := LoggingResponseWriter{http.StatusOK, 0, resp, ""}
	loggingRouter.router.ServeHTTP(&loggingWriter, req)
	statusText := http.StatusText(loggingWriter.Status)
	if loggingWriter.Status >= 400 {
		statusText = strings.Replace(loggingWriter.ResponseBody, "\n", "", -1)
	}
	log.Printf("[%s] %s %s %.6fs %d %d \"%s\"", req.RemoteAddr, req.Method, req.URL.Path[1:], time.Since(t0).Seconds(), loggingWriter.Status, loggingWriter.Length, statusText)

}
