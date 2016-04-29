package main

import (
	"net/http"
	"testing"
)

func TestLoggingResponseWriterImplementsCloseNotifier(t *testing.T) {
	http.ResponseWriter(&LoggingResponseWriter{}).(http.CloseNotifier).CloseNotify()
}
