package main

import (
	"context"
	"path"
)

type controller struct{}

func (c *controller) Boot(ctx context.Context) error {
	cfg := cfg(ctx)
	return Series{
		Concurrent{
			cfg,
			installCerts,
		},
		Concurrent{
			consul,
			&download{
				URL:  "https://releases.hashicorp.com/consul-template/0.18.0/consul-template_0.18.0_linux_amd64.zip",
				Dest: path.Join(cfg.UsrDir, "bin", "consul-template"),
				Mode: 0755,
			},
		},
		Concurrent{
			dispatchLocal,
			dispatchSLURM,
			gitHTTP,
			keepbalance,
			keepproxy,
			keepstore,
			websocket,
		},
	}.Boot(ctx)
}
