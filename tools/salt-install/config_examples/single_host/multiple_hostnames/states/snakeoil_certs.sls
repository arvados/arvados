# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set curr_tpldir = tpldir %}
{%- set tpldir = 'arvados' %}
{%- from "arvados/map.jinja" import arvados with context %}
{%- set tpldir = curr_tpldir %}

{%- set arvados_ca_cert_file = '/etc/ssl/certs/arvados-snakeoil-ca.pem' %}
{%- set arvados_ca_key_file = '/etc/ssl/private/arvados-snakeoil-ca.key' %}
{%- set arvados_cert_file = '/etc/ssl/certs/arvados-snakeoil-cert.pem' %}
{%- set arvados_csr_file = '/etc/ssl/private/arvados-snakeoil-cert.csr' %}
{%- set arvados_key_file = '/etc/ssl/private/arvados-snakeoil-cert.key' %}

{%- if grains.get('os_family') == 'Debian' %}
  {%- set arvados_ca_cert_dest = '/usr/local/share/ca-certificates/arvados-snakeoil-ca.crt' %}
  {%- set update_ca_cert = '/usr/sbin/update-ca-certificates' %}
  {%- set openssl_conf = '/etc/ssl/openssl.cnf' %}
{%- else %}
  {%- set arvados_ca_cert_dest = '/etc/pki/ca-trust/source/anchors/arvados-snakeoil-ca.pem' %}
  {%- set update_ca_cert = '/usr/bin/update-ca-trust' %}
  {%- set openssl_conf = '/etc/pki/tls/openssl.cnf' %}
{%- endif %}

arvados_test_salt_states_examples_single_host_snakeoil_certs_dependencies_pkg_installed:
  pkg.installed:
    - pkgs:
      - openssl
      - ca-certificates

arvados_test_salt_states_examples_single_host_snakeoil_certs_arvados_snake_oil_ca_cmd_run:
  # Taken from https://github.com/arvados/arvados/blob/main/tools/arvbox/lib/arvbox/docker/service/certificate/run
  cmd.run:
    - name: |
        # These dirs are not to CentOS-ish, but this is a helper script
        # and they should be enough
        mkdir -p /etc/ssl/certs/ /etc/ssl/private/ && \
        openssl req \
          -new \
          -nodes \
          -sha256 \
          -x509 \
          -subj "/C=CC/ST=Some State/O=Arvados Formula/OU=arvados-formula/CN=snakeoil-ca-{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}" \
          -extensions x509_ext \
          -config <(cat {{ openssl_conf }} \
                  <(printf "\n[x509_ext]\nbasicConstraints=critical,CA:true,pathlen:0\nkeyUsage=critical,keyCertSign,cRLSign")) \
          -out {{ arvados_ca_cert_file }} \
          -keyout {{ arvados_ca_key_file }} \
          -days 365 && \
        cp {{ arvados_ca_cert_file }} {{ arvados_ca_cert_dest }} && \
        {{ update_ca_cert }}
    - unless:
      - test -f {{ arvados_ca_cert_file }}
      - openssl verify -CAfile {{ arvados_ca_cert_file }} {{ arvados_ca_cert_file }}
    - require:
      - pkg: arvados_test_salt_states_examples_single_host_snakeoil_certs_dependencies_pkg_installed

arvados_test_salt_states_examples_single_host_snakeoil_certs_arvados_snake_oil_cert_cmd_run:
  cmd.run:
    - name: |
        cat > /tmp/openssl.cnf <<-CNF
        [req]
        default_bits = 2048
        prompt = no
        default_md = sha256
        req_extensions = rext
        distinguished_name = dn
        [dn]
        C   = CC
        ST  = Some State
        L   = Some Location
        O   = Arvados Formula
        OU  = arvados-formula
        CN  = {{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
        emailAddress = admin@{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
        [rext]
        subjectAltName = @alt_names
        [alt_names]
        {%- for entry in grains.get('ipv4') %}
        IP.{{ loop.index }} = {{ entry }}
        {%- endfor %}
        {%- for entry in [
            'keep',
            'collections',
            'download',
            'ws',
            'workbench',
            'workbench2',
          ]
        %}
        DNS.{{ loop.index }} = {{ entry }}.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
        {%- endfor %}
        DNS.7 = {{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
        CNF

        # The req
        openssl req \
          -config /tmp/openssl.cnf \
          -new \
          -nodes \
          -sha256 \
          -out {{ arvados_csr_file }} \
          -keyout {{ arvados_key_file }} > /tmp/snake_oil_certs.output 2>&1 && \
        # The cert
        openssl x509 \
          -req \
          -days 365 \
          -in {{ arvados_csr_file }} \
          -out {{ arvados_cert_file }} \
          -extfile /tmp/openssl.cnf \
          -extensions rext \
          -CA {{ arvados_ca_cert_file }} \
          -CAkey {{ arvados_ca_key_file }} \
          -set_serial $(date +%s) && \
        chmod 0644 {{ arvados_cert_file }} && \
        chmod 0640 {{ arvados_key_file }}
    - unless:
      - test -f {{ arvados_key_file }}
      - openssl verify -CAfile {{ arvados_ca_cert_file }} {{ arvados_cert_file }}
    - require:
      - pkg: arvados_test_salt_states_examples_single_host_snakeoil_certs_dependencies_pkg_installed
      - cmd: arvados_test_salt_states_examples_single_host_snakeoil_certs_arvados_snake_oil_ca_cmd_run

{%- if grains.get('os_family') == 'Debian' %}
arvados_test_salt_states_examples_single_host_snakeoil_certs_ssl_cert_pkg_installed:
  pkg.installed:
    - name: ssl-cert
    - require_in:
      - sls: postgres

arvados_test_salt_states_examples_single_host_snakeoil_certs_certs_permissions_cmd_run:
  cmd.run:
    - name: |
        chown root:ssl-cert {{ arvados_key_file }}
    - require:
      - cmd: arvados_test_salt_states_examples_single_host_snakeoil_certs_arvados_snake_oil_cert_cmd_run
      - pkg: arvados_test_salt_states_examples_single_host_snakeoil_certs_ssl_cert_pkg_installed
{%- endif %}

arvados_test_salt_states_examples_single_host_snakeoil_certs_nginx_snakeoil_file_managed:
  file.managed:
    - name: /etc/nginx/snippets/arvados-snakeoil.conf
    - contents: |
        ssl_certificate {{ arvados_cert_file }};
        ssl_certificate_key {{ arvados_key_file }};
    - watch_in:
      - service: nginx_service
    - require:
      - pkg: passenger_install
      - cmd: arvados_test_salt_states_examples_single_host_snakeoil_certs_certs_permissions_cmd_run
    - require_in:
      - file: nginx_config
      - service: nginx_service
    - watch_in:
      - service: nginx_service


