---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Refer to logrotate-formula's documentation for information about customization
# https://github.com/salt-formulas/salt-formula-logrotate/blob/master/README.rst

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
        - '  systemctl try-reload-or-restart arvados-railsapi.service'
        - endscript
