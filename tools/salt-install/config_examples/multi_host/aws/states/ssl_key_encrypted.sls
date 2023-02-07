# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set ssl_key_encrypted = pillar.get('ssl_key_encrypted', {'enabled': False}) %}

{%- if ssl_key_encrypted.enabled %}

extra_ssl_key_encrypted_password_fifo_file:
  file.mknod:
    - ntype: p
    - name: {{ ssl_key_encrypted.ssl_password_file }}
    - user: root
    - group: root
    - mode: '0600'

extra_ssl_key_encrypted_required_pkgs:
  pkg.installed:
    - name: jq

extra_ssl_key_encrypted_password_retrieval_script:
  file.managed:
    - name: {{ ssl_key_encrypted.ssl_password_connector_script }}
    - user: root
    - group: root
    - mode: '0750'
    - require:
      - pkg: extra_ssl_key_encrypted_required_pkgs
      - file: extra_ssl_key_encrypted_password_fifo_file
    - contents: |
        #!/bin/bash

        while [ true ]; do
          # AWS_SHARED_CREDENTIALS_FILE is set to an non-existant path to avoid awscli
          # loading invalid credentials on nodes who use ~/.aws/credentials for other
          # purposes (e.g.: the dispatcher credentials)
          # Access to the secrets manager is given by using an instance profile.
          AWS_SHARED_CREDENTIALS_FILE=~/nonexistant aws secretsmanager get-secret-value --secret-id {{ ssl_key_encrypted.aws_secret_name }} --region us-east-1 | jq -r .SecretString > {{ ssl_key_encrypted.ssl_password_file }}
          sleep 1
        done

extra_ssl_key_encrypted_password_retrieval_service_unit:
  file.managed:
    - name: /etc/systemd/system/password_secret_connector.service
    - user: root
    - group: root
    - mode: '0644'
    - require:
      - file: extra_ssl_key_encrypted_password_retrieval_script
    - contents: |
        [Unit]
        Description=Arvados SSL private key password retrieval service
        After=network.target
        AssertPathExists={{ ssl_key_encrypted.ssl_password_file }}
        [Service]
        ExecStart=/bin/bash {{ ssl_key_encrypted.ssl_password_connector_script }}
        [Install]
        WantedBy=multi-user.target

extra_ssl_key_encrypted_password_retrieval_service:
  service.running:
    - name: password_secret_connector
    - enable: true
    - require:
      - file: extra_ssl_key_encrypted_password_retrieval_service_unit
    - watch:
      - file: extra_ssl_key_encrypted_password_retrieval_service_unit
      - file: extra_ssl_key_encrypted_password_retrieval_script

{%- endif %}