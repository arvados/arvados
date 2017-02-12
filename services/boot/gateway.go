//+build ignore

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
)

var gateway = &nginxGatewayBooter{tmpl: `
daemon off;
error_log stderr info;          # Yes, must be specified here _and_ cmdline
events {
}
http {
  access_log {{keyOrDefault "arvados/service/gateway/access_log" "/var/log/arvados/gateway.log" | toJSON}} combined;
  upstream git-httpd {
    {{service "arvados-git-http"}}
    server {{.Address}}:{{.Port}};
    {{end}}
  }
  server {
    {{if keyExists "arvados/port/tlsGit"}}
    listen *:{{key "arvados/port/tlsGit"}} ssl default_server;
    {{end}}
    listen *:{{keyOrDefault "arvados/port/tlsGateway" 443}} ssl;
    server_name git.{{key "arvados/service/gateway/domain"}};
    ssl_certificate {{key "arvados/service/gateway/pki/certPath"}};
    ssl_certificate_key {{key "arvados/service/gateway/pki/keyPath"}};
    location  / {
      proxy_pass http://git-httpd;
    }
  }
  upstream keep-proxy {
    {{service "arvados-keepproxy"}}
    server {{.Address}}:{{.Port}};
    {{end}}
  }
  server {
    {{if keyExists "arvados/port/tlsKeepProxy"}}
    listen *:{{key "arvados/port/tlsKeepProxy"}} ssl default_server;
    {{end}}
    listen *:{{keyOrDefault "arvados/port/tlsGateway" 443}} ssl;
    server_name keep.{{key "arvados/service/gateway/domain"}};
    ssl_certificate {{key "arvados/service/gateway/pki/certPath"}};
    ssl_certificate_key {{key "arvados/service/gateway/pki/keyPath"}};
    location  / {
      proxy_pass http://keep-proxy;
    }
  }
  upstream keep-web {
    {{service "arvados-keep-web"}}
    server {{.Address}}:{{.Port}};
    {{end}}
  }
  server {
    {{if keyExists "arvados/port/tlsKeepWeb"}}
    listen *:{{key "arvados/port/tlsKeepWeb"}} ssl default_server;
    {{end}}
    listen *:{{keyOrDefault "arvados/port/tlsGateway" 443}} ssl;
    server_name download.{{key "arvados/service/gateway/domain"}}
        collections.{{key "arvados/service/gateway/domain"}}
        *.collections.{{key "arvados/service/gateway/domain"}}
        ~.*--collections.{{key "arvados/service/gateway/domain"}};
        *.collections.{{key "arvados/service/gateway/domain"}};
    ssl_certificate {{key "arvados/service/gateway/pki/certPath"}};
    ssl_certificate_key {{key "arvados/service/gateway/pki/keyPath"}};
    location  / {
      proxy_pass http://keep-web;
      proxy_set_header Host            $host;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
  }
  upstream ws {
    {{service "arvados-ws"}}
    server {{.Address}}:{{.Port}};
    {{end}}
  }
  server {
    {{if keyExists "arvados/port/tlsWS"}}
    listen *:{{key "arvados/port/tlsWS"}} ssl default_server;
    {{end}}
    listen *:{{keyOrDefault "arvados/port/tlsGateway" 443}} ssl;
    server_name ws.{{key "arvados/service/gateway/domain"}};
    ssl_certificate {{key "arvados/service/gateway/pki/certPath"}};
    ssl_certificate_key {{key "arvados/service/gateway/pki/keyPath"}};
    location  / {
      proxy_pass http://ws;
      proxy_set_header Upgrade $http_upgrade;
      proxy_set_header Connection "upgrade";
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
  }
}
`}

type nginxGatewayBooter struct {
	tmpl string
}

func (ngb *nginxGatewayBooter) Boot(ctx context.Context) error {
	cfg := cfg(ctx)

	if ngb.conf == "" {
		ngb.conf = ngb.name
	}
	if ngb.tmpl == "" {
		ngb.tmpl = "{}"
	}

	rootToken, err := ioutil.ReadFile(path.Join(cfg.DataDir, "vault-root-token.txt"))
	if err != nil {
		return err
	}

	consulCfg := path.Join(cfg.DataDir, "gateway.consul-template.hcl")
	if err = atomicWriteJSON(consulCfg+".ctmpl", map[string]interface{}{
		"exec": map[string]interface{}{
			"reload_signal": "SIGHUP",
		},
		"consul": map[string]interface{}{
			"address": fmt.Sprintf("0.0.0.0:%d", cfg.Ports.ConsulHTTP),
		},
		"vault": map[string]string{
			"address": fmt.Sprintf("http://0.0.0.0:%d", cfg.Ports.VaultServer),
			"token":   rootToken,
		}}, 0600); err != nil {
		return err
	}

	nginxCfg := path.Join(cfg.DataDir, "gateway.nginx.conf")
	if err = atomicWriteFile(nginxCfg+".ctmpl", []byte(ngb.tmpl), 0644); err != nil {
		return err
	}

	if err := (&osPackage{
		Debian: "nginx",
	}).Boot(ctx); err != nil {
		return err
	}

	nginxBin, err := exec.LookPath("nginx")
	if err != nil {
		return err
	}

	return (&supervisedService{
		name: ngb.name,
		cmd:  path.Join(cfg.UsrDir, "bin", "consul-template"),
		args: []string{
			"-config=" + consulCfg,
			"-template=" + nginxCfg + ".ctmpl:" + nginxCfg,
			"-exec",
			"nginx",
			"-g", "error_log stderr info;",
			"-g", "pid " + path.Join(cfg.DataDir, "nginx.pid") + ";",
			"-c", nginxCfg,
		},
		env: map[string]string{
			"VAULT_TOKEN": rootToken,
		},
	}).Boot(ctx)
}
