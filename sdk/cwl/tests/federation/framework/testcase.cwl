#!/usr/bin/env cwl-runner
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
hints:
  cwltool:Secrets:
    secrets: [arvados_api_token]
requirements:
  StepInputExpressionRequirement: {}
  InlineJavascriptRequirement: {}
  SubworkflowFeatureRequirement: {}
inputs:
  arvados_api_token: string
  arvado_api_host_insecure:
    type: boolean
    default: false
  arvados_api_hosts: string[]
  arvados_cluster_ids: string[]
  acr: string?
  wf: File
  obj: Any
  scrub_image: string
  scrub_collections: string[]
  runner_cluster: string?
outputs:
  out:
    type: Any
    outputSource: run-acr/out
  success:
    type: boolean
    outputSource: check-result/success
steps:
  dockerbuild:
    in:
      testcase: scrub_image
    out: [imagename]
    run: dockerbuild.cwl
  prepare:
    in:
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_host: {source: arvados_api_hosts, valueFrom: "$(self[0])"}
      arvados_cluster_ids: arvados_cluster_ids
      wf: wf
      obj: obj
      scrub_image: scrub_image
      scrub_collections: scrub_collections
    out: [done]
    run: prepare.cwl
  run-acr:
    in:
      prepare: prepare/done
      image-ready: dockerbuild/imagename
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_host: {source: arvados_api_hosts, valueFrom: "$(self[0])"}
      runner_cluster: runner_cluster
      acr: acr
      wf: wf
      obj: obj
    out: [out]
    run: run-acr.cwl
  check-result:
    in:
      acr-done: run-acr/out
      arvados_api_token: arvados_api_token
      arvado_api_host_insecure: arvado_api_host_insecure
      arvados_api_host: {source: arvados_api_hosts, valueFrom: "$(self[0])"}
      check_collections: scrub_collections
    out: [success]
    run: check-exist.cwl