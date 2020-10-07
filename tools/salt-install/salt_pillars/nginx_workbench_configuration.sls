---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: CC-BY-SA-3.0

{% set nginx_log = '/var/log/nginx' %}

### ARVADOS
arvados:
  config:
    group: www-data

### NGINX
nginx:
  ### SERVER
  server:
    config:

      ### STREAMS
      http:
        upstream workbench_upstream:
          - server: '127.0.0.1:9000 fail_timeout=10s'

  ### SITES
  servers:
    managed:
      ### DEFAULT
      arvados_workbench_default:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: workbench.arv.local
            - listen:
              - 80
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_workbench:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: workbench.arv.local
            - listen:
              - 443 http2 ssl
            - index: index.html index.htm
            - location /:
              - proxy_pass: 'http://workbench_upstream'
              - proxy_read_timeout: 300
              - proxy_connect_timeout: 90
              - proxy_redirect: 'off'
              - proxy_set_header: X-Forwarded-Proto https
              - proxy_set_header: 'Host $http_host'
              - proxy_set_header: 'X-Real-IP $remote_addr'
              - proxy_set_header: 'X-Forwarded-For $proxy_add_x_forwarded_for'
            # - include: 'snippets/letsencrypt.conf'
            - include: 'snippets/snakeoil.conf'
            - access_log: {{ nginx_log }}/workbench.arv.local.access.log combined
            - error_log: {{ nginx_log }}/workbench.arv.local.error.log

      arvados_workbench_upstream:
        enabled: true
        overwrite: true
        config:
          - server:
            - listen: '127.0.0.1:9000'
            - server_name: workbench
            - root: /var/www/arvados-workbench/current/public
            - index:  index.html index.htm
            # yamllint disable-line rule:line-length
            - access_log: {{ nginx_log }}/workbench.arv.local-upstream.access.log combined
            - error_log: {{ nginx_log }}/workbench.arv.local-upstream.error.log
