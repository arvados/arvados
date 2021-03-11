---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### ARVADOS
arvados:
  config:
    group: www-data

### NGINX
nginx:
  ### SITES
  servers:
    managed:
      ### DEFAULT
      arvados_workbench2_default:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: workbench2.__DOMAIN__
            - listen:
              - 80
            - location /.well-known:
              - root: /var/www
            - location /:
              - return: '301 https://$host$request_uri'

      arvados_workbench2_ssl:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: workbench2.__DOMAIN__
            - listen:
              - __CONTROLLER_EXT_SSL_PORT__ http2 ssl
            - index: index.html index.htm
            - location /:
              - root: /var/www/arvados-workbench2/workbench2
              - try_files: '$uri $uri/ /index.html'
              - 'if (-f $document_root/maintenance.html)':
                - return: 503
            - location /config.json:
              - return: {{ "200 '" ~ '{"API_HOST":"__CLUSTER__.__DOMAIN__:__CONTROLLER_EXT_SSL_PORT__"}' ~ "'" }}
            - include: snippets/ssl_hardening_default.conf
            - ssl_certificate: /etc/letsencrypt/live/workbench2.__CLUSTER__.__DOMAIN__/fullchain.pem
            - ssl_certificate_key: /etc/letsencrypt/live/workbench2.__CLUSTER__.__DOMAIN__/privkey.pem
            - access_log: /var/log/nginx/workbench2.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/workbench2.__DOMAIN__.error.log
