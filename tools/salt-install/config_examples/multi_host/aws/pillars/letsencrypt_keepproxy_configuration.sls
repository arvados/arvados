---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### LETSENCRYPT
letsencrypt:
  domainsets:
    keep.__CLUSTER__.__DOMAIN__:
      - keep.__CLUSTER__.__DOMAIN__

### NGINX
nginx:
  ### SNIPPETS
  snippets:
    keep.__CLUSTER__.__DOMAIN___letsencrypt_cert.conf:
      - ssl_certificate: /etc/letsencrypt/live/keep.__CLUSTER__.__DOMAIN__/fullchain.pem
      - ssl_certificate_key: /etc/letsencrypt/live/keep.__CLUSTER__.__DOMAIN__/privkey.pem
