package main

// RESTRouterWrapper
// LoggingResponseWriter

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type LoggingResponseWriter struct {
  status int
  data []byte
  http.ResponseWriter
}

func (loggingWriter *LoggingResponseWriter) WriteHeader(code int) {
  loggingWriter.status = code
  loggingWriter.ResponseWriter.WriteHeader(code)
}

func (loggingWriter *LoggingResponseWriter) Write(data []byte) (int, error){
  loggingWriter.data = data
  return loggingWriter.ResponseWriter.Write(data)
}

type RESTRouterWrapper struct {
  router *mux.Router
}

func MakeRESTRouterWrapper(r *mux.Router) (RESTRouterWrapper) {
  return (RESTRouterWrapper{r})
}

func (this RESTRouterWrapper) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
  loggingWriter := LoggingResponseWriter{200, nil, resp}
  this.router.ServeHTTP(&loggingWriter, req)
  if loggingWriter.data != nil && loggingWriter.status == 200 {
    data_len := len(loggingWriter.data)
    if data_len > 200 {  // this could be a block, so just print the size
      log.Printf("[%s] %s %s %d %d", req.RemoteAddr, req.Method, req.URL.Path[1:], loggingWriter.status, data_len)
    } else {  // this could be a hash or status or a small block etc
      log.Printf("[%s] %s %s %d %s", req.RemoteAddr, req.Method, req.URL.Path[1:], loggingWriter.status, loggingWriter.data)
    }
  } else {
    log.Printf("[%s] %s %s %d", req.RemoteAddr, req.Method, req.URL.Path[1:], loggingWriter.status)
  }
}
