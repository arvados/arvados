package httpserver

import (
	"net/http"
)

type limiterHandler struct {
	requests chan struct{}
	handler  http.Handler
}

func NewRequestLimiter(maxRequests int, handler http.Handler) http.Handler {
	return &limiterHandler{
		requests: make(chan struct{}, maxRequests),
		handler:  handler,
	}
}

func (h *limiterHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	select {
	case h.requests <- struct{}{}:
	default:
		// reached max requests
		resp.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	h.handler.ServeHTTP(resp, req)
	<-h.requests
}
