# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set ssl_key_encrypted = pillar.get('ssl_key_encrypted', {'enabled': False}) %}

{%- if ssl_key_encrypted.enabled %}

extra_ssl_key_encrypted_required_pkgs:
  pkg.installed:
    - name: jq

extra_ssl_key_encrypted_password_retrieval_script:
  file.managed:
    - name: {{ ssl_key_encrypted.privkey_password_script }}
    - user: root
    - group: root
    - mode: '0750'
    - require:
      - pkg: extra_ssl_key_encrypted_required_pkgs
    - contents: |
        #!/bin/bash

        # RUNTIME_DIRECTORY is provided by systemd.
        # NOTE: We assume systemd's set up in a way that there's just one
        # runtime dir for this particular unit, otherwise this variable could
        # contain multiple paths separated by a colon.
        PASSWORD_FILE="${RUNTIME_DIRECTORY}/{{ ssl_key_encrypted.privkey_password_filename }}"

        while [ true ]; do
          # AWS_SHARED_CREDENTIALS_FILE is set to /dev/null to avoid AWS's CLI
          # loading invalid credentials on nodes who use ~/.aws/credentials for other
          # purposes (e.g.: the dispatcher credentials)
          # Access to the secrets manager is given by using an instance profile.
          AWS_SHARED_CREDENTIALS_FILE=/dev/null aws secretsmanager get-secret-value --secret-id '{{ ssl_key_encrypted.aws_secret_name }}' --region '{{ ssl_key_encrypted.aws_region }}' | jq -r .SecretString > "${PASSWORD_FILE}"
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
        [Service]
        # WARNING: the script below assumes that RuntimeDirectory only holds one
        # path value, won't work with multiple paths.
        RuntimeDirectory=arvados
        ExecStartPre=/usr/bin/mkfifo --mode=0600 {{ ('%t/arvados/' ~ ssl_key_encrypted.privkey_password_filename) | yaml_dquote }}
        ExecStart=/bin/bash {{ ssl_key_encrypted.privkey_password_script | yaml_dquote }}
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