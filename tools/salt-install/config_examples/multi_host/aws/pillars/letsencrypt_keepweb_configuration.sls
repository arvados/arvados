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
      - '*.collections.__CLUSTER__.__DOMAIN__'
