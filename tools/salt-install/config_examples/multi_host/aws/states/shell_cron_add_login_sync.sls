# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# This state tries to query the controller using the parameters set in
# the `arvados.cluster.resources.virtual_machines` pillar, to get the
# ARVADOS_VIRTUAL_MACHINE_UUID for the host and configure the arvados login-sync cron
# as described in https://doc.arvados.org/main/install/install-shell-server.html

{%- set curr_tpldir = tpldir %}
{%- set tpldir = 'arvados' %}
{%- set sls_config_file = 'arvados.config.file' %}
{#  from "arvados/map.jinja" import arvados with context #}
{%- from "arvados/map.jinja" import arvados with context %}
{%- from "arvados/libtofs.jinja" import files_switch with context %}
{%- set tpldir = curr_tpldir %}

{%- set virtual_machines = arvados.cluster.resources.virtual_machines | default({}) %}
{%- set api_token = arvados.cluster.tokens.system_root | yaml_encode %}
{%- set api_host = arvados.cluster.Services.Controller.ExternalURL | regex_replace('^http(s?)://', '', ignorecase=true) %}

include:
  - arvados

extra_shell_cron_add_login_sync_add_jq_pkg_installed:
  pkg.installed:
    - name: jq

{%- for vm, vm_params in virtual_machines.items() %}
  {%- set vm_name = vm_params.name | default(vm) %}

  # Check if any of the specified virtual_machines parameters corresponds to this instance
  # It should be an error if we get more than one occurrence
  {%- if vm_name in [grains['id'], grains['host'], grains['fqdn'], grains['nodename']] or
         vm_params.backend in [grains['id'], grains['host'], grains['fqdn'], grains['nodename']] +
                               grains['ipv4'] + grains['ipv6'] %}

    # We need to query the VM UUID
    {%- set cmd_query_vm_uuid = 'arv --short virtual_machine list' ~
                                ' --filters \'[["hostname", "=", "' ~ vm_name ~ '"]]\''
    %}

extra_shell_cron_add_login_sync_add_{{ vm }}_get_vm_uuid_cmd_run:
  cmd.run:
    - env:
      - ARVADOS_API_TOKEN: {{ api_token }}
      - ARVADOS_API_HOST: {{ api_host }}
      - ARVADOS_API_HOST_INSECURE: {{ arvados.cluster.tls.insecure | default(false) }}
    - name: {{ cmd_query_vm_uuid }} | head -1 | tee /tmp/vm_uuid_{{ vm }}
    - unless:
      - /bin/grep -qE "[a-z0-9]{5}-2x53u-[a-z0-9]{15}" /tmp/vm_uuid_{{ vm }}

extra_shell_cron_add_login_sync_add_{{ vm }}_arvados_api_host_cron_env_present:
  cron.env_present:
    - name: ARVADOS_API_HOST
    - value: {{ api_host }}
    - onlyif:
      - /bin/grep -qE "[a-z0-9]{5}-2x53u-[a-z0-9]{15}" /tmp/vm_uuid_{{ vm }}

extra_shell_cron_add_login_sync_add_{{ vm }}_arvados_api_token_cron_env_present:
  cron.env_present:
    - name: ARVADOS_API_TOKEN
    - value: {{ api_token }}
    - onlyif:
      - /bin/grep -qE "[a-z0-9]{5}-2x53u-[a-z0-9]{15}" /tmp/vm_uuid_{{ vm }}

extra_shell_cron_add_login_sync_add_{{ vm }}_arvados_api_host_insecure_cron_env_present:
  cron.env_present:
    - name: ARVADOS_API_HOST_INSECURE
    - value: {{ arvados.cluster.tls.insecure | default(false) }}
    - onlyif:
      - /bin/grep -qE "[a-z0-9]{5}-2x53u-[a-z0-9]{15}" /tmp/vm_uuid_{{ vm }}

extra_shell_cron_add_login_sync_add_{{ vm }}_arvados_virtual_machine_uuid_cron_env_present:
  cron.env_present:
    - name: ARVADOS_VIRTUAL_MACHINE_UUID
    - value: __slot__:salt:cmd.run("cat /tmp/vm_uuid_{{ vm }}")
    - onlyif:
      - /bin/grep -qE "[a-z0-9]{5}-2x53u-[a-z0-9]{15}" /tmp/vm_uuid_{{ vm }}

extra_shell_cron_add_login_sync_add_{{ vm }}_arvados_login_sync_cron_present:
  cron.present:
    - name: arvados-login-sync
    - minute: '*/2'
    - onlyif:
      - /bin/grep -qE "[a-z0-9]{5}-2x53u-[a-z0-9]{15}" /tmp/vm_uuid_{{ vm }}

  {%- endif %}
{%- endfor %}
