package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/config"
)

func main() {
	configPath := flag.String("config", "/etc/arvados/ws/ws.yml", "`path` to config file")
	dumpConfig := flag.Bool("dump-config", false, "show current configuration and exit")
	cfg := DefaultConfig()
	flag.Parse()

	err := config.LoadFile(&cfg, *configPath)
	if err != nil {
		log.Fatal(err)
	}

	if *dumpConfig {
		txt, err := config.Dump(&cfg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(txt))
		return
	}

	srv := &http.Server{
		Addr:           cfg.Listen,
		ReadTimeout:    time.Minute,
		WriteTimeout:   time.Minute,
		MaxHeaderBytes: 1 << 20,
		Handler: &router{
			EventSource: (&pgEventSource{
				PgConfig:  cfg.Postgres,
				QueueSize: cfg.ServerEventQueue,
			}).EventSource(),
		},
	}
	log.Fatal(srv.ListenAndServe())
}
