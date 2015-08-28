package httpserver

import (
	"net/http"
)

// ResponseWriter wraps http.ResponseWriter and exposes the status
// sent, the number of bytes sent to the client, and the last write
// error.
type ResponseWriter struct {
	http.ResponseWriter
	wroteStatus *int	// Last status given to WriteHeader()
	wroteBodyBytes *int	// Bytes successfully written
	err *error		// Last error returned from Write()
}

func WrapResponseWriter(orig http.ResponseWriter) ResponseWriter {
	return ResponseWriter{orig, new(int), new(int), new(error)}
}

func (w ResponseWriter) WriteHeader(s int) {
	*w.wroteStatus = s
	w.ResponseWriter.WriteHeader(s)
}

func (w ResponseWriter) Write(data []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(data)
	*w.wroteBodyBytes += n
	*w.err = err
	return
}

func (w ResponseWriter) WroteStatus() int {
	return *w.wroteStatus
}

func (w ResponseWriter) WroteBodyBytes() int {
	return *w.wroteBodyBytes
}

func (w ResponseWriter) Err() error {
	return *w.err
}
