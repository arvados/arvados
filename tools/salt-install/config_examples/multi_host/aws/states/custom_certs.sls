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
    - user: root
    - group: root
    - dir_mode: 0750
    - file_mode: 0640
    - require:
      - pkg: nginx_install
    - recurse:
      - user
      - group
      - mode

  {%- for cert in certs %}
    {%- set cert_file = 'arvados-' ~ cert ~ '.pem' %}
    {%- set key_file = 'arvados-' ~ cert ~ '.key' %}
extra_custom_certs_{{ cert }}_cert_file_copy:
  file.copy:
    - name: {{ dest_cert_dir }}/{{ cert_file }}
    - source: {{ orig_cert_dir }}/{{ cert_file }}
    - force: true
    - user: root
    - group: root
    - mode: 0640
    - unless: cmp {{ dest_cert_dir }}/{{ cert_file }} {{ orig_cert_dir }}/{{ cert_file }}
    - require:
      - file: extra_custom_certs_file_directory_certs_dir

extra_custom_certs_{{ cert }}_key_file_copy:
  file.copy:
    - name: {{ dest_cert_dir }}/{{ key_file }}
    - source: {{ orig_cert_dir }}/{{ key_file }}
    - force: true
    - user: root
    - group: root
    - mode: 0640
    - unless: cmp {{ dest_cert_dir }}/{{ key_file }} {{ orig_cert_dir }}/{{ key_file }}
    - require:
      - file: extra_custom_certs_file_directory_certs_dir

extra_nginx_service_reload_on_{{ cert }}_certs_changes:
  cmd.run:
    - name: systemctl reload nginx
    - require:
      - file: extra_custom_certs_{{ cert }}_cert_file_copy
      - file: extra_custom_certs_{{ cert }}_key_file_copy
    - onchanges:
      - file: extra_custom_certs_{{ cert }}_cert_file_copy
      - file: extra_custom_certs_{{ cert }}_key_file_copy
    - onlyif:
      - test $(openssl rsa -modulus -noout -in {{ dest_cert_dir }}/{{ key_file }}) == $(openssl x509 -modulus -noout -in {{ dest_cert_dir }}/{{ cert_file }})
  {%- endfor %}
{%- endif %}
