---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### LETSENCRYPT
letsencrypt:
  domainsets:
    download.__DOMAIN__:
      - download.__DOMAIN__
    collections.__DOMAIN__:
      - collections.__DOMAIN__
      - '*.collections.__DOMAIN__'
