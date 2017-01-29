package main

import (
	"context"
	"io/ioutil"
	"os"
	"sync"
)

var arvadosRepo = &arvadosRepoBooter{}

type arvadosRepoBooter struct {
	sync.Mutex
}

func (*arvadosRepoBooter) Boot(ctx context.Context) error {
	cfg := cfg(ctx)
	repo := cfg.ArvadosAptRepo
	if !repo.Enabled {
		return nil
	}
	srcPath := "/etc/apt/sources.list.d/arvados.list"
	if _, err := os.Stat(srcPath); err == nil {
		return nil
	}
	if err := command("apt-key", "adv", "--keyserver", "pool.sks-keyservers.net", "--recv", "1078ECD7").Run(); err != nil {
		return err
	}
	if err := ioutil.WriteFile(srcPath, []byte("deb "+repo.URL+" "+repo.Release+" main\n"), 0644); err != nil {
		return err
	}
	return command("apt-get", "update").Run()
}
