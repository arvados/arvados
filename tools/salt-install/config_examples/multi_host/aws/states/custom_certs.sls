# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set orig_cert_dir = salt['pillar.get']('extra_custom_certs_dir', '/srv/salt/certs')  %}
{%- set dest_cert_dir = '/etc/nginx/ssl' %}
{%- set certs = salt['pillar.get']('extra_custom_certs', [])  %}

{% if certs %}
extra_custom_certs_file_directory_certs_dir:
  file.directory:
    - name: /etc/nginx/ssl
    - require:
      - pkg: nginx_install

  {%- for cert in certs %}
    {%- set cert_file = 'arvados-' ~ cert ~ '.pem' %}
    {#- set csr_file = 'arvados-' ~ cert ~ '.csr' #}
    {%- set key_file = 'arvados-' ~ cert ~ '.key' %}
    {% for c in [cert_file, key_file] %}
extra_custom_certs_file_copy_{{ c }}:
  file.copy:
    - name: {{ dest_cert_dir }}/{{ c }}
    - source: {{ orig_cert_dir }}/{{ c }}
    - force: true
    - user: root
    - group: root
    - unless: cmp {{ dest_cert_dir }}/{{ c }} {{ orig_cert_dir }}/{{ c }}
    - require:
      - file: extra_custom_certs_file_directory_certs_dir
    {%- endfor %}
  {%- endfor %}
{%- endif %}
