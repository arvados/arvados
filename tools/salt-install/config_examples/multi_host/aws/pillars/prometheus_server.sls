---
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

{%- set controller_nodes = "__CONTROLLER_NODES__".split(',') %}
{%- set enable_balancer = ("__ENABLE_BALANCER__"|to_bool) %}

### PROMETHEUS
prometheus:
  wanted:
    component:
      - prometheus
      - alertmanager
      - node_exporter
  pkg:
    use_upstream_repo: true
    component:
      prometheus:
        config:
          global:
            scrape_interval: 15s
            evaluation_interval: 15s
	  storage:
	    tsdb:
	      retention:
	        time: 90d
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
