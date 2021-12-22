# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# WARNING: This file is only used for testing purposes, and should not be used
# in a production environment

{%- set curr_tpldir = tpldir %}
{%- set tpldir = 'arvados' %}
{%- from "arvados/map.jinja" import arvados with context %}
{%- set tpldir = curr_tpldir %}

{%- set orig_cert_dir = salt['pillar.get']('extra_custom_certs_dir', '/srv/salt/certs')  %}

include:
  - nginx.passenger
  - nginx.config
  - nginx.service

# Debian uses different dirs for certs and keys, but being a Snake Oil example,
# we'll keep it simple here.
{%- set arvados_ca_cert_file = '/etc/ssl/private/arvados-snakeoil-ca.pem' %}
{%- set arvados_ca_key_file = '/etc/ssl/private/arvados-snakeoil-ca.key' %}

{%- if grains.get('os_family') == 'Debian' %}
  {%- set arvados_ca_cert_dest = '/usr/local/share/ca-certificates/arvados-snakeoil-ca.crt' %}
  {%- set update_ca_cert = '/usr/sbin/update-ca-certificates' %}
  {%- set openssl_conf = '/etc/ssl/openssl.cnf' %}

extra_snakeoil_certs_ssl_cert_pkg_installed:
  pkg.installed:
    - name: ssl-cert
    - require_in:
      - sls: postgres

{%- else %}
  {%- set arvados_ca_cert_dest = '/etc/pki/ca-trust/source/anchors/arvados-snakeoil-ca.pem' %}
  {%- set update_ca_cert = '/usr/bin/update-ca-trust' %}
  {%- set openssl_conf = '/etc/pki/tls/openssl.cnf' %}

{%- endif %}

extra_snakeoil_certs_dependencies_pkg_installed:
  pkg.installed:
    - pkgs:
      - openssl
      - ca-certificates

# Remove the RANDFILE parameter in openssl.cnf as it makes openssl fail in Ubuntu 18.04
# Saving and restoring the rng state is not necessary anymore in the openssl 1.1.1
# random generator, cf
#   https://github.com/openssl/openssl/issues/7754
#
extra_snakeoil_certs_file_comment_etc_openssl_conf:
  file.comment:
    - name: /etc/ssl/openssl.cnf
    - regex: ^RANDFILE.*
    - onlyif: grep -q ^RANDFILE /etc/ssl/openssl.cnf
    - require_in:
      - cmd: extra_snakeoil_certs_arvados_snakeoil_ca_cmd_run

extra_snakeoil_certs_arvados_snakeoil_ca_cmd_run:
  # Taken from https://github.com/arvados/arvados/blob/master/tools/arvbox/lib/arvbox/docker/service/certificate/run
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
      - pkg: extra_snakeoil_certs_dependencies_pkg_installed

# Create independent certs for each vhost
{%- for vh in [
  'collections',
  'controller',
  'download',
  'keepproxy',
  'webshell',
  'workbench',
  'workbench2',
  'websocket',
  ]
%}
# We're creating these in a tmp directory, so they're copied to their destination
# with the `custom_certs` state file, as if using custom certificates.
{%- set arvados_cert_file = orig_cert_dir ~ '/arvados-' ~ vh ~ '.pem' %}
{%- set arvados_csr_file = orig_cert_dir ~ '/arvados-' ~ vh ~ '.csr' %}
{%- set arvados_key_file = orig_cert_dir ~ '/arvados-' ~ vh ~ '.key' %}

extra_snakeoil_certs_arvados_snakeoil_cert_{{ vh }}_cmd_run:
  cmd.run:
    - name: |
        cat > /tmp/{{ vh }}.openssl.cnf <<-CNF
        [req]
        default_bits = 2048
        prompt = no
        default_md = sha256
        distinguished_name = dn
        req_extensions = rext
        [rext]
        subjectAltName = @alt_names
        [dn]
        C   = CC
        ST  = Some State
        L   = Some Location
        O   = Arvados Provision Example Single Host / Multiple Hostnames
        OU  = arvados-provision-example-single_host_multiple_hostnames
        CN  = {{ vh }}.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
        emailAddress = admin@{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
        [alt_names]
        {%- for entry in grains.get('ipv4') %}
        IP.{{ loop.index }} = {{ entry }}
        {%- endfor %}
        DNS.1 = {{ vh }}.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
        {%- if vh in [
          'controller',
          'keepproxy',
          'websocket'
          ]
        %}
          {%- if vh == 'controller' %}
        DNS.2 = {{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
          {%- elif vh == 'keepproxy' %}
        DNS.2 = keep.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
          {%- elif vh == 'websocket' %}
        DNS.2 = ws.{{ arvados.cluster.name }}.{{ arvados.cluster.domain }}
          {%- endif %}
        {%- endif %}
        CNF

        # The req
        openssl req \
          -config /tmp/{{ vh }}.openssl.cnf \
          -new \
          -nodes \
          -sha256 \
          -out {{ arvados_csr_file }} \
          -keyout {{ arvados_key_file }} > /tmp/snakeoil_certs.{{ vh }}.output 2>&1 && \
        # The cert
        openssl x509 \
          -req \
          -days 365 \
          -in {{ arvados_csr_file }} \
          -out {{ arvados_cert_file }} \
          -extfile /tmp/{{ vh }}.openssl.cnf \
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
      - pkg: extra_snakeoil_certs_dependencies_pkg_installed
      - cmd: extra_snakeoil_certs_arvados_snakeoil_ca_cmd_run
    - require_in:
      - file: extra_custom_certs_file_copy_arvados-{{ vh }}.pem
      - file: extra_custom_certs_file_copy_arvados-{{ vh }}.key

  {%- if grains.get('os_family') == 'Debian' %}
extra_snakeoil_certs_certs_permissions_{{ vh}}_cmd_run:
  file.managed:
    - name: {{ arvados_key_file }}
    - owner: root
    - group: ssl-cert
    - require:
      - cmd: extra_snakeoil_certs_arvados_snakeoil_cert_{{ vh }}_cmd_run
      - pkg: extra_snakeoil_certs_ssl_cert_pkg_installed
  {%- endif %}
{%- endfor %}
