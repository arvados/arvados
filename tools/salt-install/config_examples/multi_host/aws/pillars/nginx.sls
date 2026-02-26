---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set _workers = ("__CONTROLLER_MAX_WORKERS__" or grains['num_cpus']*2)|int %}
{%- set max_workers = [_workers, 8]|max %}
{%- set max_reqs = ("__CONTROLLER_MAX_QUEUED_REQUESTS__" or 128)|int %}
{%- set max_tunnels = ("__CONTROLLER_MAX_GATEWAY_TUNNELS__" or 1000)|int %}

### NGINX
nginx:
  ### SERVER
  server:
    config:
      worker_processes: {{ max_workers }}

      # Each client request is up to 3 connections (1 with client, 1 proxy to
      # controller, then potentially 1 from controller back to
      # passenger).  Each connection consumes a file descriptor.
      # That's how we get these calculations
      # (we're multiplying by 5 instead to be on the safe side)
      worker_rlimit_nofile: {{ (max_reqs + max_tunnels) * 5 + 1 }}
      events:
        worker_connections: {{ (max_reqs + max_tunnels) * 5 + 1 }}

  ### SITES
  servers:
    managed:
      # Remove default webserver
      default:
        enabled: false
