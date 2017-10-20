package main

import (
	"flag"
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	log "github.com/Sirupsen/logrus"
)

func main() {
	configFile := flag.String("config", arvados.DefaultConfigFile, "`path` to arvados configuration file")
	flag.Parse()

	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000000000Z07:00",
	})
	cfg, err := arvados.GetConfig(*configFile)
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

	log := log.WithField("Service", "Health")
	srv := &httpserver.Server{
		Addr: nodeCfg.Health.Listen,
		Server: http.Server{
			Handler: &health.Aggregator{
				Config: cfg,
				Log: func(req *http.Request, err error) {
					log.WithField("RemoteAddr", req.RemoteAddr).
						WithField("Path", req.URL.Path).
						WithError(err).
						Info("HTTP request")
				},
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
