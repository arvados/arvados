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

// CloseNotify implements http.CloseNotifier.
func (resp *LoggingResponseWriter) CloseNotify() <-chan bool {
	wrapped, ok := resp.ResponseWriter.(http.CloseNotifier)
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
func (resp *LoggingResponseWriter) WriteHeader(code int) {
	if resp.sentHdr == zeroTime {
		resp.sentHdr = time.Now()
	}
	resp.Status = code
	resp.ResponseWriter.WriteHeader(code)
}

var zeroTime time.Time

func (resp *LoggingResponseWriter) Write(data []byte) (int, error) {
	if resp.Length == 0 && len(data) > 0 && resp.sentHdr == zeroTime {
		resp.sentHdr = time.Now()
	}
	resp.Length += len(data)
	if resp.Status >= 400 {
		resp.ResponseBody += string(data)
	}
	return resp.ResponseWriter.Write(data)
}

// LoggingRESTRouter is used to add logging capabilities to mux.Router
type LoggingRESTRouter struct {
	router http.Handler
}

func (loggingRouter *LoggingRESTRouter) ServeHTTP(wrappedResp http.ResponseWriter, req *http.Request) {
	t0 := time.Now()
	resp := LoggingResponseWriter{http.StatusOK, 0, wrappedResp, "", zeroTime}
	loggingRouter.router.ServeHTTP(&resp, req)
	statusText := http.StatusText(resp.Status)
	if resp.Status >= 400 {
		statusText = strings.Replace(resp.ResponseBody, "\n", "", -1)
	}
	now := time.Now()
	tTotal := now.Sub(t0)
	tLatency := resp.sentHdr.Sub(t0)
	tResponse := now.Sub(resp.sentHdr)
	log.Printf("[%s] %s %s %d %.6fs %.6fs %.6fs %d %d \"%s\"", req.RemoteAddr, req.Method, req.URL.Path[1:], req.ContentLength, tTotal.Seconds(), tLatency.Seconds(), tResponse.Seconds(), resp.Status, resp.Length, statusText)

}
