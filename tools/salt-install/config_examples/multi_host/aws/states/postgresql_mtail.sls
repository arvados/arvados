# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

### PACKAGES
monitoring_required_pkgs:
  pkg.installed:
    - name: mtail

### FILES
prometheus_pg_exporter_etc_default:
  file.managed:
    - name: /etc/default/prometheus-postgres-exporter
    - contents: |
        ### This file managed by Salt, do not edit by hand!!
        #
        # For details, check /usr/share/doc/prometheus-postgres-exporter/README.Debian
        DATA_SOURCE_NAME='user=prometheus host=/run/postgresql dbname=postgres'
    - require:
      - pkg: prometheus-package-install-postgres_exporter-installed

mtail_postgresql_conf:
  file.managed:
    - name: /etc/mtail/postgresql.mtail
    - contents: |
        ########################################################################
        # File managed by Salt.
        # Your changes will be overwritten.
        ########################################################################

        # Parser for postgresql's log statement duration

        gauge postgresql_statement_duration_seconds by statement

        /^/ +
        /(?P<timestamp>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} (\w+)) / + # 2019-01-16 16:53:45 GMT
        /LOG: +duration: / +
        /(?P<duration>[0-9\.]+) ms/ + # 153.967 ms
        /(.*?): (?P<statement>.+)/ + # statement: SELECT COUNT(*) FROM (SELECT rolname FROM pg_roles WHERE rolname='arvados') count
        /$/ {
          strptime($timestamp, "2006-01-02 15:04:05 MST") # for tests

          postgresql_statement_duration_seconds[$statement] = $duration / 1000
        }
    - require:
      - pkg: monitoring_required_pkgs

mtail_etc_default:
  file.managed:
    - name: /etc/default/mtail
    - contents: |
        ### This file managed by Salt, do not edit by hand!!
        #
        ENABLED=true
        # List of files to monitor (mandatory).
        LOGS=/var/log/postgresql/postgresql*log
    - require:
      - pkg: monitoring_required_pkgs

### SERVICES
prometheus_pg_exporter_service:
  service.running:
    - name: prometheus-postgres-exporter
    - enable: true
    - require:
      - pkg: prometheus-package-install-postgres_exporter-installed
    - watch:
      - file: /etc/default/prometheus-postgres-exporter

mtail_service:
  service.running:
    - name: mtail
    - enable: true
    - require:
      - pkg: monitoring_required_pkgs
    - watch:
      - file: /etc/mtail/postgresql.mtail
      - file: /etc/default/mtail
