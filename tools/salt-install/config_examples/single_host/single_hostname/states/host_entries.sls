# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set curr_tpldir = tpldir %}
{%- set tpldir = 'arvados' %}
{%- from "arvados/map.jinja" import arvados with context %}
{%- set tpldir = curr_tpldir %}

# We need the external hostname to resolve to the internal IP for docker. We
# tell docker to resolve via the local dnsmasq, which reads from /etc/hosts by
# default.
arvados_local_access_to_hostname_ext:
  host.present:
    - ip: __IP_INT__
    - names:
      - __HOSTNAME_EXT__

arvados_test_salt_states_examples_single_host_etc_hosts_host_present:
  host.present:
    - ip: 127.0.1.1
    - names:
      - {{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
      # NOTE! This just works for our testing.
      # Won't work if the cluster name != host name
      {%- for entry in [
          'api',
          'collections',
          'controller',
          'download',
          'keep',
          'keepweb',
          'keep0',
          'shell',
          'workbench',
          'workbench2',
          'ws',
        ]
      %}
      - {{ entry }}
      - {{ entry }}.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
      {%- endfor %}
    - require_in:
      - file: nginx_config
      - service: nginx_service
