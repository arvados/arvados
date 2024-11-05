---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set passenger_pkg = 'nginx-mod-http-passenger'
                          if grains.osfinger in ('CentOS Linux-7') else
                        'libnginx-mod-http-passenger' %}
{%- set passenger_mod = '/usr/lib64/nginx/modules/ngx_http_passenger_module.so'
                          if grains.osfinger in ('CentOS Linux-7',) else
                        '/usr/lib/nginx/modules/ngx_http_passenger_module.so' %}
{%- set _workers = ("__CONTROLLER_MAX_WORKERS__" or grains['num_cpus']*2)|int %}
{%- set max_workers = [_workers, 8]|max %}
{%- set max_reqs = ("__CONTROLLER_MAX_QUEUED_REQUESTS__" or 128)|int %}
{%- set max_tunnels = ("__CONTROLLER_MAX_GATEWAY_TUNNELS__" or 1000)|int %}

### NGINX
nginx:
  __NGINX_INSTALL_SOURCE__: true
  lookup:
    passenger_package: {{ passenger_pkg }}
  ### PASSENGER
  passenger:
    passenger_max_pool_size: {{ max_workers }}

    # Make the passenger queue small (twice the concurrency, so
    # there's at most one pending request for each busy worker)
    # because controller reorders requests based on priority, and
    # won't send more than API.MaxConcurrentRailsRequests to passenger
    # (which is max_workers * 2), so things that are moved to the head
    # of the line get processed quickly.
    passenger_max_request_queue_size: {{ max_workers * 2 + 1 }}

  ### SERVER
  server:
    config:
      # Needed for RVM, harmless otherwise. Cf. https://dev.arvados.org/issues/19015
      env: GEM_HOME
      # As we now differentiate where passenger is required or not, we need to
      # load this module conditionally, so we add this conditional just to use
      # the same pillar file
      {% if "install_from_phusionpassenger" == "__NGINX_INSTALL_SOURCE__" %}
      # This is required to get the passenger module loaded
      # In Debian it can be done with this
      # include: 'modules-enabled/*.conf'
      load_module: {{ passenger_mod }}
      {% endif %}
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
