package main

type Config struct {
	// 5 alphanumeric chars. Must be either xx*, yy*, zz*, or
	// globally unique.
	SiteID string

	// Hostnames or IP addresses of control hosts. Use at least 3
	// in production. System functions only when a majority are
	// alive.
	ControlHosts []string

	// addr:port to serve web-based setup/monitoring application
	WebListen string

	UsrDir  string
	DataDir string
}

func (c *Config) SetDefaults() {
	if len(c.ControlHosts) == 0 {
		c.ControlHosts = []string{"127.0.0.1"}
	}
	if c.DataDir == "" {
		c.DataDir = "/var/lib/arvados"
	}
	if c.UsrDir == "" {
		c.DataDir = "/usr/local"
	}
	if c.WebListen == "" {
		c.WebListen = "localhost:8000"
	}
}
