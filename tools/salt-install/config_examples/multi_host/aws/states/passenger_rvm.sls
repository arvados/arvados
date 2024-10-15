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
