package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"git.curoverse.com/arvados.git/sdk/go/config"
)

const defaultCfgPath = "/etc/arvados/boot/boot.yml"

func main() {
	cfgPath := flag.String("config", defaultCfgPath, "`path` to config file")
	flag.Parse()

	cfg := DefaultConfig()
	if err := config.LoadFile(cfg, *cfgPath); os.IsNotExist(err) && *cfgPath == defaultCfgPath {
		log.Printf("WARNING: No config file specified or found, starting fresh!")
	} else if err != nil {
		log.Fatal(err)
	}

	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "  ")
	enc.Encode(cfg)

	var ctl Booter = &controller{}
	err := ctl.Boot(withCfg(context.Background(), cfg))
	if err != nil {
		log.Printf("controller boot failed: %v", err)
	}
}
