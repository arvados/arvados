#!/usr/bin/env cwl-runner
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

cwlVersion: v1.2
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"

doc: |
  This workflow reports on the data and workflows in an Arvados cluster to
  help administrators understand growth and costs. It is entirely
  self-contained: you can run this workflow with Workbench or
  `arvados-cwl-runner` to generate a report.

inputs:
  reporting_start:
    type: string
    label: Report start date in `YYYY-MM-DD` format
  reporting_end:
    type: string?
    label: Report end date in `YYYY-MM-DD` format
    doc: Defaults to today

  prometheus_host:
    type: string?
    label: Prometheus server URL
    doc: The base URL of your Arvados cluster's Prometheus server, like `https://prometheus.arvados.example/`
  prometheus_apikey:
    type: string?
    label: Prometheus API token
  prometheus_user:
    type: string?
    label: Prometheus API username
  prometheus_password:
    type: string?
    label: Prometheus API password
  exclude:
    type: string?
    label: Exclude matching workflows
    doc: Specify a Python regular expression. Workflows whose name match the expression will be excluded from the report.
  include_workflow_steps:
    type: boolean?
    label: Include workflow steps?
    doc: If set, individual workflow steps will be reported alongside their parent workflows.

requirements:
  DockerRequirement:
    dockerFile: |
      FROM python:3.11-slim-bookworm
      RUN pip install --no-cache-dir "arvados-cluster-activity[prometheus]"
    dockerImageId: arvados/cluster-activity

  InlineJavascriptRequirement: {}

  arv:APIRequirement: {}

  ResourceRequirement:
    ramMin: 768

  EnvVarRequirement:
    envDef:
      PROMETHEUS_APIKEY: "$(inputs.prometheus_apikey || '')"
      PROMETHEUS_HOST: "$(inputs.prometheus_host || '')"
      PROMETHEUS_PASSWORD: "$(inputs.prometheus_password || '')"
      PROMETHEUS_USER: "$(inputs.prometheus_user || '')"
      REQUESTS_CA_BUNDLE: /etc/arvados/ca-certificates.crt

hints:
  cwltool:Secrets:
    secrets: [prometheus_apikey, prometheus_password]

arguments:
  - arv-cluster-activity
  - {prefix: '--start', valueFrom: $(inputs.reporting_start)}
  - {prefix: '--end', valueFrom: $(inputs.reporting_end)}
  - {prefix: '--exclude', valueFrom: $(inputs.exclude)}
  - {prefix: '--html-report-file', valueFrom: report.html}
  - {prefix: '--cost-report-file', valueFrom: cost.csv}
  - {prefix: '--include-workflow-steps', valueFrom: $(inputs.include_workflow_steps)}

outputs:
  report:
    type: File
    outputBinding:
      glob: report.html
