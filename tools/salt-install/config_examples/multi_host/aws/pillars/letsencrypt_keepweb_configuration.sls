---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### LETSENCRYPT
letsencrypt:
  domainsets:
    download.__CLUSTER__.__DOMAIN__:
      - download.__CLUSTER__.__DOMAIN__
    collections.__CLUSTER__.__DOMAIN__:
      - collections.__CLUSTER__.__DOMAIN__
      - *.collections.__CLUSTER__.__DOMAIN__

### NGINX
nginx:
  ### SNIPPETS
  snippets:
    download.__CLUSTER__.__DOMAIN___letsencrypt_cert.conf:
      - ssl_certificate: /etc/letsencrypt/live/download.__CLUSTER__.__DOMAIN__/fullchain.pem
      - ssl_certificate_key: /etc/letsencrypt/live/download.__CLUSTER__.__DOMAIN__/privkey.pem
    collections.__CLUSTER__.__DOMAIN___letsencrypt_cert.conf:
      - ssl_certificate: /etc/letsencrypt/live/collections.__CLUSTER__.__DOMAIN__/fullchain.pem
      - ssl_certificate_key: /etc/letsencrypt/live/collections.__CLUSTER__.__DOMAIN__/privkey.pem
