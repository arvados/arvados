package main

// LoggingRESTRouter
// LoggingResponseWriter

import (
	"net/http"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	log "github.com/Sirupsen/logrus"
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
	router      http.Handler
	idGenerator httpserver.IDGenerator
}

func (loggingRouter *LoggingRESTRouter) ServeHTTP(wrappedResp http.ResponseWriter, req *http.Request) {
	tStart := time.Now()
	lgr := log.WithFields(log.Fields{
		"RequestID":       loggingRouter.idGenerator.Next(),
		"RemoteAddr":      req.RemoteAddr,
		"X-Forwarded-For": req.Header.Get("X-Forwarded-For"),
		"ReqMethod":       req.Method,
		"ReqPath":         req.URL.Path[1:],
		"ReqBytes":        req.ContentLength,
	})
	lgr.Info("request")

	resp := LoggingResponseWriter{http.StatusOK, 0, wrappedResp, "", zeroTime}
	loggingRouter.router.ServeHTTP(&resp, req)
	statusText := http.StatusText(resp.Status)
	if resp.Status >= 400 {
		statusText = strings.Replace(resp.ResponseBody, "\n", "", -1)
	}

	tDone := time.Now()
	lgr.WithFields(log.Fields{
		"TimeTotal":      tDone.Sub(tStart).Seconds(),
		"TimeToStatus":   resp.sentHdr.Sub(tStart).Seconds(),
		"TimeWriteBody":  tDone.Sub(resp.sentHdr).Seconds(),
		"RespStatusCode": resp.Status,
		"RespStatus":     statusText,
		"RespBytes":      resp.Length,
	}).Info("response")
}
