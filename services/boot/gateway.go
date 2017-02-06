//+build ignore

package main

import (
	"context"
	"fmt"
	"path"
)

var gateway = &nginxGatewayBooter{tmpl: `
daemon off;
error_log stderr info;          # Yes, must be specified here _and_ cmdline
events {
}
http {
  access_log {{keyOrDefault "service/gateway/access_log" "/var/log/arvados/gateway.log" | toJSON}} combined;
  upstream arv-git-http {
    server localhost:{{GITPORT}};
  }
  server {
    {{if keyExists"service/gateway/ports/tlsGit"}}
    listen *:{{key "service/gateway/ports/tlsGit"}} ssl default_server;
    {{end}}
    listen *:{{keyOrDefault "service/gateway/ports/tlsGateway" 443}} ssl;
    server_name git.{{key "service/gateway/domain"}};
    ssl_certificate {{SSLCERT}};
    ssl_certificate_key {{SSLKEY}};
    location  / {
      proxy_pass http://arv-git-http;
    }
  }
  upstream keepproxy {
    server localhost:{{KEEPPROXYPORT}};
  }
  server {
    listen *:{{KEEPPROXYSSLPORT}} ssl default_server;
    server_name _;
    ssl_certificate {{SSLCERT}};
    ssl_certificate_key {{SSLKEY}};
    location  / {
      proxy_pass http://keepproxy;
    }
  }
  upstream keep-web {
    server localhost:{{KEEPWEBPORT}};
  }
  server {
    listen *:{{KEEPWEBSSLPORT}} ssl default_server;
    server_name ~^(?<request_host>.*)$;
    ssl_certificate {{SSLCERT}};
    ssl_certificate_key {{SSLKEY}};
    location  / {
      proxy_pass http://keep-web;
      proxy_set_header Host $request_host:{{KEEPWEBPORT}};
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
  }
  server {
    listen *:{{KEEPWEBDLSSLPORT}} ssl default_server;
    server_name ~.*;
    ssl_certificate {{SSLCERT}};
    ssl_certificate_key {{SSLKEY}};
    location  / {
      proxy_pass http://keep-web;
      proxy_set_header Host download:{{KEEPWEBPORT}};
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_redirect //download:{{KEEPWEBPORT}}/ https://$host:{{KEEPWEBDLSSLPORT}}/;
    }
  }
  upstream ws {
    server localhost:{{WSPORT}};
  }
  server {
    listen *:{{WSSPORT}} ssl default_server;
    server_name ~^(?<request_host>.*)$;
    ssl_certificate {{SSLCERT}};
    ssl_certificate_key {{SSLKEY}};
    location  / {
      proxy_pass http://ws;
      proxy_set_header Upgrade $http_upgrade;
      proxy_set_header Connection "upgrade";
      proxy_set_header Host $request_host:{{WSPORT}};
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

	cfgPath := path.Join(cfg.DataDir, "gateway.consul-template.hcl")
	if err = atomicWriteJSON(cfgPath+".ctmpl", map[string]interface{}{
		"consul": map[string]interface{}{
			"address": fmt.Sprintf("0.0.0.0:%d", cfg.Ports.ConsulHTTP),
		},
		"vault": map[string]string{
			"address": fmt.Sprintf("http://0.0.0.0:%d", cfg.Ports.VaultServer),
			"token":   rootToken,
		}}, 0600); err != nil {
		return err
	}

	tmplPath := path.Join(cfg.DataDir, "gateway.nginx.conf")
	if err = atomicWriteFile(tmplPath+".ctmpl", []byte(ngb.tmpl), 0644); err != nil {
		return err
	}

	return Series{
		&osPackage{
			Debian: "nginx",
		},
		&supervisedService{
			name: ngb.name,
			cmd:  path.Join(cfg.UsrDir, "bin", "consul-template"),
			args: []string{
				"-config=" + cfgPath,
				"-template=" + tmplPath + ".ctmpl:" + tmplPath,
				"-exec",
				"nginx",
			},
			env: map[string]string{
				"VAULT_TOKEN": rootToken,
			},
		},
	}.Boot(ctx)
}
