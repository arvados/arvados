package main

// LoggingRESTRouter
// LoggingResponseWriter

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// LoggingResponseWriter has anonymous fields ResponseWriter and ResponseBody
type LoggingResponseWriter struct {
	Status int
	Length int
	http.ResponseWriter
	ResponseBody string
	sentHdr      time.Time
}

func (w *LoggingResponseWriter) CloseNotify() <-chan bool {
	wrapped, ok := w.ResponseWriter.(http.CloseNotifier)
	if !ok {
		// If upstream doesn't implement CloseNotifier, we can
		// satisfy the interface by returning a channel that
		// never sends anything (the interface doesn't
		// guarantee that anything will ever be sent on the
		// channel even if the client disconnects).
		return nil
	}
	return wrapped.CloseNotify()
}

// WriteHeader writes header to ResponseWriter
func (loggingWriter *LoggingResponseWriter) WriteHeader(code int) {
	if loggingWriter.sentHdr == zeroTime {
		loggingWriter.sentHdr = time.Now()
	}
	loggingWriter.Status = code
	loggingWriter.ResponseWriter.WriteHeader(code)
}

var zeroTime time.Time

func (loggingWriter *LoggingResponseWriter) Write(data []byte) (int, error) {
	if loggingWriter.Length == 0 && len(data) > 0 && loggingWriter.sentHdr == zeroTime {
		loggingWriter.sentHdr = time.Now()
	}
	loggingWriter.Length += len(data)
	if loggingWriter.Status >= 400 {
		loggingWriter.ResponseBody += string(data)
	}
	return loggingWriter.ResponseWriter.Write(data)
}

// LoggingRESTRouter is used to add logging capabilities to mux.Router
type LoggingRESTRouter struct {
	router http.Handler
}

func (loggingRouter *LoggingRESTRouter) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	t0 := time.Now()
	loggingWriter := LoggingResponseWriter{http.StatusOK, 0, resp, "", zeroTime}
	loggingRouter.router.ServeHTTP(&loggingWriter, req)
	statusText := http.StatusText(loggingWriter.Status)
	if loggingWriter.Status >= 400 {
		statusText = strings.Replace(loggingWriter.ResponseBody, "\n", "", -1)
	}
	now := time.Now()
	tTotal := now.Sub(t0)
	tLatency := loggingWriter.sentHdr.Sub(t0)
	tResponse := now.Sub(loggingWriter.sentHdr)
	log.Printf("[%s] %s %s %d %.6fs %.6fs %.6fs %d %d \"%s\"", req.RemoteAddr, req.Method, req.URL.Path[1:], req.ContentLength, tTotal.Seconds(), tLatency.Seconds(), tResponse.Seconds(), loggingWriter.Status, loggingWriter.Length, statusText)

}
