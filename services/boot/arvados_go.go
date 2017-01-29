package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
)

var (
	dispatchLocal = &arvadosGoBooter{name: "crunch-dispatch-local"}
	dispatchSLURM = &arvadosGoBooter{name: "crunch-dispatch-slurm"}
	gitHTTP       = &arvadosGoBooter{name: "arvados-git-httpd"}
	keepbalance   = &arvadosGoBooter{name: "keep-balance"}
	keepproxy     = &arvadosGoBooter{name: "keepproxy"}
	keepstore     = &arvadosGoBooter{name: "keepstore"}
	websocket     = &arvadosGoBooter{name: "arvados-ws"}
)

type arvadosGoBooter struct {
	name string
}

func availablePort() int {
	return rand.Intn(10000) + 20000
}

func (agb *arvadosGoBooter) Boot(ctx context.Context) error {
	cfg := cfg(ctx)
	cmd := path.Join(cfg.UsrDir, "bin", agb.name)
	if _, err := os.Stat(cmd); err != nil {
		if found, err := filepath.Glob(path.Join(cfg.UsrDir, "pkg", agb.name+"_*.deb")); err == nil && len(found) > 0 {
			cmd := command("dpkg", "-i", found[0])
			osPackageMutex.Lock()
			err = cmd.Run()
			osPackageMutex.Unlock()
			if err != nil {
				log.Print(err)
			}
		}
	}
	cfgPath := path.Join("/etc/arvados", agb.name, agb.name+".yml")
	atomicWriteFile(cfgPath+".ctmpl", []byte("{}"), 0644)
	return Series{
		&osPackage{
			Debian: agb.name,
		},
		&supervisedService{
			name: agb.name,
			cmd:  path.Join(cfg.UsrDir, "bin", "consul-template"),
			args: []string{
				"-consul-addr=127.0.0.1:8500",
				"-template="+cfgPath+".ctmpl:"+cfgPath,
				"-exec",
				agb.name,
			},
		},
	}.Boot(ctx)
}
