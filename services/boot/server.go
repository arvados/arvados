package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"time"
)

func main() {
	listen := flag.String("listen", ":80", "addr:port or :port to listen on")
	flag.Parse()
	log.Printf("starting server at %s", *listen)
	log.Fatal(http.ListenAndServe(*listen, stack(logger, apiOrAssets)))
}

type middleware func(http.Handler) http.Handler

var notFound = http.NotFoundHandler()

// returns a handler that implements a stack of middlewares.
func stack(m ...middleware) http.Handler {
	if len(m) == 0 {
		return notFound
	}
	return m[0](stack(m[1:]...))
}

// logs each request.
func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%.6f %q %q %q", time.Since(t).Seconds(), r.RemoteAddr, r.Method, r.URL.Path)
	})
}

// dispatches /api/ to the API stack, everything else to the static
// assets stack.
func apiOrAssets(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", stack(apiHeaders, apiRoutes))
	mux.Handle("/", http.FileServer(assetFS()))
	return mux
}

// adds response headers suitable for API responses
func apiHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// dispatches API routes
func apiRoutes(http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"time": time.Now().UTC()})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	})
	return mux
}
