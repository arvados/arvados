package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	log "github.com/Sirupsen/logrus"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000000000Z07:00",
	})
	sysConf, err := arvados.GetSystemConfig()
	if err != nil {
		log.Fatal(err)
	}

	srv := &httpserver.Server{
		Addr: ":", // FIXME: should be dictated by Health on this SystemNode
		Handler: &health.Aggregator{
			SystemConfig: sysConf,
		},
	}
	srv.HandleFunc()
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
	log.WithField("Listen", srv.Addr).Info("listening")
	if err := srv.Wait(); err != nil {
		log.Fatal(err)
	}
}
