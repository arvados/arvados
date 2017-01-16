package main

// LoggingRESTRouter
// LoggingResponseWriter

import (
	"context"
	"net/http"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"git.curoverse.com/arvados.git/sdk/go/stats"
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

	// Attach a requestID-aware logger to the request context.
	lgr := log.WithField("RequestID", loggingRouter.idGenerator.Next())
	ctx := context.WithValue(req.Context(), "logger", lgr)
	req = req.WithContext(ctx)

	lgr = lgr.WithFields(log.Fields{
		"remoteAddr":      req.RemoteAddr,
		"reqForwardedFor": req.Header.Get("X-Forwarded-For"),
		"reqMethod":       req.Method,
		"reqPath":         req.URL.Path[1:],
		"reqBytes":        req.ContentLength,
	})
	lgr.Debug("request")

	resp := LoggingResponseWriter{http.StatusOK, 0, wrappedResp, "", zeroTime}
	loggingRouter.router.ServeHTTP(&resp, req)
	tDone := time.Now()

	statusText := http.StatusText(resp.Status)
	if resp.Status >= 400 {
		statusText = strings.Replace(resp.ResponseBody, "\n", "", -1)
	}
	if resp.sentHdr == zeroTime {
		// Nobody changed status or wrote any data, i.e., we
		// returned a 200 response with no body.
		resp.sentHdr = tDone
	}

	lgr.WithFields(log.Fields{
		"timeTotal":      stats.Duration(tDone.Sub(tStart)),
		"timeToStatus":   stats.Duration(resp.sentHdr.Sub(tStart)),
		"timeWriteBody":  stats.Duration(tDone.Sub(resp.sentHdr)),
		"respStatusCode": resp.Status,
		"respStatus":     statusText,
		"respBytes":      resp.Length,
	}).Info("response")
}
