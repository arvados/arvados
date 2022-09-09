---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### LOGROTATE
logrotate:
  jobs:
    arvados-api:
      path:
        - /var/www/arvados-api/shared/log/*.log
      config:
        - daily
        - missingok
        - rotate 365
        - compress
        - nodelaycompress
        - copytruncate
        - sharedscripts
        - postrotate
        - '  [ -s /run/nginx.pid ] && kill -USR1 `cat /run/nginx.pid`'
        - endscript