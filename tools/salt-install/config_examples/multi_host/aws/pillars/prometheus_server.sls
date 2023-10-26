---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set controller_nodes = "__CONTROLLER_NODES__".split(',') %}
{%- set enable_balancer = ("__ENABLE_BALANCER__"|to_bool) %}
{%- set data_retention_time = "__PROMETHEUS_DATA_RETENTION_TIME__" %}

### PROMETHEUS
prometheus:
  wanted:
    component:
      - prometheus
      - alertmanager
      - node_exporter
      - blackbox_exporter
  pkg:
    use_upstream_repo: false
    use_upstream_archive: true
    component:
      blackbox_exporter:
        config_file: /etc/prometheus/blackbox_exporter.yml
        config:
          modules:
            http_2xx:
              prober: http
              timeout: 5s
              http:
                valid_http_versions: [HTTP/1.1, HTTP/2]
                valid_status_codes: [200]
                method: GET
                tls_config:
                  insecure_skip_verify: true # Avoid failures on self-signed certs
                fail_if_ssl: false
                fail_if_not_ssl: true
            http_2xx_mngmt_token:
              prober: http
              timeout: 5s
              http:
                valid_http_versions: [HTTP/1.1, HTTP/2]
                valid_status_codes: [200]
                method: GET
                bearer_token: __MANAGEMENT_TOKEN__
                tls_config:
                  insecure_skip_verify: true # Avoid failures on self-signed certs
                fail_if_ssl: false
                fail_if_not_ssl: true
            http_2xx_basic_auth:
              prober: http
              timeout: 5s
              http:
                valid_http_versions: [HTTP/1.1, HTTP/2]
                valid_status_codes: [200]
                method: GET
                basic_auth:
                  username: "__MONITORING_USERNAME__"
                  password: "__MONITORING_PASSWORD__"
                tls_config:
                  insecure_skip_verify: true # Avoid failures on self-signed certs
                fail_if_ssl: false
                fail_if_not_ssl: true
      prometheus:
        service:
           args:
             storage.tsdb.retention.time: {{ data_retention_time }}
        config:
          global:
            scrape_interval: 15s
            evaluation_interval: 15s
          rule_files:
            - rules.yml

          scrape_configs:
            - job_name: prometheus
              # metrics_path defaults to /metrics
              # scheme defaults to http.
              static_configs:
              - targets: ['localhost:9090']
                labels:
                  instance: mon.__CLUSTER__
                  cluster: __CLUSTER__

            - job_name: http_probe
              metrics_path: /probe
              params:
                module: [http_2xx]
              static_configs:
                - targets: ['https://workbench.__DOMAIN__']
                  labels:
                    instance: workbench.__CLUSTER__
                - targets: ['https://workbench2.__DOMAIN__']
                  labels:
                    instance: workbench2.__CLUSTER__
                - targets: ['https://webshell.__DOMAIN__']
                  labels:
                    instance: webshell.__CLUSTER__
              relabel_configs:
                - source_labels: [__address__]
                  target_label: __param_target
                - source_labels: [__param_target]
                  target_label: instance
                - target_label: __address__
                  replacement: 127.0.0.1:9115          # blackbox exporter.

            - job_name: http_probe_mngmt_token
              metrics_path: /probe
              params:
                module: [http_2xx_mngmt_token]
              static_configs:
                - targets: ['https://__DOMAIN__/_health/ping']
                  labels:
                    instance: controller.__CLUSTER__
                - targets: ['https://download.__DOMAIN__/_health/ping']
                  labels:
                    instance: download.__CLUSTER__
                - targets: ['https://ws.__DOMAIN__/_health/ping']
                  labels:
                    instance: ws.__CLUSTER__
              relabel_configs:
                - source_labels: [__address__]
                  target_label: __param_target
                - source_labels: [__param_target]
                  target_label: instance
                - target_label: __address__
                  replacement: 127.0.0.1:9115          # blackbox exporter.

            - job_name: http_probe_basic_auth
              metrics_path: /probe
              params:
                module: [http_2xx_basic_auth]
              static_configs:
                - targets: ['https://grafana.__DOMAIN__']
                  labels:
                    instance: grafana.__CLUSTER__
                - targets: ['https://prometheus.__DOMAIN__']
                  labels:
                    instance: prometheus.__CLUSTER__
              relabel_configs:
                - source_labels: [__address__]
                  target_label: __param_target
                - source_labels: [__param_target]
                  target_label: instance
                - target_label: __address__
                  replacement: 127.0.0.1:9115          # blackbox exporter.

            ## Arvados unique jobs
            - job_name: arvados_ws
              bearer_token: __MANAGEMENT_TOKEN__
              scheme: https
              static_configs:
                - targets: ['ws.__DOMAIN__:443']
                  labels:
                    instance: ws.__CLUSTER__
                    cluster: __CLUSTER__
            - job_name: arvados_controller
              bearer_token: __MANAGEMENT_TOKEN__
              {%- if enable_balancer %}
              scheme: http
              {%- else %}
              scheme: https
              {%- endif %}
              static_configs:
                {%- if enable_balancer %}
                  {%- for controller in controller_nodes %}
                - targets: ['{{ controller }}']
                  labels:
                    instance: {{ controller.split('.')[0] }}.__CLUSTER__
                    cluster: __CLUSTER__
                  {%- endfor %}
                {%- else %}
                - targets: ['__DOMAIN__:443']
                  labels:
                    instance: controller.__CLUSTER__
                    cluster: __CLUSTER__
                {%- endif %}
            - job_name: keep_web
              bearer_token: __MANAGEMENT_TOKEN__
              scheme: https
              static_configs:
                - targets: ['keep.__DOMAIN__:443']
                  labels:
                    instance: keep-web.__CLUSTER__
                    cluster: __CLUSTER__
            - job_name: keep_balance
              bearer_token: __MANAGEMENT_TOKEN__
              static_configs:
                - targets: ['__KEEPBALANCE_INT_IP__:9005']
                  labels:
                    instance: keep-balance.__CLUSTER__
                    cluster: __CLUSTER__
            - job_name: keepstore
              bearer_token: __MANAGEMENT_TOKEN__
              static_configs:
                - targets: ['__KEEPSTORE0_INT_IP__:25107']
                  labels:
                    instance: keep0.__CLUSTER__
                    cluster: __CLUSTER__
            - job_name: arvados_dispatch_cloud
              bearer_token: __MANAGEMENT_TOKEN__
              static_configs:
                - targets: ['__DISPATCHER_INT_IP__:9006']
                  labels:
                    instance: arvados-dispatch-cloud.__CLUSTER__
                    cluster: __CLUSTER__

            {%- if "__DATABASE_INT_IP__" != "" %}
            # Database
            - job_name: postgresql
              static_configs:
                - targets: [
                    '__DATABASE_INT_IP__:9187',
                    '__DATABASE_INT_IP__:3903'
                  ]
                  labels:
                    instance: database.__CLUSTER__
                    cluster: __CLUSTER__
            {%- endif %}

            # Nodes
            {%- set node_list = "__NODELIST__".split(',') %}
            {%- set nodes = [] %}
            {%- for node in node_list %}
              {%- set _ = nodes.append(node.split('.')[0]) %}
            {%- endfor %}
            - job_name: node
              static_configs:
                {% for node in nodes %}
                - targets: [ "{{ node }}.__DOMAIN__:9100" ]
                  labels:
                    instance: "{{ node }}.__CLUSTER__"
                    cluster: __CLUSTER__
                {% endfor %}
