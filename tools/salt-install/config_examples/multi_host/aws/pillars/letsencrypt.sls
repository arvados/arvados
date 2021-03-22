---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### LETSENCRYPT
letsencrypt:
  use_package: true
  pkgs:
    - certbot: latest
    - python3-certbot-nginx
  config:
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: __INITIAL_USER_EMAIL__
    authenticator: nginx
    webroot-path: /var/www
    agree-tos: true
    keep-until-expiring: true
    expand: true
    max-log-backups: 0
    deploy-hook: systemctl reload nginx

### NGINX
nginx:
  ### SNIPPETS
  snippets:
    ### LETSENCRYPT DEFAULT PATH
    letsencrypt_well_known.conf:
      - location /.well-known:
        - root: /var/www
