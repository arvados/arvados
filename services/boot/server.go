package main

import (
	"context"
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

func main() {
	cfgPath := flag.String("config", defaultCfgPath, "`path` to config file")
	flag.Parse()

	var cfg Config
	if err := config.LoadFile(&cfg, *cfgPath); os.IsNotExist(err) && *cfgPath == defaultCfgPath {
		log.Printf("WARNING: No config file specified or found, starting fresh!")
	} else if err != nil {
		log.Fatal(err)
	}
	cfg.SetDefaults()
	go func() {
		log.Printf("starting server at %s", cfg.WebGUI.Listen)
		log.Fatal(http.ListenAndServe(cfg.WebGUI.Listen, stack(cfg.logger, cfg.apiOrAssets)))
	}()
	go func() {
		var ctl Booter = &controller{}
		ticker := time.NewTicker(5 * time.Second)
		for {
			err := ctl.Boot(withCfg(context.Background(), &cfg))
			log.Printf("ctl.Boot: %v", err)
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
func (cfg *Config) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%.6f %q %q %q", time.Since(t).Seconds(), r.RemoteAddr, r.Method, r.URL.Path)
	})
}

// dispatches /api/ to the API stack, everything else to the static
// assets stack.
func (cfg *Config) apiOrAssets(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/", stack(cfg.apiHeaders, cfg.apiRoutes))
	mux.Handle("/", http.FileServer(assetFS()))
	return mux
}

// adds response headers suitable for API responses
func (cfg *Config) apiHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// dispatches API routes
func (cfg *Config) apiRoutes(http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"time": time.Now().UTC()})
	})
	mux.HandleFunc("/api/status/controller", func(w http.ResponseWriter, r *http.Request) {
		timeout := time.Minute
		if v, err := strconv.ParseInt(r.FormValue("timeout"), 10, 64); err == nil {
			timeout = time.Duration(v) * time.Second
		}
		if v, err := strconv.ParseInt(r.FormValue("newerThan"), 10, 64); err == nil {
			log.Println(v, timeout)
			// TODO: wait
			// TaskState.Wait(version(v), timeout, r.Context())
		}
		// TODO:
		// rep, v := report(ctlTasks)
		json.NewEncoder(w).Encode(map[string]interface{}{
			// "Version": v,
			// "Tasks":   rep,
			// TODO:
			"Version": 1,
			"Tasks": []int{},
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	})
	return mux
}
