---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

### PROMETHEUS
prometheus:
  wanted:
    component:
      - node_exporter
  pkg:
    use_upstream_repo: true
    component:
      node_exporter:
        service:
          args:
            collector.textfile.directory: /var/lib/prometheus/node-exporter
