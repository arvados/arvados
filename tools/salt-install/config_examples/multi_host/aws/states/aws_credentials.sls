# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set aws_credentials = pillar.get('aws_credentials', {}) %}

{%- if aws_credentials %}
extra_extra_aws_credentials_root_aws_config_file_managed:
  file.managed:
    - name: /root/.aws/config
    - makedirs: true
    - user: root
    - group: root
    - mode: '0600'
    - replace: false
    - contents: |
        [default]
        region= {{ aws_credentials.region }}

extra_extra_aws_credentials_root_aws_credentials_file_managed:
  file.managed:
    - name: /root/.aws/credentials
    - makedirs: true
    - user: root
    - group: root
    - mode: '0600'
    - replace: false
    - contents: |
        [default]
        aws_access_key_id = {{ aws_credentials.access_key_id }}
        aws_secret_access_key = {{ aws_credentials.secret_access_key }}
{%- endif %}
