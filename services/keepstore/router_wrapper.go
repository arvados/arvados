package main

// RESTRouterWrapper
// LoggingResponseWriter

import (
  "bytes"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type LoggingResponseWriter struct {
  Status int
  Data *bytes.Buffer
  http.ResponseWriter
}

func (loggingWriter *LoggingResponseWriter) WriteHeader(code int) {
  loggingWriter.Status = code
  loggingWriter.ResponseWriter.WriteHeader(code)
}

func (loggingWriter *LoggingResponseWriter) Write(data []byte) (int, error){
  loggingWriter.Data.Write(data)
  return loggingWriter.ResponseWriter.Write(data)
}

type LoggingRESTRouter struct {
  router *mux.Router
}

func MakeLoggingRESTRouter() (LoggingRESTRouter) {
  router := MakeRESTRouter()
  return (LoggingRESTRouter{router})
}

func (wrapper LoggingRESTRouter) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
  loggingWriter := LoggingResponseWriter{200, bytes.NewBuffer(make([]byte, 0, 0)), resp}
  wrapper.router.ServeHTTP(&loggingWriter, req)
  if loggingWriter.Status == 200 {
    if loggingWriter.Data.Len() > 200 {  // could be large block, so just print the size
      log.Printf("[%s] %s %s %d %d", req.RemoteAddr, req.Method, req.URL.Path[1:],
          loggingWriter.Status, loggingWriter.Data.Len())
    } else {  // this could be a hash or status or a small block etc
      log.Printf("[%s] %s %s %d %s", req.RemoteAddr, req.Method, req.URL.Path[1:],
          loggingWriter.Status, loggingWriter.Data)
    }
  } else {
    log.Printf("[%s] %s %s %d", req.RemoteAddr, req.Method, req.URL.Path[1:], loggingWriter.Status)
  }
}
