---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# The logrotate formula checks that an associated service is running.
# The default it checks is cron.
# All the distributions Arvados supports (Debian 11+/Ubuntu 20.04+)
# have switched to a systemd timer, so check that instead.
# Refer to logrotate-formula's documentation for details
# https://github.com/salt-formulas/salt-formula-logrotate/blob/master/README.rst

logrotate:
  service: logrotate.timer
