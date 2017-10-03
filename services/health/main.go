package main

import (
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	log "github.com/Sirupsen/logrus"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000000000Z07:00",
	})
	cfg, err := arvados.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	clusterCfg, err := cfg.GetCluster("")
	if err != nil {
		log.Fatal(err)
	}
	nodeCfg, err := clusterCfg.GetThisSystemNode()
	if err != nil {
		log.Fatal(err)
	}

	srv := &httpserver.Server{
		Addr: nodeCfg.Health.Listen,
		Server: http.Server{
			Handler: &health.Aggregator{
				Config: cfg,
			},
		},
	}
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
	log.WithField("Listen", srv.Addr).Info("listening")
	if err := srv.Wait(); err != nil {
		log.Fatal(err)
	}
}
