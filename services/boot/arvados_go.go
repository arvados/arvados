package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
)

var (
	dispatchLocal = &arvadosGoBooter{name: "crunch-dispatch-local"}
	dispatchSLURM = &arvadosGoBooter{name: "crunch-dispatch-slurm"}
	gitHTTP       = &arvadosGoBooter{name: "arvados-git-httpd", conf: "git-httpd"}
	keepbalance   = &arvadosGoBooter{name: "keep-balance", tmpl: keepbalanceTmpl}
	keepproxy     = &arvadosGoBooter{name: "keepproxy"}
	keepstore     = &arvadosGoBooter{name: "keepstore"}
	websocket     = &arvadosGoBooter{name: "arvados-ws", conf: "ws"}

	keepbalanceTmpl = `{"RunPeriod":"1m"}`
)

type arvadosGoBooter struct {
	name string
	conf string
	tmpl string
}

func (agb *arvadosGoBooter) Boot(ctx context.Context) error {
	cfg := cfg(ctx)

	if agb.conf == "" {
		agb.conf = agb.name
	}
	if agb.tmpl == "" {
		agb.tmpl = "{}"
	}

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
	cfgPath := path.Join("/etc/arvados", agb.conf, agb.conf+".yml")
	if err := os.MkdirAll(path.Dir(cfgPath), 0755); err != nil {
		return err
	}
	// ctmpl := []byte(fmt.Sprintf(`{{tree %q | explode | toJSONPretty}}`, agb.name))
	ctmpl := []byte(`{}`)
	if err := atomicWriteFile(cfgPath+".ctmpl", ctmpl, 0644); err != nil {
		return err
	}
	return Series{
		&osPackage{
			Debian: agb.name,
		},
		&supervisedService{
			name: agb.name,
			cmd:  path.Join(cfg.UsrDir, "bin", "consul-template"),
			args: []string{
				"-consul-addr=" + fmt.Sprintf("0.0.0.0:%d", cfg.Ports.ConsulHTTP),
				"-template=" + cfgPath + ".ctmpl:" + cfgPath,
				"-exec",
				agb.name,
			},
		},
	}.Boot(ctx)
}
