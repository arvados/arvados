# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- if grains.os_family in ('RedHat',) %}
  {%- set group = 'nginx' %}
{%- else %}
  {%- set group = 'www-data' %}
{%- endif %}

# Make sure that /var/www/.passenger exists with the proper ownership
# so that passenger can build passenger_native_support.so
extra_var_www_passenger:
  file.directory:
    - name: /var/www/.passenger
    - user: {{ group }}
    - group: {{ group }}
    - mode: '0755'
    - makedirs: True

{%- if grains.osfinger in ('CentOS Linux-7', 'Ubuntu-18.04', 'Debian-10') %}
# Work around passenger issue when RVM is in use, cf
# https://dev.arvados.org/issues/19015
extra_nginx_set_gem_home:
  file.managed:
    - name: /etc/systemd/system/nginx.service.d/override.conf
    - mode: '0644'
    - user: root
    - group: root
    - makedirs: True
    - replace: False
    - contents: |
        [Service]
        ExecStart=
        ExecStart=/bin/bash -a -c "GEM_HOME=`/usr/local/rvm/bin/rvm-exec default env |grep GEM_HOME=|cut -f2 -d=` && /usr/sbin/nginx -g 'daemon on; master_process on;'"
  cmd.run:
    - name: systemctl daemon-reload
    - require:
      - file: extra_nginx_set_gem_home
      - file: extra_var_www_passenger
    - onchanges:
      - file: extra_nginx_set_gem_home
{%- endif -%}
