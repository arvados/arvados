package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/config"
	"github.com/Sirupsen/logrus"
)

func main() {
	log := logger(nil)

	configPath := flag.String("config", "/etc/arvados/ws/ws.yml", "`path` to config file")
	dumpConfig := flag.Bool("dump-config", false, "show current configuration and exit")
	cfg := DefaultConfig()
	flag.Parse()

	err := config.LoadFile(&cfg, *configPath)
	if err != nil {
		log.Fatal(err)
	}

	lvl, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	rootLogger.Level = lvl
	switch cfg.LogFormat {
	case "text":
		rootLogger.Formatter = &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		}
	case "json":
		rootLogger.Formatter = &logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		}
	default:
		log.WithField("LogFormat", cfg.LogFormat).Fatal("unknown log format")
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
			Config:         &cfg,
			eventSource:    eventSource,
			newPermChecker: func() permChecker { return NewPermChecker(cfg.Client) },
		},
	}
	eventSource.NewSink().Stop()

	log.WithField("Listen", srv.Addr).Info("listening")
	log.Fatal(srv.ListenAndServe())
}
