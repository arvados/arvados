package main

import (
	"context"
	"fmt"
	"os"
)

type Config struct {
	// 5 alphanumeric chars. Must be either xx*, yy*, zz*, or
	// globally unique.
	SiteID string

	// Hostnames or IP addresses of control hosts. Use at least 3
	// in production. System functions only when a majority are
	// alive.
	ControlHosts []string

	ConsulPorts struct {
		DNS     int
		HTTP    int
		HTTPS   int
		RPC     int
		SerfLAN int `json:"Serf_LAN"`
		SerfWAN int `json:"Serf_WAN"`
		Server  int
	}

	WebGUI struct {
		// addr:port to serve web-based setup/monitoring
		// application
		Listen string
	}

	DataDir string
	UsrDir  string

	RunitSvDir string
}

func (c *Config) Boot(ctx context.Context) error {
	for _, path := range []string{c.DataDir, c.UsrDir, c.UsrDir + "/bin"} {
		if fi, err := os.Stat(path); err != nil {
			err = os.MkdirAll(path, 0755)
			if err != nil {
				return err
			}
		} else if !fi.IsDir() {
			return fmt.Errorf("%s: is not a directory", path)
		}
	}
	return nil
}

func (c *Config) SetDefaults() {
	if len(c.ControlHosts) == 0 {
		c.ControlHosts = []string{"127.0.0.1"}
	}
	defaultPort := []int{18600, 18500, -1, 18400, 18301, 18302, 18300}
	for i, port := range []*int{
		&c.ConsulPorts.DNS,
		&c.ConsulPorts.HTTP,
		&c.ConsulPorts.HTTPS,
		&c.ConsulPorts.RPC,
		&c.ConsulPorts.SerfLAN,
		&c.ConsulPorts.SerfWAN,
		&c.ConsulPorts.Server,
	} {
		if *port == 0 {
			*port = defaultPort[i]
		}
	}
	if c.DataDir == "" {
		c.DataDir = "/var/lib/arvados"
	}
	if c.UsrDir == "" {
		c.UsrDir = "/usr/local/arvados"
	}
	if c.RunitSvDir == "" {
		c.RunitSvDir = "/etc/sv"
	}
	if c.WebGUI.Listen == "" {
		c.WebGUI.Listen = "localhost:18000"
	}
}
