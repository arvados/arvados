# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set curr_tpldir = tpldir %}
{%- set tpldir = 'arvados' %}
{%- from "arvados/map.jinja" import arvados with context %}
{%- set tpldir = curr_tpldir %}

#CRUDE, but functional
extra_extra_hosts_entries_etc_hosts_database_host_present:
  host.present:
    - ip: __DATABASE_INT_IP__
    - names:
      - db.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
      - database.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}

extra_extra_hosts_entries_etc_hosts_api_host_present:
  host.present:
    - ip: __CONTROLLER_INT_IP__
    - names:
      - {{ arvados.cluster.name }}.{{ arvados.cluster.domain }}

extra_extra_hosts_entries_etc_hosts_websocket_host_present:
  host.present:
    - ip: __CONTROLLER_INT_IP__
    - names:
      - ws.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}

extra_extra_hosts_entries_etc_hosts_workbench_host_present:
  host.present:
    - ip: __WORKBENCH1_INT_IP__
    - names:
      - workbench.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}

extra_extra_hosts_entries_etc_hosts_workbench2_host_present:
  host.present:
    - ip: __WORKBENCH1_INT_IP__
    - names:
      - workbench2.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}

extra_extra_hosts_entries_etc_hosts_keepproxy_host_present:
  host.present:
    - ip: __KEEP_INT_IP__
    - names:
      - keep.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}

extra_extra_hosts_entries_etc_hosts_keepweb_host_present:
  host.present:
    - ip: __KEEP_INT_IP__
    - names:
      - download.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
      - collections.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}

extra_extra_hosts_entries_etc_hosts_shell_host_present:
  host.present:
    - ip: __WEBSHELL_INT_IP__
    - names:
      - shell.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}

extra_extra_hosts_entries_etc_hosts_keep0_host_present:
  host.present:
    - ip: __KEEPSTORE0_INT_IP__
    - names:
      - keep0.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}

extra_extra_hosts_entries_etc_hosts_keep1_host_present:
  host.present:
    - ip: __KEEPSTORE1_INT_IP__
    - names:
      - keep1.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
