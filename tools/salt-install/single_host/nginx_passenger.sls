---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

### NGINX
nginx:
  install_from_phusionpassenger: true
  lookup:
    passenger_package: libnginx-mod-http-passenger
    passenger_config_file: /etc/nginx/conf.d/mod-http-passenger.conf

  ### SERVER
  server:
    config:
      include: 'modules-enabled/*.conf'
      worker_processes: 4

  ### SITES
  servers:
    managed:
      # Remove default webserver
      default:
        enabled: false
