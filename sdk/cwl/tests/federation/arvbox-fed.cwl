# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

cwlVersion: v1.0
class: Workflow
$namespaces:
  arv: "http://arvados.org/cwl#"
  cwltool: "http://commonwl.org/cwltool#"
requirements:
  ScatterFeatureRequirement: {}
  StepInputExpressionRequirement: {}
  cwltool:LoadListingRequirement:
    loadListing: no_listing
  InlineJavascriptRequirement: {}
inputs:
  containers:
    type: string[]
    default: [fedbox1, fedbox2, fedbox3]
  arvbox_base: Directory
  in_acr: string?
  insecure:
    type: boolean
    default: true
outputs:
  arvados_api_token:
    type: string
    outputSource: setup-user/test_user_token
  arvados_api_hosts:
    type: string[]
    outputSource: start/container_host
  arvados_cluster_ids:
    type: string[]
    outputSource: start/cluster_id
  acr:
    type: string?
    outputSource: in_acr
  arvado_api_host_insecure:
    type: boolean
    outputSource: insecure
steps:
  mkdir:
    in:
      containers: containers
      arvbox_base: arvbox_base
    out: [arvbox_data]
    run: arvbox-mkdir.cwl
  start:
    in:
      container_name: containers
      arvbox_data: mkdir/arvbox_data
    out: [cluster_id, container_host, arvbox_data_out, superuser_token]
    scatter: [container_name, arvbox_data]
    scatterMethod: dotproduct
    run: arvbox-start.cwl
  fed-config:
    in:
      container_name: containers
      this_cluster_id: start/cluster_id
      cluster_ids: start/cluster_id
      cluster_hosts: start/container_host
      arvbox_data: start/arvbox_data_out
    out: []
    scatter: [container_name, this_cluster_id, arvbox_data]
    scatterMethod: dotproduct
    run: arvbox-fed-config.cwl
  setup-user:
    in:
      container_host: {source: start/container_host, valueFrom: "$(self[0])"}
      superuser_token: {source: start/superuser_token, valueFrom: "$(self[0])"}
    out: [test_user_uuid, test_user_token]
    run: arvbox-setup-user.cwl
