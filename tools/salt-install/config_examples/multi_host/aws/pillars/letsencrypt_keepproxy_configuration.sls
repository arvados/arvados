---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### LETSENCRYPT
letsencrypt:
  domainsets:
    keepproxy.__CLUSTER__.__DOMAIN__:
      - keep.__CLUSTER__.__DOMAIN__
