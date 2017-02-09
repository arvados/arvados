package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/config"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"github.com/coreos/go-systemd/daemon"
)

var logger = ctxlog.FromContext

func main() {
	log := logger(nil)

	configPath := flag.String("config", "/etc/arvados/ws/ws.yml", "`path` to config file")
	dumpConfig := flag.Bool("dump-config", false, "show current configuration and exit")
	cfg := defaultConfig()
	flag.Parse()

	err := config.LoadFile(&cfg, *configPath)
	if err != nil {
		log.Fatal(err)
	}

	ctxlog.SetLevel(cfg.LogLevel)
	ctxlog.SetFormat(cfg.LogFormat)

	if *dumpConfig {
		txt, err := config.Dump(&cfg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(txt))
		return
	}

	log.Info("started")
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
			Config:         &cfg,
			eventSource:    eventSource,
			newPermChecker: func() permChecker { return newPermChecker(cfg.Client) },
		},
	}

	go func() {
		eventSource.Run()
		log.Fatal("event source stopped")
	}()

	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		log.WithError(err).Warn("error notifying init daemon")
	}

	log.WithField("Listen", srv.Addr).Info("listening")
	log.Fatal(srv.ListenAndServe())
}
