# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set curr_tpldir = tpldir %}
{%- set tpldir = 'arvados' %}
{%- from "arvados/map.jinja" import arvados with context %}
{%- set tpldir = curr_tpldir %}

extra_shell_sudo_passwordless_sudo_pkg_installed:
  pkg.installed:
    - name: sudo

extra_shell_sudo_passwordless_config_file_managed:
  file.managed:
    - name: /etc/sudoers.d/arvados_passwordless
    - makedirs: true
    - user: root
    - group: root
    - mode: '0440'
    - replace: false
    - contents: |
        # This file managed by Salt, do not edit by hand!!
        # Allow members of group sudo to execute any command without password
        %sudo ALL=(ALL:ALL) NOPASSWD:ALL
    - require:
      - pkg: extra_shell_sudo_passwordless_sudo_pkg_installed
