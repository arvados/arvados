---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

prometheus_pg_exporter:
  enabled: true

### PROMETHEUS
prometheus:
  wanted:
    component:
      - postgres_exporter
      - node_exporter
