---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### NGINX
nginx:
  ### SERVER
  server:
    config:

      ### STREAMS
      http:
        upstream webshell_upstream:
          - server: 'localhost:4200 fail_timeout=10s'

  ### SITES
  servers:
    managed:
      arvados_webshell_default:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: webshell.__DOMAIN__
            - listen:
              - 80
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_webshell_ssl:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: webshell.__DOMAIN__
            - listen:
              - __CONTROLLER_EXT_SSL_PORT__ http2 ssl
            - index: index.html index.htm
            - location /shell.__DOMAIN__:
              - proxy_pass: 'http://webshell_upstream'
              - proxy_read_timeout: 90
              - proxy_connect_timeout: 90
              - proxy_set_header: 'Host $http_host'
              - proxy_set_header: 'X-Real-IP $remote_addr'
              - proxy_set_header: X-Forwarded-Proto https
              - proxy_set_header: 'X-Forwarded-For $proxy_add_x_forwarded_for'
              - proxy_ssl_session_reuse: 'off'

              - "if ($request_method = 'OPTIONS')":
                - add_header: "'Access-Control-Allow-Origin' '*'"
                - add_header: "'Access-Control-Allow-Methods' 'GET, POST, OPTIONS'"
                - add_header: "'Access-Control-Allow-Headers' 'DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type'"
                - add_header: "'Access-Control-Max-Age' 1728000"
                - add_header: "'Content-Type' 'text/plain charset=UTF-8'"
                - add_header: "'Content-Length' 0"
                - return: 204

              - "if ($request_method = 'POST')":
                - add_header: "'Access-Control-Allow-Origin' '*'"
                - add_header: "'Access-Control-Allow-Methods' 'GET, POST, OPTIONS'"
                - add_header: "'Access-Control-Allow-Headers' 'DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type'"

              - "if ($request_method = 'GET')":
                - add_header: "'Access-Control-Allow-Origin' '*'"
                - add_header: "'Access-Control-Allow-Methods' 'GET, POST, OPTIONS'"
                - add_header: "'Access-Control-Allow-Headers' 'DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type'"

            - ssl_certificate: /etc/ssl/certs/wildcard.__DOMAIN__.crt
            - ssl_certificate_key: /etc/ssl/private/wildcard.__DOMAIN__.key
            - access_log: /var/log/nginx/webshell.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/webshell.__DOMAIN__.error.log

