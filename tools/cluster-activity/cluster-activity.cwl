#!/usr/bin/env cwl-runner
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

cwlVersion: v1.2
class: CommandLineTool
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"

inputs:
  reporting_days: int
  prometheus_host: string
  prometheus_apikey: string?
  prometheus_user: string?
  prometheus_password: string?
  exclude: string?
  include_workflow_steps: boolean

requirements:
  DockerRequirement:
    dockerPull: 'arvados/cluster-activity:2.8.0.dev20240702194009'

  InitialWorkDirRequirement:
    listing:
      - entryname: prometheus.env
        entry: |
          PROMETHEUS_HOST=$(inputs.prometheus_host)
          PROMETHEUS_APIKEY=$(inputs.prometheus_apikey)
          PROMETHEUS_USER=$(inputs.prometheus_user)
          PROMETHEUS_PASSWORD=$(inputs.prometheus_password)

  arv:APIRequirement: {}

  ResourceRequirement:
    ramMin: 768

hints:
  cwltool:Secrets:
    secrets: [prometheus_apikey, prometheus_password]

arguments:
  - arv-cluster-activity
  - {prefix: '--prometheus-auth', valueFrom: prometheus.env}
  - {prefix: '--days', valueFrom: $(inputs.reporting_days)}
  - {prefix: '--exclude', valueFrom: $(inputs.exclude)}
  - {prefix: '--html-report-file', valueFrom: report.html}
  - {prefix: '--cost-report-file', valueFrom: cost.csv}
  - {prefix: '--include-workflow-steps', valueFrom: $(inputs.include_workflow_steps)}

outputs:
  report:
    type: File
    outputBinding:
      glob: report.html
