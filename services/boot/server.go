package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/config"
)

const defaultCfgPath = "/etc/arvados/boot/boot.yml"

var cfg Config

func main() {
	cfgPath := flag.String("config", defaultCfgPath, "`path` to config file")
	flag.Parse()

	if err := config.LoadFile(&cfg, *cfgPath); os.IsNotExist(err) && *cfgPath == defaultCfgPath {
		log.Printf("WARNING: No config file specified or found, starting fresh!")
	} else if err != nil {
		log.Fatal(err)
	}
	cfg.SetDefaults()
	go func() {
		log.Printf("starting server at %s", cfg.WebListen)
		log.Fatal(http.ListenAndServe(cfg.WebListen, stack(logger, apiOrAssets)))
	}()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			runTasks(&cfg, ctlTasks)
			<-ticker.C
		}
	}()
	<-(chan struct{})(nil)
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
	mux.HandleFunc("/api/tasks/ctl", func(w http.ResponseWriter, r *http.Request) {
		timeout := time.Minute
		if v, err := strconv.ParseInt(r.FormValue("timeout"), 10, 64); err == nil {
			timeout = time.Duration(v) * time.Second
		}
		if v, err := strconv.ParseInt(r.FormValue("newerThan"), 10, 64); err == nil {
			TaskState.Wait(version(v), timeout, r.Context())
		}
		rep, v := report(ctlTasks)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Version": v,
			"Tasks":   rep,
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	})
	return mux
}
