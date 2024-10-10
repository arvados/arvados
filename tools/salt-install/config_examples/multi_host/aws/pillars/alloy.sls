# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set prom_user = "__MONITORING_USERNAME__" %}
{%- set prom_pass = "__MONITORING_PASSWORD__" %}
{%- set prom_host = "prometheus.__DOMAIN__" %}
{%- set loki_user = "__MONITORING_USERNAME__" %}
{%- set loki_pass = "__MONITORING_PASSWORD__" %}
{%- set loki_host = "loki.__DOMAIN__" %}

alloy:
  enabled: True
  package: "alloy"
  service: "alloy"
  config_path: "/etc/alloy/config.alloy"
  config_contents: |
    ////////////////////////////////////////////////////////////////////////
    // File managed by Salt. Your changes will be overwritten.
    ////////////////////////////////////////////////////////////////////////
    logging {
      level = "warn"
    }

    prometheus.exporter.unix "default" {
      include_exporter_metrics = true
      disable_collectors       = ["mdadm"]
    }

    prometheus.scrape "default" {
      targets = concat(
        prometheus.exporter.unix.default.targets,
        [{
          // Self-collect metrics
          job         = "alloy",
          __address__ = "127.0.0.1:12345",
        }],
      )

      forward_to = [
        prometheus.remote_write.metrics_service.receiver,
      ]
    }

    prometheus.remote_write "metrics_service" {
      endpoint {
        url = "https://{{ prom_host }}/api/v1/write"

        basic_auth {
          username = "{{ prom_user }}"
          password = "{{ prom_pass }}"
        }
      }
    }

    local.file_match "file_logs" {
      path_targets = [
        {"__path__" = "/var/log/nginx/*.log"},
        {"__path__" = "/var/www/arvados-api/shared/log/production.log"},
      ]
      sync_period = "5s"
    }

    loki.source.file "log_scrape" {
      targets    = local.file_match.file_logs.targets
      forward_to = [loki.write.grafana_loki.receiver]
      tail_from_end = true
    }

    loki.source.journal "journal_logs" {
      relabel_rules = loki.relabel.journal.rules
      forward_to = [loki.write.grafana_loki.receiver]
      labels = {component = "loki.source.journal"}
    }

    loki.relabel "journal" {
      forward_to = []

      rule {
        source_labels = ["__journal__systemd_unit"]
        target_label  = "systemd_unit"
      }
      rule {
        source_labels = ["__journal__hostname"]
        target_label = "systemd_hostname"
      }
      rule {
        source_labels = ["__journal__transport"]
        target_label = "systemd_transport"
      }
    }

    loki.write "grafana_loki" {
      endpoint {
        url = "https://{{ loki_host }}/loki/api/v1/push"

        basic_auth {
          username = "{{ loki_user }}"
          password = "{{ loki_pass }}"
        }
      }
    }
