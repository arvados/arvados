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
      arvados_workbench2_ssl:
        enabled: true
        overwrite: true
        config:
          - server:
            - server_name: __HOSTNAME_EXT__
            - listen:
              - __WORKBENCH2_EXT_SSL_PORT__ http2 ssl
            - index: index.html index.htm
            - location /:
              - root: /var/www/arvados-workbench2/workbench2
              - try_files: '$uri $uri/ /index.html'
              - 'if (-f $document_root/maintenance.html)':
                - return: 503
            - location /config.json:
              - return: {{ "200 '" ~ '{"API_HOST":"__HOSTNAME_EXT__:__CONTROLLER_EXT_SSL_PORT__"}' ~ "'" }}
            - include: 'snippets/arvados-snakeoil.conf'
            - access_log: /var/log/nginx/workbench2.__CLUSTER__.__DOMAIN__.access.log combined
            - error_log: /var/log/nginx/workbench2.__CLUSTER__.__DOMAIN__.error.log
