package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/config"
)

var debugLogf = func(string, ...interface{}) {}

func main() {
	configPath := flag.String("config", "/etc/arvados/ws/ws.yml", "`path` to config file")
	dumpConfig := flag.Bool("dump-config", false, "show current configuration and exit")
	cfg := DefaultConfig()
	flag.Parse()

	err := config.LoadFile(&cfg, *configPath)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.Debug {
		debugLogf = log.Printf
	}

	if *dumpConfig {
		txt, err := config.Dump(&cfg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(txt))
		return
	}

	eventSource := &pgEventSource{
		DataSource: cfg.Postgres.ConnectionString(),
		QueueSize:  cfg.ServerEventQueue,
	}
	srv := &http.Server{
		Addr:           cfg.Listen,
		ReadTimeout:    time.Minute,
		WriteTimeout:   time.Minute,
		MaxHeaderBytes: 1 << 20,
		Handler: &router{
			Config:      &cfg,
			eventSource: eventSource,
		},
	}
	eventSource.NewSink().Stop()

	log.Printf("listening at %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
