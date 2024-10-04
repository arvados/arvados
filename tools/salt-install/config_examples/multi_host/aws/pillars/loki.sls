# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

{%- set aws_key_id = "__LOKI_AWS_S3_ACCESS_KEY_ID__" %}
{%- set aws_secret = "__LOKI_AWS_S3_SECRET_ACCESS_KEY__" %}
{%- set aws_region = "__LOKI_AWS_REGION__" %}
{%- set aws_s3_bucket = "__LOKI_AWS_S3_BUCKET__" %}
{%- set log_retention = "__LOKI_LOG_RETENTION_TIME__" %}
{%- set data_path = "/var/lib/loki" %}

loki:
  enabled: True
  package: "loki"
  service: "loki"
  config_path: "/etc/loki/config.yml"
  data_path: {{ data_path }}
  config_contents: |
    ########################################################################
    # File managed by Salt. Your changes will be overwritten.
    ########################################################################
    server:
      http_listen_port: 3100
      grpc_listen_port: 9096

    common:
      instance_addr: 127.0.0.1
      path_prefix: {{ data_path }}
      storage:
        filesystem:
          chunks_directory: {{ data_path }}/chunks
          rules_directory: {{ data_path }}/rules
      replication_factor: 1
      ring:
        kvstore:
          store: inmemory

    query_range:
      results_cache:
        cache:
          embedded_cache:
            enabled: true
            max_size_mb: 100

    storage_config:
      tsdb_shipper:
        active_index_directory: {{ data_path }}/index
        cache_location: {{ data_path }}/index_cache
        cache_ttl: 24h
      aws:
        s3: s3://{{ aws_key_id }}:{{ aws_secret }}@{{ aws_region }}
        bucketnames: {{ aws_s3_bucket }}

    schema_config:
      configs:
        - from: 2024-01-01
          store: tsdb
          object_store: aws
          schema: v13
          index:
            prefix: index_
            period: 24h

    limits_config:
      retention_period: {{ log_retention }}

    compactor:
      working_directory: {{ data_path }}/retention
      delete_request_store: aws
      retention_enabled: true
      compaction_interval: 10m
      retention_delete_delay: 2h
      retention_delete_worker_count: 100

    frontend:
      encoding: protobuf

    analytics:
      reporting_enabled: false
